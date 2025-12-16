package internal

import (
	"context"
	"fmt"
	"slices"

	sdk "github.com/collibra/data-access-go-sdk"
	dataAccessType "github.com/collibra/data-access-go-sdk/types"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/collibra/data-access-terraform-provider/internal/utils"
)

var _ resource.Resource = (*MaskResource)(nil)

type MaskResourceModel struct {
	// AccessControlResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	State             types.String `tfsdk:"state"`
	Who               types.Set    `tfsdk:"who"`
	Owners            types.Set    `tfsdk:"owners"`
	WhoAbacRules      types.Set    `tfsdk:"who_abac_rules"`
	WhoLocked         types.Bool   `tfsdk:"who_locked"`
	InheritanceLocked types.Bool   `tfsdk:"inheritance_locked"`

	// MaskResourceModel properties.
	DataSources   types.Set  `tfsdk:"data_sources"`
	Columns       types.Set  `tfsdk:"columns"`
	WhatAbacRules types.Set  `tfsdk:"what_abac_rules"`
	WhatLocked    types.Bool `tfsdk:"what_locked"`
}

func (m *MaskResourceModel) GetAccessControlResourceModel() *AccessControlResourceModel {
	return &AccessControlResourceModel{
		Id:                m.Id,
		Name:              m.Name,
		Description:       m.Description,
		State:             m.State,
		Who:               m.Who,
		Owners:            m.Owners,
		WhoAbacRules:      m.WhoAbacRules,
		WhoLocked:         m.WhoLocked,
		InheritanceLocked: m.InheritanceLocked,
	}
}

func (m *MaskResourceModel) SetAccessControlResourceModel(ap *AccessControlResourceModel) {
	m.Id = ap.Id
	m.Name = ap.Name
	m.Description = ap.Description
	m.State = ap.State
	m.Who = ap.Who
	m.Owners = ap.Owners
	m.WhoAbacRules = ap.WhoAbacRules
	m.WhoLocked = ap.WhoLocked
	m.InheritanceLocked = ap.InheritanceLocked
}

func (m *MaskResourceModel) ToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *dataAccessType.AccessControlInput) diag.Diagnostics {
	diagnostics := m.GetAccessControlResourceModel().ToAccessControlInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	dataSourcesToAccessControlInput(m.DataSources, result)

	result.Action = utils.Ptr(dataAccessType.AccessControlActionMask)

	if !m.Columns.IsNull() && !m.Columns.IsUnknown() {
		elements := m.Columns.Elements()

		result.WhatDataObjects = make([]dataAccessType.AccessControlWhatInputDO, 0, len(elements))

		for _, whatDataObject := range elements {
			dataObject := whatDataObject.(types.Object)
			doType, doPath, dsId := dataObjectReferenceToComponents(dataObject.Attributes())
			fullName := dataAccessType.FullName{
				Type: doType,
				Path: doPath,
			}

			result.WhatDataObjects = append(result.WhatDataObjects, dataAccessType.AccessControlWhatInputDO{
				DataObjectByName: []dataAccessType.AccessControlWhatDoByNameInput{{
					FullName:   fullName.ToDataObjectURI(),
					DataSource: dsId,
				}},
			})
		}
	}

	if !m.WhatAbacRules.IsNull() {
		diagnostics.Append(m.abacWhatToAccessControlInput(ctx, client, result)...)

		if diagnostics.HasError() {
			return diagnostics
		}
	}

	if m.WhatLocked.ValueBool() {
		result.Locks = append(result.Locks, dataAccessType.AccessControlLockDataInput{
			LockKey: dataAccessType.AccessControlLockWhatlock,
			Details: &dataAccessType.AccessControlLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		})
	}

	return diagnostics
}

func (m *MaskResourceModel) FromAccessControl(ctx context.Context, client *sdk.CollibraClient, input *dataAccessType.AccessControl) diag.Diagnostics {
	apResourceModel := m.GetAccessControlResourceModel()
	diagnostics := apResourceModel.FromAccessControl(input)

	if diagnostics.HasError() {
		return diagnostics
	}

	m.SetAccessControlResourceModel(apResourceModel)

	if len(input.SyncData) != 1 {
		diagnostics.AddError("Failed to get data source", fmt.Sprintf("Expected exactly one data source, got: %d.", len(input.SyncData)))

		return diagnostics
	}

	defaultMaskType, err := client.DataSource().GetMaskingMetadata(ctx, input.SyncData[0].DataSource.Id)
	if err != nil {
		diagnostics.AddError("Failed to get default mask type", err.Error())

		return diagnostics
	}

	dataSources, d, done := dataSourcesFromAccessControl(input, diagnostics, defaultMaskType.DefaultMaskExternalName)
	if done {
		return d
	}

	m.DataSources = dataSources

	m.WhatLocked = types.BoolValue(slices.ContainsFunc(input.Locks, func(data dataAccessType.AccessControlLocksAccessControlLockData) bool {
		return data.LockKey == dataAccessType.AccessControlLockWhatlock
	}))

	if input.WhatAbacRules != nil {
		object, objectDiagnostics := m.abacWhatFromAccessControl(ctx, client, input)
		diagnostics.Append(objectDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		m.WhatAbacRules = object
	}

	return diagnostics
}

func (m *MaskResourceModel) UpdateOwners(owners types.Set) {
	m.Owners = owners
}

func (m *MaskResourceModel) abacWhatToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *dataAccessType.AccessControlInput) (diagnostics diag.Diagnostics) {
	result.WhatAbacRules = make([]*dataAccessType.WhatAbacRuleInput, 0, len(m.WhatAbacRules.Elements()))

	for _, abacRuleItem := range m.WhatAbacRules.Elements() {
		abacRuleObject := abacRuleItem.(types.Object)
		attributes := abacRuleObject.Attributes()

		scope := make([]string, 0)
		scopeSet := attributes["scope"].(types.Set)

		scope, d, done := abacWhatScopeToAccessControlInput(ctx, client, scopeSet, diagnostics, scope)
		if done {
			return d
		}

		abacInput, ruleDiag := abacRuleToGqlInput(attributes, "rule")
		if ruleDiag.HasError() {
			return ruleDiag
		}

		result.WhatAbacRules = append(result.WhatAbacRules, &dataAccessType.WhatAbacRuleInput{
			Scope:   scope,
			Rule:    *abacInput,
			Id:      getOptionalString(attributes, "id"),
			DoTypes: []string{"column"},
		})
	}

	return diagnostics
}

func (m *MaskResourceModel) abacWhatFromAccessControl(ctx context.Context, client *sdk.CollibraClient, ac *dataAccessType.AccessControl) (_ types.Set, diagnostics diag.Diagnostics) {
	whatAbacRuleList := make([]attr.Value, 0, len(ac.WhatAbacRules))

	scopeType := types.ObjectType{AttrTypes: dataObjectReferenceTypeAttributeTypes}
	whatAbacRuleType := map[string]attr.Type{
		"scope": types.SetType{ElemType: scopeType},
		"rule":  jsontypes.NormalizedType{},
		"id":    types.StringType,
	}
	whatAbacRulesType := types.ObjectType{AttrTypes: whatAbacRuleType}

	for _, rule := range ac.WhatAbacRules {
		abacRule := jsontypes.NewNormalizedPointerValue(rule.RuleJson)

		var scopeItems []attr.Value //nolint:prealloc

		for scopeItem, err := range client.AccessControl().GetAccessControlAbacWhatScope(ctx, ac.Id, rule.Id) {
			if err != nil {
				diagnostics.AddError("Failed to load access provider abac scope", err.Error())

				return types.SetNull(whatAbacRulesType), diagnostics
			}

			scopeItemValue, diags := dataObjectToReference(scopeItem, diagnostics)
			diagnostics.Append(diags...)

			if diagnostics.HasError() {
				return types.SetNull(whatAbacRulesType), diagnostics
			}

			scopeItems = append(scopeItems, scopeItemValue)
		}

		scope, scopeDiagnostics := types.SetValue(scopeType, scopeItems)
		diagnostics.Append(scopeDiagnostics...)

		if diagnostics.HasError() {
			return types.SetNull(whatAbacRulesType), diagnostics
		}

		whatAbacRuleList = append(whatAbacRuleList, types.ObjectValueMust(whatAbacRuleType, map[string]attr.Value{
			"rule":  abacRule,
			"scope": scope,
			"id":    types.StringValue(rule.Id),
		}))
	}

	whatAbacRules, whatAbacRulesDiag := types.SetValue(whatAbacRulesType, whatAbacRuleList)

	diagnostics.Append(whatAbacRulesDiag...)

	if diagnostics.HasError() {
		return types.SetNull(whatAbacRulesType), diagnostics
	}

	return whatAbacRules, diagnostics
}

type MaskResource struct {
	AccessControlResource[MaskResourceModel, *MaskResourceModel]
}

func NewMaskResource() resource.Resource {
	return &MaskResource{
		AccessControlResource: AccessControlResource[MaskResourceModel, *MaskResourceModel]{
			readHooks: []ReadHook[MaskResourceModel, *MaskResourceModel]{
				readMaskResourceColumns,
			},
			validationHooks: []ValidationHook[MaskResourceModel, *MaskResourceModel]{
				validateMaskWhatLock,
			},
			planModifierHooks: []PlanModifierHook[MaskResourceModel, *MaskResourceModel]{
				maskModifyPlan,
			},
		},
	}
}

func (m *MaskResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mask"
}

func (m *MaskResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := m.schema("mask")
	attributes["data_sources"] = schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"data_source": schema.StringAttribute{
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "The ID of the data source that this mask is applicable to",
					MarkdownDescription: "The ID of the data source that this mask is applicable to",
					Validators: []validator.String{
						stringvalidator.LengthAtLeast(3),
					},
				},
				"type": schema.StringAttribute{
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "The masking type to use for the mask in this data source",
					MarkdownDescription: "The masking type to use for the mask in this data source. Available types are defined by the data source.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
			},
		},
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The list of data sources that this mask is applicable to",
		MarkdownDescription: "The list of data sources that this mask is applicable to",
		Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
	}
	attributes["columns"] = schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: dataObjectReferenceTypeAttributes,
		},
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "The list of columns that should be included in the mask",
		MarkdownDescription: "The list of columns that should be included in the mask.",
	}

	attributes["what_abac_rules"] = schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Required:            false,
					Optional:            true,
					Computed:            true,
					Sensitive:           false,
					Description:         "The ID of the abac rule",
					MarkdownDescription: "The ID of the abac rule",
					Default:             nil,
				},
				"scope": schema.SetNestedAttribute{
					NestedObject: schema.NestedAttributeObject{
						Attributes: dataObjectReferenceTypeAttributes,
					},
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "Scope of the defined abac rule as a list of data objects",
					MarkdownDescription: "Scope of the defined abac rule as a list of data objects",
					Validators: []validator.Set{
						setvalidator.SizeAtLeast(1),
					},
				},
				"rule": schema.StringAttribute{
					CustomType:          jsontypes.NormalizedType{},
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "json representation of the abac rule",
					MarkdownDescription: "json representation of the abac rule",
					Default:             nil,
				},
			},
		},
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "The abac rules for defining the what of a make.",
		MarkdownDescription: "The abac rules for defining the what of a make.",
	}
	attributes["what_locked"] = schema.BoolAttribute{
		Required:            false,
		Optional:            true,
		Computed:            true,
		Sensitive:           false,
		Description:         "Indicates whether it should lock the what. Should be set to true if columns or what_abac_rule is set.",
		MarkdownDescription: "Indicates whether it should lock the what. Should be set to true if columns or what_abac_rule is set.",
	}

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "The mask access control resource",
		MarkdownDescription: "The resource for representing a Column Mask access control.",
		Version:             1,
	}
}

func readMaskResourceColumns(ctx context.Context, client *sdk.CollibraClient, data *MaskResourceModel) (diagnostics diag.Diagnostics) {
	if !data.Columns.IsNull() {
		stateWhatItems := make([]attr.Value, 0)

		for whatItem, err := range client.AccessControl().GetAccessControlWhatDataObjectList(ctx, data.Id.ValueString()) {
			if err != nil {
				diagnostics.AddError("Failed to get what data objects", err.Error())

				return diagnostics
			}

			if whatItem.DataObject != nil {
				whatItemValue, diags := dataObjectToReference(&whatItem.DataObject.DataObject, diagnostics)
				diagnostics.Append(diags...)

				stateWhatItems = append(stateWhatItems, whatItemValue)
			} else {
				diagnostics.AddError("Invalid what data object", "Received data object is nil")
			}
		}

		scopeType := types.ObjectType{AttrTypes: dataObjectReferenceTypeAttributeTypes}
		columnsObject, columnsDiag := types.SetValue(scopeType, stateWhatItems)

		diagnostics.Append(columnsDiag...)

		if diagnostics.HasError() {
			return diagnostics
		}

		data.Columns = columnsObject
	}

	return diagnostics
}

func validateMaskWhatLock(_ context.Context, data *MaskResourceModel) (diagnostics diag.Diagnostics) {
	if (!data.Columns.IsNull() || !data.WhatAbacRules.IsNull()) && (!data.WhatLocked.IsNull() && !data.WhatLocked.ValueBool()) {
		diagnostics.AddError("What lock should be true", "Columns or what abac rule should be set, so what lock should be true")
	}

	return diagnostics
}

func maskModifyPlan(_ context.Context, data *MaskResourceModel) (_ *MaskResourceModel, diagnostics diag.Diagnostics) {
	if !data.Columns.IsNull() || (!data.WhatAbacRules.IsNull() && len(data.WhatAbacRules.Elements()) > 0) {
		data.WhatLocked = types.BoolValue(true)
	} else if data.WhatLocked.IsUnknown() {
		data.WhatLocked = types.BoolValue(false)
	}

	return data, diagnostics
}
