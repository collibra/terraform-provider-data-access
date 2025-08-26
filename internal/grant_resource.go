package internal

import (
	"context"
	"fmt"
	"slices"
	"strings"

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
	"github.com/raito-io/sdk-go"
	raitoType "github.com/raito-io/sdk-go/types"
	"github.com/raito-io/sdk-go/types/models"

	types2 "github.com/raito-io/terraform-provider-raito/internal/types"
	"github.com/raito-io/terraform-provider-raito/internal/types/abac_expression"
	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*GrantResource)(nil)

type GrantResourceModel struct {
	// AccessProviderResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id                types.String         `tfsdk:"id"`
	Name              types.String         `tfsdk:"name"`
	Description       types.String         `tfsdk:"description"`
	State             types.String         `tfsdk:"state"`
	Who               types.Set            `tfsdk:"who"`
	Owners            types.Set            `tfsdk:"owners"`
	WhoAbacRule       jsontypes.Normalized `tfsdk:"who_abac_rule"`
	WhoLocked         types.Bool           `tfsdk:"who_locked"`
	InheritanceLocked types.Bool           `tfsdk:"inheritance_locked"`

	// GrantResourceModel properties.
	Category        types.String `tfsdk:"category"`
	DataSource      types.Set    `tfsdk:"data_source"`
	WhatDataObjects types.Set    `tfsdk:"what_data_objects"`
	WhatAbacRule    types.Object `tfsdk:"what_abac_rule"`
	WhatLocked      types.Bool   `tfsdk:"what_locked"`
}

func (m *GrantResourceModel) GetAccessProviderResourceModel() *AccessProviderResourceModel {
	return &AccessProviderResourceModel{
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

func (m *GrantResourceModel) SetAccessProviderResourceModel(ap *AccessProviderResourceModel) {
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

func (m *GrantResourceModel) ToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) diag.Diagnostics {
	diagnostics := m.GetAccessProviderResourceModel().ToAccessProviderInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	if !m.DataSource.IsNull() && !m.DataSource.IsUnknown() {
		dataSourceElements := m.DataSource.Elements()

		result.DataSources = make([]raitoType.AccessProviderDataSourceInput, 0, len(dataSourceElements))

		for _, dsElement := range dataSourceElements {
			dsAttributes := dsElement.(types.Object).Attributes()

			var apType *string

			if !dsAttributes["type"].(types.String).IsUnknown() {
				apType = dsAttributes["type"].(types.String).ValueStringPointer()
			}

			result.DataSources = append(result.DataSources, raitoType.AccessProviderDataSourceInput{
				DataSource: dsAttributes["data_source"].(types.String).ValueString(),
				Type:       apType,
			})
		}
	}

	result.Action = utils.Ptr(models.AccessProviderActionGrant)
	result.WhatType = utils.Ptr(raitoType.WhoAndWhatTypeStatic)

	if !m.WhatDataObjects.IsNull() && !m.WhatDataObjects.IsUnknown() {
		m.whatDoToApInput(result)
	} else if !m.WhatAbacRule.IsNull() {
		diagnostics.Append(m.abacWhatToAccessProviderInput(ctx, client, result)...)

		if diagnostics.HasError() {
			return diagnostics
		}
	}

	if m.WhatLocked.ValueBool() {
		result.Locks = append(result.Locks, raitoType.AccessProviderLockDataInput{
			LockKey: raitoType.AccessProviderLockWhatlock,
			Details: &raitoType.AccessProviderLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		})
	}

	if !m.Category.IsUnknown() {
		result.Category = m.Category.ValueStringPointer()
	}

	return diagnostics
}

func (m *GrantResourceModel) whatDoToApInput(result *raitoType.AccessProviderInput) {
	elements := m.WhatDataObjects.Elements()

	result.WhatDataObjects = make([]raitoType.AccessProviderWhatInputDO, 0, len(elements))

	for _, whatDataObject := range elements {
		whatDataObjectObject := whatDataObject.(types.Object)
		whatDataObjectAttributes := whatDataObjectObject.Attributes()

		fullname := whatDataObjectAttributes["fullname"].(types.String).ValueString()
		dataSource := whatDataObjectAttributes["data_source"].(types.String).ValueString()

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

		result.WhatDataObjects = append(result.WhatDataObjects, raitoType.AccessProviderWhatInputDO{
			DataObjectByName: []raitoType.AccessProviderWhatDoByNameInput{{
				Fullname:   fullname,
				Datasource: dataSource,
			},
			},
			Permissions:       permissions,
			GlobalPermissions: globalPermissions,
		})
	}
}

func (m *GrantResourceModel) FromAccessProvider(ctx context.Context, client *sdk.RaitoClient, ap *raitoType.AccessProvider) diag.Diagnostics {
	apResourceModel := m.GetAccessProviderResourceModel()
	diagnostics := apResourceModel.FromAccessProvider(ap)

	if diagnostics.HasError() {
		return diagnostics
	}

	m.SetAccessProviderResourceModel(apResourceModel)

	dataSourceValues := make([]attr.Value, 0, len(ap.SyncData))

	for i := range ap.SyncData {
		ds := &ap.SyncData[i]
		dsId := types.StringValue(ds.DataSource.Id)
		dsType := types.StringPointerValue(ds.AccessProviderType.Type)

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
			return diagnostics
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
		return diagnostics
	}

	m.DataSource = dataSources

	m.WhatLocked = types.BoolValue(slices.ContainsFunc(ap.Locks, func(l raitoType.AccessProviderLocksAccessProviderLockData) bool {
		return l.LockKey == raitoType.AccessProviderLockWhatlock
	}))

	if ap.WhatType == raitoType.WhoAndWhatTypeDynamic && ap.WhatAbacRule != nil {
		object, objectDiagnostics := m.abacWhatFromAccessProvider(ctx, client, ap)
		diagnostics.Append(objectDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		m.WhatAbacRule = object
	}

	m.Category = types.StringValue(ap.Category.Id)

	return diagnostics
}

func (m *GrantResourceModel) UpdateOwners(owners types.Set) {
	m.Owners = owners
}

func (m *GrantResourceModel) abacWhatToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) (diagnostics diag.Diagnostics) {
	attributes := m.WhatAbacRule.Attributes()

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

	scope := make([]string, 0)
	scopeSet := attributes["scope"].(types.Set)

	for _, scopeItem := range scopeSet.Elements() {
		scopeObject := scopeItem.(types.Object)
		scopeAttributes := scopeObject.Attributes()

		dataSourceId := scopeAttributes["data_source"].(types.String).ValueString()
		fullname := scopeAttributes["fullname"].(types.String).ValueString()

		id, err := client.DataObject().GetDataObjectIdByName(ctx, fullname, dataSourceId)
		if err != nil {
			diagnostics.AddError("Failed to get data object id", err.Error())

			return diagnostics
		}

		scope = append(scope, id)
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

	result.WhatType = utils.Ptr(raitoType.WhoAndWhatTypeDynamic)
	result.WhatAbacRule = &raitoType.WhatAbacRuleInput{
		DoTypes:           doTypes,
		Permissions:       permissions,
		GlobalPermissions: globalPermissions,
		Scope:             scope,
		Rule:              *abacInput,
	}

	return diagnostics
}

func (m *GrantResourceModel) abacWhatFromAccessProvider(ctx context.Context, client *sdk.RaitoClient, ap *raitoType.AccessProvider) (_ types.Object, diagnostics diag.Diagnostics) {
	scopeType := types.ObjectType{AttrTypes: map[string]attr.Type{"data_source": types.StringType, "fullname": types.StringType}}
	objectTypes := map[string]attr.Type{
		"do_types":           types.SetType{ElemType: types.StringType},
		"permissions":        types.SetType{ElemType: types.StringType},
		"global_permissions": types.SetType{ElemType: types.StringType},
		"scope":              types.SetType{ElemType: scopeType},
		"rule":               jsontypes.NormalizedType{},
	}

	permissions, pDiagnostics := utils.SliceToStringSet(ctx, ap.WhatAbacRule.Permissions)
	diagnostics.Append(pDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	globalPermissionList := utils.Map(ap.WhatAbacRule.GlobalPermissions, strings.ToUpper)
	globalPermissions, gpDiagnostics := utils.SliceToStringSet(ctx, globalPermissionList)

	diagnostics.Append(gpDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	doTypes, dtDiagnostics := utils.SliceToStringSet(ctx, ap.WhatAbacRule.DoTypes)
	diagnostics.Append(dtDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	abacRule := jsontypes.NewNormalizedPointerValue(ap.WhatAbacRule.RuleJson)

	var scopeItems []attr.Value //nolint:prealloc

	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for scopeItem := range client.AccessProvider().GetAccessProviderAbacWhatScope(cancelCtx, ap.Id) {
		if scopeItem.HasError() {
			diagnostics.AddError("Failed to load access provider abac scope", scopeItem.GetError().Error())

			return types.ObjectNull(objectTypes), diagnostics
		}

		scopeItems = append(scopeItems, types.ObjectValueMust(scopeType.AttrTypes, map[string]attr.Value{
			"fullname":    types.StringValue(scopeItem.MustGetItem().FullName),
			"data_source": types.StringValue(scopeItem.MustGetItem().DataSource.Id),
		}))
	}

	scope, scopeDiagnostics := types.SetValue(scopeType, scopeItems)
	diagnostics.Append(scopeDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	object, whatAbacDiagnostics := types.ObjectValue(objectTypes, map[string]attr.Value{
		"do_types":           doTypes,
		"permissions":        permissions,
		"global_permissions": globalPermissions,
		"rule":               abacRule,
		"scope":              scope,
	})

	diagnostics.Append(whatAbacDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	return object, diagnostics
}

type GrantResource struct {
	AccessProviderResource[GrantResourceModel, *GrantResourceModel]
}

func NewGrantResource() resource.Resource {
	return &GrantResource{
		AccessProviderResource[GrantResourceModel, *GrantResourceModel]{
			readHooks:         []ReadHook[GrantResourceModel, *GrantResourceModel]{readGrantWhatItems},
			validationHooks:   []ValidationHook[GrantResourceModel, *GrantResourceModel]{validateGrantWhatItems},
			planModifierHooks: []PlanModifierHook[GrantResourceModel, *GrantResourceModel]{grantModifyPlan},
		},
	}
}

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
	attributes["data_source"] = schema.SetNestedAttribute{
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
		Description:         "The ID of the data source of the grant",
		MarkdownDescription: "The ID of the data source of the grant",
		Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
	}
	attributes["what_data_objects"] = schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"fullname": schema.StringAttribute{
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "The full name of the data object in the data source",
					MarkdownDescription: "The full name of the data object in the data source",
				},
				"data_source": schema.StringAttribute{
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "The data source of the data object",
					MarkdownDescription: "The data source of the data object",
					Validators: []validator.String{
						stringvalidator.LengthAtLeast(3),
					},
				},
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
		MarkdownDescription: "The data object what items associated to the grant. When this is not set (nil), the what list will not be overridden. This is typically used when this should be managed from Raito Cloud.",
	}
	attributes["what_abac_rule"] = schema.SingleNestedAttribute{
		Attributes: map[string]schema.Attribute{
			"scope": schema.SetNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"fullname": schema.StringAttribute{
							Required:            true,
							Optional:            false,
							Computed:            false,
							Sensitive:           false,
							Description:         "The full name of the data object in the data source",
							MarkdownDescription: "The full name of the data object in the data source",
						},
						"data_source": schema.StringAttribute{
							Required:            true,
							Optional:            false,
							Computed:            false,
							Sensitive:           false,
							Description:         "The data source of the data object",
							MarkdownDescription: "The data source of the data object",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(3),
							},
						},
					},
					CustomType:    nil,
					Validators:    nil,
					PlanModifiers: nil,
				},
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "Scope of the defined abac rule",
				MarkdownDescription: "Scope of the defined abac rule",
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
		Description:         "Indicates whether it should lock the what. Should be set to true if what_data_objects or what_abac_rule is set.",
		MarkdownDescription: "Indicates whether it should lock the what. Should be set to true if what_data_objects or what_abac_rule is set.",
	}

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "Grant access control resource",
		MarkdownDescription: "The resource for representing a Raito [Grant](https://docs.raito.io/docs/cloud/access_management/grants) access control.",
		Version:             1,
	}
}

func readGrantWhatItems(ctx context.Context, client *sdk.RaitoClient, data *GrantResourceModel) (diagnostics diag.Diagnostics) {
	if !data.WhatDataObjects.IsNull() {
		cancelCtx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		whatItemsChannel := client.AccessProvider().GetAccessProviderWhatDataObjectList(cancelCtx, data.Id.ValueString())

		stateWhatItems := make([]attr.Value, 0)

		for whatItem := range whatItemsChannel {
			if whatItem.HasError() {
				diagnostics.AddError("Failed to get what data objects", whatItem.GetError().Error())

				return diagnostics
			}

			what := whatItem.GetItem()

			var id *string
			var dataSourceId *string

			if what.DataObject != nil {
				id = &what.DataObject.FullName
				dataSourceId = &what.DataObject.DataSource.Id
			} else {
				diagnostics.AddError("Invalid what data object", "Received data object is nil")

				continue
			}

			permissions := make([]attr.Value, 0, len(what.Permissions))
			for _, p := range what.Permissions {
				permissions = append(permissions, types.StringPointerValue(p))
			}

			globalPermissions := make([]attr.Value, 0, len(what.GlobalPermissions))
			for _, p := range what.GlobalPermissions {
				globalPermissions = append(globalPermissions, types.StringValue(strings.ToUpper(*p)))
			}

			stateWhatItems = append(stateWhatItems, types.ObjectValueMust(map[string]attr.Type{
				"fullname":    types.StringType,
				"data_source": types.StringType,
				"permissions": types.SetType{
					ElemType: types.StringType,
				},
				"global_permissions": types.SetType{
					ElemType: types.StringType,
				},
			}, map[string]attr.Value{
				"fullname":           types.StringPointerValue(id),
				"data_source":        types.StringPointerValue(dataSourceId),
				"permissions":        types.SetValueMust(types.StringType, permissions),
				"global_permissions": types.SetValueMust(types.StringType, globalPermissions),
			}))
		}

		whatDataObject, whatDiag := types.SetValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"fullname":    types.StringType,
				"data_source": types.StringType,
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
	if !data.WhatDataObjects.IsNull() && !data.WhatAbacRule.IsNull() {
		diagnostics.AddError("Cannot set both what_data_objects and what_abac_rule", "Grant Resource cannot have both what_data_objects and what_abac_rule")
	}

	if (!data.WhatDataObjects.IsNull() || !data.WhatAbacRule.IsNull()) && (!data.WhatLocked.IsNull() && !data.WhatLocked.ValueBool()) {
		diagnostics.AddError("What lock should be true", "What data objects or what abac rule is set, so what lock should be true")
	}

	return diagnostics
}

func grantModifyPlan(_ context.Context, data *GrantResourceModel) (_ *GrantResourceModel, diagnostics diag.Diagnostics) {
	if !data.WhatDataObjects.IsNull() || !data.WhatAbacRule.IsNull() {
		data.WhatLocked = types.BoolValue(true)
	} else if data.WhatLocked.IsUnknown() {
		data.WhatLocked = types.BoolValue(false)
	}

	return data, diagnostics
}
