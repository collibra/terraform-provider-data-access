package internal

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/collibra/data-access-go-sdk"
	dataAccessType "github.com/collibra/data-access-go-sdk/types"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	types2 "github.com/collibra/data-access-terraform-provider/internal/types"
	"github.com/collibra/data-access-terraform-provider/internal/utils"
)

//
// Model
//

var _ resource.Resource = (*GrantResource)(nil)

type GrantResourceModel struct {
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

	// GrantResourceModel properties.
	Category        types.String `tfsdk:"category"`
	DataSources     types.Set    `tfsdk:"data_sources"`
	WhatDataObjects types.Set    `tfsdk:"what_data_objects"`
	WhatAbacRules   types.Set    `tfsdk:"what_abac_rules"`
	WhatLocked      types.Bool   `tfsdk:"what_locked"`
}

func (m *GrantResourceModel) GetAccessControlResourceModel() *AccessControlResourceModel {
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

func (m *GrantResourceModel) SetAccessControlResourceModel(ac *AccessControlResourceModel) {
	m.Id = ac.Id
	m.Name = ac.Name
	m.Description = ac.Description
	m.State = ac.State
	m.Who = ac.Who
	m.Owners = ac.Owners
	m.WhoAbacRules = ac.WhoAbacRules
	m.WhoLocked = ac.WhoLocked
	m.InheritanceLocked = ac.InheritanceLocked
}

func (m *GrantResourceModel) UpdateOwners(owners types.Set) {
	m.Owners = owners
}

type GrantResource struct {
	AccessControlResource[GrantResourceModel, *GrantResourceModel]
}

func NewGrantResource() resource.Resource {
	return &GrantResource{
		AccessControlResource[GrantResourceModel, *GrantResourceModel]{
			readHooks:         []ReadHook[GrantResourceModel, *GrantResourceModel]{readGrantWhatItems},
			validationHooks:   []ValidationHook[GrantResourceModel, *GrantResourceModel]{validateGrantWhatItems},
			planModifierHooks: []PlanModifierHook[GrantResourceModel, *GrantResourceModel]{grantModifyPlan},
		},
	}
}

//
// Schema
//

func (g *GrantResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_grant"
}

func (g *GrantResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := g.schema("grant")
	attributes["category"] = schema.StringAttribute{
		Required:            false,
		Optional:            true,
		Computed:            true,
		Sensitive:           false,
		Description:         "The ID of the category of the grant",
		MarkdownDescription: "The ID of the category of the grant",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
			stringplanmodifier.RequiresReplace(),
		},
	}
	attributes["data_sources"] = schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"data_source": schema.StringAttribute{
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "The ID of the data source of the grant",
					MarkdownDescription: "The ID of the data source of the grant",
					Validators: []validator.String{
						stringvalidator.LengthAtLeast(3),
					},
				},
				"type": schema.StringAttribute{
					Required:            false,
					Optional:            true,
					Computed:            true,
					Sensitive:           false,
					Description:         "The implementation type of the grant for this data source",
					MarkdownDescription: "The implementation type of the grant for this data source",
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
		Description:         "The list of data sources that this grant is applicable to",
		MarkdownDescription: "The list of data sources that this grant is applicable to",
		Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
	}
	attributes["what_data_objects"] = schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"data_object": dataObjectReferenceType,
				"permissions": schema.SetAttribute{
					ElementType:         types.StringType,
					Required:            false,
					Optional:            true,
					Computed:            true,
					Sensitive:           false,
					Description:         "The set of permissions granted to the data object",
					MarkdownDescription: "The set of permissions granted to the data object",
					Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				},
				"global_permissions": schema.SetAttribute{
					ElementType:         types.StringType,
					Required:            false,
					Optional:            true,
					Computed:            true,
					Sensitive:           false,
					Description:         "The set of global permissions granted to the data object",
					MarkdownDescription: fmt.Sprintf("The set of global permissions granted to the data object. Allowed values are %v", types2.AllGlobalPermissions),
					Validators: []validator.Set{
						setvalidator.ValueStringsAre(
							stringvalidator.OneOf(types2.AllGlobalPermissions...),
						),
					},
					Default: setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{
						types.StringValue(types2.GlobalPermissionRead),
					})),
				},
			},
		},
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "The data object what items associated to the grant.",
		MarkdownDescription: "The data object what items associated to the grant. When this is not set (nil), the what list will not be overridden. This is typically used when this should be managed from Collibra Data Access.",
	}
	attributes["what_abac_rules"] = schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "A unique ID of the abac rule within this access control",
					MarkdownDescription: "A unique ID of the abac rule within this access control",
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
				"do_types": schema.SetAttribute{
					ElementType:         types.StringType,
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "Set of data object types associated to the abac rule",
					MarkdownDescription: "Set of data object types associated to the abac rule",
					Validators: []validator.Set{
						setvalidator.SizeAtLeast(1),
					},
				},
				"permissions": schema.SetAttribute{
					ElementType:         types.StringType,
					Required:            false,
					Optional:            true,
					Computed:            true,
					Sensitive:           false,
					Description:         "Set of permissions that should be granted on the matching data object",
					MarkdownDescription: "Set of permissions that should be granted on the matching data object",
					Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				},
				"global_permissions": schema.SetAttribute{
					ElementType:         types.StringType,
					Required:            false,
					Optional:            true,
					Computed:            true,
					Sensitive:           false,
					Description:         "Set of global permissions that should be granted on the matching data object",
					MarkdownDescription: fmt.Sprintf("Set of global permissions that should be granted on the matching data object. Allowed values are %v", types2.AllGlobalPermissions),
					Validators: []validator.Set{
						setvalidator.ValueStringsAre(
							stringvalidator.OneOf(types2.AllGlobalPermissions...),
						),
					},
					Default: setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{
						types.StringValue(types2.GlobalPermissionRead),
					})),
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
		Description:         "The abac rules for defining the what of a grant.",
		MarkdownDescription: "The abac rules for defining the what of a grant.",
	}
	attributes["what_locked"] = schema.BoolAttribute{
		Required:            false,
		Optional:            true,
		Computed:            true,
		Sensitive:           false,
		Description:         "Indicates whether it should lock the what. Should be set to true if what_data_objects or what_abac_rule is set.",
		MarkdownDescription: "Indicates whether it should lock the what. Should be set to true if what_data_objects or what_abac_rule is set.",
	}

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "Grant access control resource",
		MarkdownDescription: "The resource for representing a Collibra Data Access Grant access control.",
		Version:             1,
	}
}

//
// Actions
//

// ToAccessControlInput converts the Terraform model to an AccessControlInput for a grant.
func (m *GrantResourceModel) ToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *dataAccessType.AccessControlInput) diag.Diagnostics {
	diagnostics := m.GetAccessControlResourceModel().ToAccessControlInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	dataSourcesToAccessControlInput(m.DataSources, result)

	result.Action = utils.Ptr(dataAccessType.AccessControlActionGrant)

	if !m.WhatDataObjects.IsNull() && !m.WhatDataObjects.IsUnknown() {
		m.whatDoToApInput(result)
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

	if !m.Category.IsUnknown() {
		result.Category = m.Category.ValueStringPointer()
	}

	return diagnostics
}

// whatDoToApInput converts the WHAT DataObjects from the Terraform model to the AccessControlInput.
func (m *GrantResourceModel) whatDoToApInput(result *dataAccessType.AccessControlInput) {
	elements := m.WhatDataObjects.Elements()

	result.WhatDataObjects = make([]dataAccessType.AccessControlWhatInputDO, 0, len(elements))

	for _, whatDataObject := range elements {
		whatDataObjectObject := whatDataObject.(types.Object)
		whatDataObjectAttributes := whatDataObjectObject.Attributes()

		dataObject := whatDataObjectAttributes["data_object"].(types.Object)
		doType, doPath, dsId := dataObjectReferenceToComponents(dataObject.Attributes())
		fullName := dataAccessType.FullName{
			Type: doType,
			Path: doPath,
		}

		permissionSet := whatDataObjectAttributes["permissions"].(types.Set)
		permissions := make([]*string, 0, len(permissionSet.Elements()))

		for _, p := range permissionSet.Elements() {
			permission := p.(types.String)
			permissions = append(permissions, permission.ValueStringPointer())
		}

		globalPermissionSet := whatDataObjectAttributes["global_permissions"].(types.Set)
		globalPermissions := make([]*string, 0, len(globalPermissionSet.Elements()))

		for _, p := range globalPermissionSet.Elements() {
			permission := p.(types.String)
			globalPermissions = append(globalPermissions, permission.ValueStringPointer())
		}

		result.WhatDataObjects = append(result.WhatDataObjects, dataAccessType.AccessControlWhatInputDO{
			DataObjectByName: []dataAccessType.AccessControlWhatDoByNameInput{{
				FullName:   fullName.ToDataObjectURI(),
				DataSource: dsId,
			},
			},
			Permissions:       permissions,
			GlobalPermissions: globalPermissions,
		})
	}
}

// FromAccessControl converts the AccessControl from Collibra to the Terraform model for a grant.
func (m *GrantResourceModel) FromAccessControl(ctx context.Context, client *sdk.CollibraClient, ac *dataAccessType.AccessControl) diag.Diagnostics {
	apResourceModel := m.GetAccessControlResourceModel()
	diagnostics := apResourceModel.FromAccessControl(ac)

	if diagnostics.HasError() {
		return diagnostics
	}

	m.SetAccessControlResourceModel(apResourceModel)

	dataSources, d, done := dataSourcesFromAccessControl(ac, diagnostics, nil)
	if done {
		return d
	}

	m.DataSources = dataSources

	m.WhatLocked = types.BoolValue(slices.ContainsFunc(ac.Locks, func(l dataAccessType.AccessControlLocksAccessControlLockData) bool {
		return l.LockKey == dataAccessType.AccessControlLockWhatlock
	}))

	if ac.WhatAbacRules != nil {
		object, objectDiagnostics := m.abacWhatFromAccessControl(ctx, client, ac)
		diagnostics.Append(objectDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		m.WhatAbacRules = object
	}

	m.Category = types.StringValue(ac.Category.Id)

	return diagnostics
}

// dataSourcesFromAccessControl converts the data sources from the AccessControl Collibra model to the Terraform model.
func dataSourcesFromAccessControl(ac *dataAccessType.AccessControl, diagnostics diag.Diagnostics, defaultType *string) (basetypes.SetValue, diag.Diagnostics, bool) {
	dataSourceValues := make([]attr.Value, 0, len(ac.SyncData))

	for i := range ac.SyncData {
		ds := &ac.SyncData[i]
		dsId := types.StringValue(ds.DataSource.Id)

		dsType := types.StringPointerValue(defaultType)

		if ds.AccessControlType != nil && ds.AccessControlType.Type != nil {
			dsType = types.StringPointerValue(ds.AccessControlType.Type)
		}

		dataSource, diag := types.ObjectValue(map[string]attr.Type{
			"data_source": types.StringType,
			"type":        types.StringType,
		},
			map[string]attr.Value{
				"data_source": dsId,
				"type":        dsType,
			})

		diagnostics.Append(diag...)

		if diagnostics.HasError() {
			return basetypes.SetValue{}, diagnostics, true
		}

		dataSourceValues = append(dataSourceValues, dataSource)
	}

	dataSources, diag := types.SetValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"data_source": types.StringType,
			"type":        types.StringType,
		},
	}, dataSourceValues)

	diagnostics.Append(diag...)

	if diagnostics.HasError() {
		return basetypes.SetValue{}, diagnostics, true
	}

	return dataSources, nil, false
}

// abacWhatToAccessControlInput converts the WHAT ABAC rules from the Terraform model to the AccessControlInput.
func (m *GrantResourceModel) abacWhatToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *dataAccessType.AccessControlInput) (diagnostics diag.Diagnostics) {
	result.WhatAbacRules = make([]*dataAccessType.WhatAbacRuleInput, 0, len(m.WhatAbacRules.Elements()))

	for _, abacRuleItem := range m.WhatAbacRules.Elements() {
		abacRuleObject := abacRuleItem.(types.Object)
		attributes := abacRuleObject.Attributes()

		doTypes, doDiagnostics := utils.StringSetToSlice(ctx, attributes["do_types"].(types.Set))
		diagnostics.Append(doDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		permissions, permissionDiagnostics := utils.StringSetToSlice(ctx, attributes["permissions"].(types.Set))
		diagnostics.Append(permissionDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		globalPermissions, globalPermissionDiagnostics := utils.StringSetToSlice(ctx, attributes["global_permissions"].(types.Set))
		diagnostics.Append(globalPermissionDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		scopeSet := attributes["scope"].(types.Set)

		scope, d, done := abacWhatScopeToAccessControlInput(ctx, client, scopeSet)
		if done {
			return d
		}

		abacInput, ruleDiag := abacRuleToGqlInput(attributes, "rule")
		if ruleDiag.HasError() {
			return ruleDiag
		}

		result.WhatAbacRules = append(result.WhatAbacRules, &dataAccessType.WhatAbacRuleInput{
			DoTypes:           doTypes,
			Permissions:       permissions,
			GlobalPermissions: globalPermissions,
			Scope:             scope,
			Rule:              *abacInput,
			Id:                getOptionalString(attributes, "id"),
		})
	}

	return diagnostics
}

// abacWhatFromAccessControl converts the WHAT ABAC rules from the AccessControl Collibra model to the Terraform model.
func (m *GrantResourceModel) abacWhatFromAccessControl(ctx context.Context, client *sdk.CollibraClient, ac *dataAccessType.AccessControl) (_ types.Set, diagnostics diag.Diagnostics) {
	whatAbacRuleList := make([]attr.Value, 0, len(ac.WhatAbacRules))

	scopeType := types.ObjectType{AttrTypes: dataObjectReferenceTypeAttributeTypes}
	whatAbacRuleType := map[string]attr.Type{
		"do_types":           types.SetType{ElemType: types.StringType},
		"permissions":        types.SetType{ElemType: types.StringType},
		"global_permissions": types.SetType{ElemType: types.StringType},
		"scope":              types.SetType{ElemType: scopeType},
		"rule":               jsontypes.NormalizedType{},
		"id":                 types.StringType,
	}
	whatAbacRulesType := types.ObjectType{AttrTypes: whatAbacRuleType}

	for _, rule := range ac.WhatAbacRules {
		permissions, pDiagnostics := utils.SliceToStringSet(ctx, rule.Permissions)
		diagnostics.Append(pDiagnostics...)

		if diagnostics.HasError() {
			return types.SetNull(whatAbacRulesType), diagnostics
		}

		globalPermissionList := utils.Map(rule.GlobalPermissions, strings.ToUpper)
		globalPermissions, gpDiagnostics := utils.SliceToStringSet(ctx, globalPermissionList)

		diagnostics.Append(gpDiagnostics...)

		if diagnostics.HasError() {
			return types.SetNull(whatAbacRulesType), diagnostics
		}

		doTypes, dtDiagnostics := utils.SliceToStringSet(ctx, rule.DoTypes)
		diagnostics.Append(dtDiagnostics...)

		if diagnostics.HasError() {
			return types.SetNull(whatAbacRulesType), diagnostics
		}

		abacRule := jsontypes.NewNormalizedPointerValue(rule.RuleJson)

		var scopeItems []attr.Value //nolint:prealloc

		for scopeItem, err := range client.AccessControl().GetAccessControlAbacWhatScope(ctx, ac.Id, rule.Id) {
			if err != nil {
				diagnostics.AddError("Failed to load access provider abac scope", err.Error())

				return types.SetNull(whatAbacRulesType), diagnostics
			}

			scopeItemValue, diags := dataObjectToReference(scopeItem)
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
			"do_types":           doTypes,
			"permissions":        permissions,
			"global_permissions": globalPermissions,
			"rule":               abacRule,
			"scope":              scope,
			"id":                 types.StringValue(rule.Id),
		}))
	}

	whatAbacRules, whatAbacRulesDiag := types.SetValue(whatAbacRulesType, whatAbacRuleList)

	diagnostics.Append(whatAbacRulesDiag...)

	if diagnostics.HasError() {
		return types.SetNull(whatAbacRulesType), diagnostics
	}

	return whatAbacRules, diagnostics
}

// readGrantWhatItems reads the WHAT DataObjects from Collibra and converts it to the Terraform model (called as a hook)
func readGrantWhatItems(ctx context.Context, client *sdk.CollibraClient, data *GrantResourceModel) (diagnostics diag.Diagnostics) {
	if !data.WhatDataObjects.IsNull() {
		whatItems := client.AccessControl().GetAccessControlWhatDataObjectList(ctx, data.Id.ValueString())

		stateWhatItems := make([]attr.Value, 0)

		for whatItem, err := range whatItems {
			if err != nil {
				diagnostics.AddError("Failed to get what data objects", err.Error())

				return diagnostics
			}

			whatItemDataObject, diags := dataObjectToReference(&whatItem.DataObject.DataObject)
			diagnostics.Append(diags...)

			if diagnostics.HasError() {
				return diagnostics
			}

			permissions := make([]attr.Value, 0, len(whatItem.Permissions))
			for _, p := range whatItem.Permissions {
				permissions = append(permissions, types.StringPointerValue(p))
			}

			globalPermissions := make([]attr.Value, 0, len(whatItem.GlobalPermissions))
			for _, p := range whatItem.GlobalPermissions {
				globalPermissions = append(globalPermissions, types.StringValue(strings.ToUpper(*p)))
			}

			stateWhatItems = append(stateWhatItems, types.ObjectValueMust(map[string]attr.Type{
				"data_object": types.ObjectType{AttrTypes: dataObjectReferenceTypeAttributeTypes},
				"permissions": types.SetType{
					ElemType: types.StringType,
				},
				"global_permissions": types.SetType{
					ElemType: types.StringType,
				},
			}, map[string]attr.Value{
				"data_object":        whatItemDataObject,
				"permissions":        types.SetValueMust(types.StringType, permissions),
				"global_permissions": types.SetValueMust(types.StringType, globalPermissions),
			}))
		}

		whatDataObject, whatDiag := types.SetValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"data_object": types.ObjectType{AttrTypes: dataObjectReferenceTypeAttributeTypes},
				"permissions": types.SetType{
					ElemType: types.StringType,
				},
				"global_permissions": types.SetType{
					ElemType: types.StringType,
				},
			},
		}, stateWhatItems)

		diagnostics.Append(whatDiag...)

		if diagnostics.HasError() {
			return diagnostics
		}

		data.WhatDataObjects = whatDataObject
	}

	return diagnostics
}

func validateGrantWhatItems(_ context.Context, data *GrantResourceModel) (diagnostics diag.Diagnostics) {
	if (!data.WhatDataObjects.IsNull() || !data.WhatAbacRules.IsNull()) && (!data.WhatLocked.IsNull() && !data.WhatLocked.ValueBool()) {
		diagnostics.AddError("What lock should be true", "What data objects or what abac rule is set, so what lock should be true")
	}

	return diagnostics
}

func grantModifyPlan(_ context.Context, data *GrantResourceModel) (_ *GrantResourceModel, diagnostics diag.Diagnostics) {
	if !data.WhatDataObjects.IsNull() || (!data.WhatAbacRules.IsNull() && len(data.WhatAbacRules.Elements()) > 0) {
		data.WhatLocked = types.BoolValue(true)
	} else if data.WhatLocked.IsUnknown() {
		data.WhatLocked = types.BoolValue(false)
	}

	return data, diagnostics
}

//
// Helper functions
//

// dataObjectReferenceToId converts a data object reference from the Terraform model to a data object ID by querying Collibra.
func dataObjectReferenceToId(ctx context.Context, client *sdk.CollibraClient, dataObjectAttributes map[string]attr.Value) (string, error) {
	doType, path, dataSourceId := dataObjectReferenceToComponents(dataObjectAttributes)

	fullName := dataAccessType.FullName{
		Type: doType,
		Path: path,
	}

	ret, err := client.DataObject().GetDataObjectIdByName(ctx, fullName.ToDataObjectURI(), dataSourceId)
	if err != nil {
		return "", fmt.Errorf("get data object id for data object %s and data source %s: %w", fullName.ToDataObjectURI(), dataSourceId, err)
	}

	return ret, nil
}

// dataObjectReferenceToComponents extracts the components of a data object reference from the Terraform model (type, path and data source id).
func dataObjectReferenceToComponents(dataObjectAttributes map[string]attr.Value) (dataObjectType string, dataObjectPath []string, dataSourceId string) {
	dataObjectType = dataObjectAttributes["type"].(types.String).ValueString()
	path := dataObjectAttributes["path"].(types.List).Elements()
	dataSourceId = dataObjectAttributes["data_source"].(types.String).ValueString()

	dataObjectPath = make([]string, len(path))
	for i, doPathElement := range path {
		dataObjectPath[i] = doPathElement.(types.String).ValueString()
	}

	return
}

// dataObjectToReference converts a DataObject from Collibra to a data object reference object in the Terraform model.
func dataObjectToReference(dataObject *dataAccessType.DataObject) (obj basetypes.ObjectValue, diagnostics diag.Diagnostics) {
	fullName, err := dataAccessType.FromDataObjectURI(dataObject.FullName)
	if err != nil {
		diagnostics.AddError("Failed to parse data object full name", err.Error())

		return basetypes.ObjectValue{}, diagnostics
	}

	pathItems := make([]attr.Value, 0, len(fullName.Path))
	for _, pathItem := range fullName.Path {
		pathItems = append(pathItems, types.StringValue(pathItem))
	}

	pathValue, diag := types.ListValue(types.StringType, pathItems)
	if diag.HasError() {
		diagnostics = append(diagnostics, diag...)

		return basetypes.ObjectValue{}, diagnostics
	}

	return types.ObjectValueMust(dataObjectReferenceTypeAttributeTypes, map[string]attr.Value{
		"type":        types.StringValue(fullName.Type),
		"path":        pathValue,
		"data_source": types.StringValue(dataObject.DataSource.Id),
	}), nil
}

// abacWhatScopeToAccessControlInput converts the scope of an ABAC rule from the Terraform model to a list of data object IDs by querying Collibra.
func abacWhatScopeToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, scopeSet types.Set) (scope []string, diagnostics diag.Diagnostics, done bool) {
	scope = make([]string, 0, len(scopeSet.Elements()))

	for _, scopeItem := range scopeSet.Elements() {
		scopeObject := scopeItem.(types.Object)
		scopeAttributes := scopeObject.Attributes()

		id, err := dataObjectReferenceToId(ctx, client, scopeAttributes)
		if err != nil {
			diagnostics.AddError("Failed to get data object id", err.Error())

			return nil, diagnostics, true
		}

		scope = append(scope, id)
	}

	return scope, nil, false
}
