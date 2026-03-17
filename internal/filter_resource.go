package internal

import (
	"context"
	"fmt"
	"slices"

	"github.com/collibra/data-access-go-sdk"
	dataAccessType "github.com/collibra/data-access-go-sdk/types"
	"github.com/collibra/terraform-provider-data-access/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*FilterResource)(nil)

//
// Model
//

type FilterResourceModel struct {
	// AccessControlResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`

	// FilterResourceModel properties
	Table       types.Object `tfsdk:"table"`
	WhatLocked  types.Bool   `tfsdk:"what_locked"`
	FilterRules types.Set    `tfsdk:"filter_rules"`
	Owners      types.Set    `tfsdk:"owners"`
}

func (f *FilterResourceModel) GetAccessControlResourceModel() *AccessControlResourceModel {
	return &AccessControlResourceModel{
		Id:          f.Id,
		Name:        f.Name,
		Description: f.Description,
		State:       f.State,
	}
}

func (f *FilterResourceModel) SetAccessControlResourceModel(ap *AccessControlResourceModel) {
	f.Id = ap.Id
	f.Name = ap.Name
	f.Description = ap.Description
	f.State = ap.State

	if !ap.Who.IsUnknown() && !ap.Who.IsNull() {
		filterRules := make([]attr.Value, 0, len(ap.Who.Elements()))

		for _, elem := range ap.Who.Elements() {
			whoObject := elem.(types.Object)

			attributes := whoObject.Attributes()
			if ac, found := attributes["access_control"]; found && !ac.IsNull() && !ac.IsUnknown() {
				filterRules = append(filterRules, ac)
			}
		}

		f.FilterRules = types.SetValueMust(types.StringType, filterRules)
	}
}

func (f *FilterResourceModel) UpdateOwners(owners types.Set) {
	f.Owners = owners
}

func (f *FilterResourceModel) GetOwners() (types.Set, bool) {
	return f.Owners, true
}

type FilterResource struct {
	AccessControlResource[FilterResourceModel, *FilterResourceModel]
}

func NewFilterResource() resource.Resource {
	return &FilterResource{
		AccessControlResource: AccessControlResource[FilterResourceModel, *FilterResourceModel]{
			readHooks: []ReadHook[FilterResourceModel, *FilterResourceModel]{
				filterTableToTerraform,
			},
			validationHooks: []ValidationHook[FilterResourceModel, *FilterResourceModel]{
				validateFilterWhatLock,
			},
			planModifierHooks: []PlanModifierHook[FilterResourceModel, *FilterResourceModel]{
				filterModifyPlan,
			},
		},
	}
}

//
// Schema
//

func (f *FilterResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_filter"
}

func (f *FilterResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := f.schema("filter", withAccessControlSchemaExcludeWho())
	attributes["owners"] = schema.SetAttribute{
		ElementType:         types.StringType,
		Required:            false,
		Optional:            true,
		Computed:            true,
		Sensitive:           false,
		Description:         "User id of the owners of this filter",
		MarkdownDescription: "User id of the owners of this filter",
		Validators: []validator.Set{
			setvalidator.ValueStringsAre(
				stringvalidator.LengthAtLeast(3),
			),
		},
		Default: nil,
	}

	attributes["table"] = schema.ObjectAttribute{
		AttributeTypes:      dataObjectReferenceTypeAttributeTypes,
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "The table that should be filtered",
		MarkdownDescription: "The table that should be filtered",
	}
	attributes["what_locked"] = schema.BoolAttribute{
		Required:            false,
		Optional:            true,
		Computed:            true,
		Sensitive:           false,
		Description:         "Indicates whether it should lock the what. Should be set to true if table is set.",
		MarkdownDescription: "Indicates whether it should lock the what. Should be set to true if table is set.",
	}
	attributes["filter_rules"] = schema.SetAttribute{
		ElementType:         types.StringType,
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "Set of filter rules ids that are applicable for this filter.",
		MarkdownDescription: "Set of filter rules ids that are applicable for this filter",
		Validators:          nil,
		PlanModifiers:       nil,
		Default:             nil,
	}

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "The filter access control resource",
		MarkdownDescription: "The resource for representing a Row-level Filter access control. This should be used in combination with a Filter Rule.",
		Version:             1,
	}
}

//
// Actions
//

// ToAccessControlInput converts the Terraform model for a filter to the Collibra model for AccessControlInput.
func (f *FilterResourceModel) ToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *dataAccessType.AccessControlInput) diag.Diagnostics {
	diagnostics := f.GetAccessControlResourceModel().ToAccessControlInput(ctx, client, result, WithToAccessControlLockOwners(!f.Owners.IsNull()))

	if diagnostics.HasError() {
		return diagnostics
	}

	result.Action = utils.Ptr(dataAccessType.AccessControlActionFilter)

	if !f.FilterRules.IsNull() && !f.FilterRules.IsUnknown() {
		var ruleIds []string

		fdDiag := f.FilterRules.ElementsAs(ctx, &ruleIds, false)
		if fdDiag.HasError() {
			diagnostics = append(diagnostics, fdDiag...)

			return diagnostics
		}

		for _, ruleId := range ruleIds {
			result.WhoItems = append(result.WhoItems, dataAccessType.WhoItemInput{
				AccessControl: &ruleId,
			})
		}

		result.Locks = append(result.Locks, dataAccessType.AccessControlLockDataInput{
			LockKey: dataAccessType.AccessControlLockInheritancelock,
			Details: &dataAccessType.AccessControlLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		})
	}

	if !f.Table.IsNull() && !f.Table.IsUnknown() {
		result.Locks = append(result.Locks, dataAccessType.AccessControlLockDataInput{
			LockKey: dataAccessType.AccessControlLockWhatlock,
			Details: &dataAccessType.AccessControlLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		})

		doType, doPath, dsId := dataObjectReferenceToComponents(f.Table.Attributes())
		fullName := dataAccessType.FullName{
			Type: doType,
			Path: doPath,
		}

		result.DataSources = []dataAccessType.AccessControlDataSourceInput{
			{
				DataSource: dsId,
			},
		}

		result.WhatDataObjects = []dataAccessType.AccessControlWhatInputDO{
			{
				DataObjectByName: []dataAccessType.AccessControlWhatDoByNameInput{
					{
						FullName:   fullName.ToDataObjectURI(),
						DataSource: dsId,
					},
				},
			},
		}
	} else if !f.WhatLocked.IsNull() && f.WhatLocked.ValueBool() {
		result.Locks = append(result.Locks, dataAccessType.AccessControlLockDataInput{
			LockKey: dataAccessType.AccessControlLockWhatlock,
			Details: &dataAccessType.AccessControlLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		})
	}

	return diagnostics
}

// FromAccessControl converts the Collibra model for AccessControl to the Terraform model for a filter.
func (f *FilterResourceModel) FromAccessControl(_ context.Context, _ *sdk.CollibraClient, input *dataAccessType.AccessControl) diag.Diagnostics {
	apResourceModel := f.GetAccessControlResourceModel()
	diagnostics := apResourceModel.FromAccessControl(input)

	if diagnostics.HasError() {
		return diagnostics
	}

	f.SetAccessControlResourceModel(apResourceModel)

	if len(input.SyncData) != 1 {
		diagnostics.AddError("Failed to get data source", fmt.Sprintf("Expected exactly one data source, got: %d.", len(input.SyncData)))

		return diagnostics
	}

	f.WhatLocked = types.BoolValue(slices.ContainsFunc(input.Locks, func(data dataAccessType.AccessControlLocksAccessControlLockData) bool {
		return data.LockKey == dataAccessType.AccessControlLockWhatlock
	}))

	return diagnostics
}

// filterTableToTerraform reads the table DataObject from Collibra and converts it to the Terraform model
func filterTableToTerraform(ctx context.Context, client *sdk.CollibraClient, data *FilterResourceModel) (diagnostics diag.Diagnostics) {
	if !data.Table.IsNull() {
		whatItems := client.AccessControl().GetAccessControlWhatDataObjectList(ctx, data.Id.ValueString())

		first := true

		for whatItem, err := range whatItems {
			if !first {
				diagnostics.AddError("Received multiple tables. Expect exactly one", "Filter resource only supports one table")

				return diagnostics
			}

			first = false

			if err != nil {
				diagnostics.AddError("Failed to get filter what data objects", err.Error())

				return diagnostics
			}

			table, diags := dataObjectToReference(&whatItem.DataObject.DataObject)
			diagnostics.Append(diags...)

			if diagnostics.HasError() {
				return diagnostics
			}

			data.Table = table
		}
	}

	return diagnostics
}

func validateFilterWhatLock(_ context.Context, data *FilterResourceModel) (diagnostics diag.Diagnostics) {
	if !data.Table.IsNull() && !data.WhatLocked.IsNull() && !data.WhatLocked.ValueBool() {
		diagnostics.AddError("What_locked should be true", "Table is set, but what_locked is set to false")

		return diagnostics
	}

	return diagnostics
}

func filterModifyPlan(_ context.Context, data *FilterResourceModel) (_ *FilterResourceModel, diagnostics diag.Diagnostics) {
	if !data.Table.IsNull() {
		data.WhatLocked = types.BoolValue(true)
	} else if data.WhatLocked.IsUnknown() {
		data.WhatLocked = types.BoolValue(false)
	}

	return data, diagnostics
}
