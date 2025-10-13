package internal

import (
	"context"
	"fmt"
	"slices"

	"github.com/collibra/access-governance-go-sdk"
	accessGovernanceType "github.com/collibra/access-governance-go-sdk/types"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/collibra/access-governance-terraform-provider/internal/types/abac_expression"
	"github.com/collibra/access-governance-terraform-provider/internal/utils"
)

var _ resource.Resource = (*MaskResource)(nil)

type MaskResourceModel struct {
	// AccessControlResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id                types.String         `tfsdk:"id"`
	Name              types.String         `tfsdk:"name"`
	Description       types.String         `tfsdk:"description"`
	State             types.String         `tfsdk:"state"`
	Who               types.Set            `tfsdk:"who"`
	Owners            types.Set            `tfsdk:"owners"`
	WhoAbacRule       jsontypes.Normalized `tfsdk:"who_abac_rule"`
	WhoLocked         types.Bool           `tfsdk:"who_locked"`
	InheritanceLocked types.Bool           `tfsdk:"inheritance_locked"`

	// MaskResourceModel properties.
	Type         types.String `tfsdk:"type"`
	DataSource   types.String `tfsdk:"data_source"`
	Columns      types.Set    `tfsdk:"columns"`
	WhatAbacRule types.Object `tfsdk:"what_abac_rule"`
	WhatLocked   types.Bool   `tfsdk:"what_locked"`
}

func (m *MaskResourceModel) GetAccessControlResourceModel() *AccessControlResourceModel {
	return &AccessControlResourceModel{
		Id:                m.Id,
		Name:              m.Name,
		Description:       m.Description,
		State:             m.State,
		Who:               m.Who,
		Owners:            m.Owners,
		WhoAbacRule:       m.WhoAbacRule,
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
	m.WhoAbacRule = ap.WhoAbacRule
	m.WhoLocked = ap.WhoLocked
	m.InheritanceLocked = ap.InheritanceLocked
}

func (m *MaskResourceModel) ToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *accessGovernanceType.AccessControlInput) diag.Diagnostics {
	diagnostics := m.GetAccessControlResourceModel().ToAccessControlInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	if !m.DataSource.IsNull() && !m.DataSource.IsUnknown() {
		var apType *string
		if !m.Type.IsUnknown() {
			apType = m.Type.ValueStringPointer()
		}

		result.DataSources = []accessGovernanceType.AccessControlDataSourceInput{
			{
				DataSource: m.DataSource.ValueString(),
				Type:       apType,
			},
		}
	}

	result.Action = utils.Ptr(accessGovernanceType.AccessControlActionMask)

	if !m.Columns.IsNull() && !m.Columns.IsUnknown() {
		elements := m.Columns.Elements()

		result.WhatDataObjects = make([]accessGovernanceType.AccessControlWhatInputDO, 0, len(elements))

		// Assume that currently only 1 dataSource is provided
		dataSource := result.DataSources[0].DataSource

		for _, whatDataObject := range elements {
			columnName := whatDataObject.(types.String).ValueString()

			result.WhatDataObjects = append(result.WhatDataObjects, accessGovernanceType.AccessControlWhatInputDO{
				DataObjectByName: []accessGovernanceType.AccessControlWhatDoByNameInput{{
					FullName:   columnName,
					DataSource: dataSource,
				}},
			})
		}
	} else if !m.WhatAbacRule.IsNull() {
		diagnostics.Append(m.abacWhatToAccessControlInput(ctx, client, result)...)

		if diagnostics.HasError() {
			return diagnostics
		}
	}

	if m.WhatLocked.ValueBool() {
		result.Locks = append(result.Locks, accessGovernanceType.AccessControlLockDataInput{
			LockKey: accessGovernanceType.AccessControlLockWhatlock,
			Details: &accessGovernanceType.AccessControlLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		})
	}

	return diagnostics
}

func (m *MaskResourceModel) FromAccessControl(ctx context.Context, client *sdk.CollibraClient, input *accessGovernanceType.AccessControl) diag.Diagnostics {
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

	m.DataSource = types.StringValue(input.SyncData[0].DataSource.Id)
	m.WhatLocked = types.BoolValue(slices.ContainsFunc(input.Locks, func(data accessGovernanceType.AccessControlLocksAccessControlLockData) bool {
		return data.LockKey == accessGovernanceType.AccessControlLockWhatlock
	}))

	if input.SyncData[0].AccessControlType == nil || input.SyncData[0].AccessControlType.Type == nil {
		maskType, err := client.DataSource().GetMaskingMetadata(ctx, input.SyncData[0].DataSource.Id)
		if err != nil {
			diagnostics.AddError("Failed to get default mask type", err.Error())

			return diagnostics
		}

		m.Type = types.StringPointerValue(maskType.DefaultMaskExternalName)
	} else {
		m.Type = types.StringPointerValue(input.SyncData[0].AccessControlType.Type)
	}

	if input.WhatType == accessGovernanceType.WhoAndWhatTypeDynamic && input.WhatAbacRule != nil {
		object, objectDiagnostics := m.abacWhatFromAccessControl(ctx, client, input)
		diagnostics.Append(objectDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		m.WhatAbacRule = object
	}

	return diagnostics
}

func (m *MaskResourceModel) UpdateOwners(owners types.Set) {
	m.Owners = owners
}

func (m *MaskResourceModel) abacWhatToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *accessGovernanceType.AccessControlInput) (diagnostics diag.Diagnostics) {
	attributes := m.WhatAbacRule.Attributes()

	scopeAttr := attributes["scope"]

	scope := make([]string, 0)

	if !scopeAttr.IsNull() && !scopeAttr.IsUnknown() {
		scopeFullnameItems, scopeDiagnostics := utils.StringSetToSlice(ctx, attributes["scope"].(types.Set))
		diagnostics.Append(scopeDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		// Assume that currently only 1 dataSource is provided
		dataSource := result.DataSources[0].DataSource

		for _, scopeFullnameItem := range scopeFullnameItems {
			id, err := client.DataObject().GetDataObjectIdByName(ctx, scopeFullnameItem, dataSource)
			if err != nil {
				diagnostics.AddError("Failed to get data object id", err.Error())

				return diagnostics
			}

			scope = append(scope, id)
		}
	}

	jsonRule := attributes["rule"].(jsontypes.Normalized)

	var abacRule abac_expression.BinaryExpression
	diagnostics.Append(jsonRule.Unmarshal(&abacRule)...)

	if diagnostics.HasError() {
		return diagnostics
	}

	abacInput, err := abacRule.ToGqlInput()
	if err != nil {
		diagnostics.AddError("Failed to convert abac rule to gql input", err.Error())

		return diagnostics
	}

	result.WhatType = utils.Ptr(accessGovernanceType.WhoAndWhatTypeDynamic)
	result.WhatAbacRule = &accessGovernanceType.WhatAbacRuleInput{
		DoTypes: []string{"column"},
		Scope:   scope,
		Rule:    *abacInput,
	}

	return diagnostics
}

func (m *MaskResourceModel) abacWhatFromAccessControl(ctx context.Context, client *sdk.CollibraClient, ap *accessGovernanceType.AccessControl) (_ types.Object, diagnostics diag.Diagnostics) {
	objectTypes := map[string]attr.Type{
		"scope": types.SetType{ElemType: types.StringType},
		"rule":  jsontypes.NormalizedType{},
	}

	abacRule := jsontypes.NewNormalizedPointerValue(ap.WhatAbacRule.RuleJson)

	var scopeItems []attr.Value //nolint:prealloc

	for scopeItem, err := range client.AccessControl().GetAccessControlAbacWhatScope(ctx, ap.Id) {
		if err != nil {
			diagnostics.AddError("Failed to load access provider abac scope", err.Error())

			return types.ObjectNull(objectTypes), diagnostics
		}

		scopeItems = append(scopeItems, types.StringValue(scopeItem.FullName))
	}

	scope, scopeDiagnostics := types.SetValue(types.StringType, scopeItems)
	diagnostics.Append(scopeDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	object, whatAbacDiagnostics := types.ObjectValue(objectTypes, map[string]attr.Value{
		"rule":  abacRule,
		"scope": scope,
	})

	diagnostics.Append(whatAbacDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	return object, diagnostics
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
	attributes["type"] = schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The masking method",
		MarkdownDescription: "The masking method, which defines how the data is masked. Available types are defined by the data source.",
	}
	attributes["data_source"] = schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The ID of the data source of the mask",
		MarkdownDescription: "The ID of the data source of the mask",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(3),
		},
	}
	attributes["columns"] = schema.SetAttribute{
		ElementType:         types.StringType,
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "The full name of columns that should be included in the mask",
		MarkdownDescription: "The full name of columns that should be included in the mask. Items are managed by Collibra Access Governance if columns is not set (nil).",
	}

	attributes["what_abac_rule"] = schema.SingleNestedAttribute{
		Attributes: map[string]schema.Attribute{
			"scope": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Scope of the defined abac rule",
				MarkdownDescription: "Scope of the defined abac rule",
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
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "What data object defined by abac rule. Cannot be set when what_data_objects is set.",
		MarkdownDescription: "What data object defined by abac rule. Cannot be set when what_data_objects is set.",
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
		MarkdownDescription: "The resource for representing a [Column Mask](https://docs.raito.io/docs/cloud/access_management/masks) access control.",
		Version:             1,
	}
}

func readMaskResourceColumns(ctx context.Context, client *sdk.CollibraClient, data *MaskResourceModel) (diagnostics diag.Diagnostics) {
	if !data.Columns.IsNull() {
		stateWhatItems := make([]attr.Value, 0)

		for whatItem, err := range client.AccessControl().GetAccessControlWhatDataObjectList(ctx, data.Id.ValueString()) {
			if err != nil {
				diagnostics.AddError("Fauled to get what data objects", err.Error())

				return diagnostics
			}

			if whatItem.DataObject != nil {
				stateWhatItems = append(stateWhatItems, types.StringValue(whatItem.DataObject.FullName))
			} else {
				diagnostics.AddError("Invalid what data object", "Received data object is nil")
			}
		}

		columnsObject, columnsDiag := types.SetValue(types.StringType, stateWhatItems)

		diagnostics.Append(columnsDiag...)

		if diagnostics.HasError() {
			return diagnostics
		}

		data.Columns = columnsObject
	}

	return diagnostics
}

func validateMaskWhatLock(_ context.Context, data *MaskResourceModel) (diagnostics diag.Diagnostics) {
	if (!data.Columns.IsNull() || !data.WhatAbacRule.IsNull()) && (!data.WhatLocked.IsNull() && !data.WhatLocked.ValueBool()) {
		diagnostics.AddError("What lock should be true", "Columns or what abac rule should be set, so what lock should be true")
	}

	return diagnostics
}

func maskModifyPlan(_ context.Context, data *MaskResourceModel) (_ *MaskResourceModel, diagnostics diag.Diagnostics) {
	if !data.Columns.IsNull() || !data.WhatAbacRule.IsNull() {
		data.WhatLocked = types.BoolValue(true)
	} else if data.WhatLocked.IsUnknown() {
		data.WhatLocked = types.BoolValue(false)
	}

	return data, diagnostics
}
