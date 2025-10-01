package internal

import (
	"context"
	"fmt"
	"slices"

	"github.com/collibra/access-governance-go-sdk"
	raitoType "github.com/collibra/access-governance-go-sdk/types"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*FilterResource)(nil)

type FilterResourceModel struct {
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

	// FilterResourceModel properties
	DataSource   types.String `tfsdk:"data_source"`
	Table        types.String `tfsdk:"table"`
	FilterPolicy types.String `tfsdk:"filter_policy"`
	WhatLocked   types.Bool   `tfsdk:"what_locked"`
}

func (f *FilterResourceModel) GetAccessProviderResourceModel() *AccessProviderResourceModel {
	return &AccessProviderResourceModel{
		Id:                f.Id,
		Name:              f.Name,
		Description:       f.Description,
		State:             f.State,
		Who:               f.Who,
		Owners:            f.Owners,
		WhoAbacRule:       f.WhoAbacRule,
		WhoLocked:         f.WhoLocked,
		InheritanceLocked: f.InheritanceLocked,
	}
}

func (f *FilterResourceModel) SetAccessProviderResourceModel(ap *AccessProviderResourceModel) {
	f.Id = ap.Id
	f.Name = ap.Name
	f.Description = ap.Description
	f.State = ap.State
	f.Who = ap.Who
	f.Owners = ap.Owners
	f.WhoAbacRule = ap.WhoAbacRule
	f.WhoLocked = ap.WhoLocked
	f.InheritanceLocked = ap.InheritanceLocked
}

func (f *FilterResourceModel) ToAccessProviderInput(ctx context.Context, client *sdk.CollibraClient, result *raitoType.AccessControlInput) diag.Diagnostics {
	diagnostics := f.GetAccessProviderResourceModel().ToAccessProviderInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	result.Action = utils.Ptr(raitoType.AccessControlActionFilter)

	if !f.DataSource.IsNull() && !f.DataSource.IsUnknown() {
		result.DataSources = []raitoType.AccessControlDataSourceInput{
			{
				DataSource: f.DataSource.ValueString(),
			},
		}
	}

	result.PolicyRule = f.FilterPolicy.ValueStringPointer()

	if !f.Table.IsNull() && !f.Table.IsUnknown() {
		result.Locks = append(result.Locks, raitoType.AccessControlLockDataInput{
			LockKey: raitoType.AccessControlLockWhatlock,
			Details: &raitoType.AccessControlLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		})

		result.WhatDataObjects = []raitoType.AccessControlWhatInputDO{
			{
				DataObjectByName: []raitoType.AccessControlWhatDoByNameInput{
					{
						FullName:   f.Table.ValueString(),
						DataSource: f.DataSource.ValueString(),
					},
				},
			},
		}
	} else if !f.WhatLocked.IsNull() && f.WhatLocked.ValueBool() {
		result.Locks = append(result.Locks, raitoType.AccessControlLockDataInput{
			LockKey: raitoType.AccessControlLockWhatlock,
			Details: &raitoType.AccessControlLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		})
	}

	return diagnostics
}

func (f *FilterResourceModel) FromAccessProvider(_ context.Context, _ *sdk.CollibraClient, input *raitoType.AccessControl) diag.Diagnostics {
	apResourceModel := f.GetAccessProviderResourceModel()
	diagnostics := apResourceModel.FromAccessProvider(input)

	if diagnostics.HasError() {
		return diagnostics
	}

	f.SetAccessProviderResourceModel(apResourceModel)

	if len(input.SyncData) != 1 {
		diagnostics.AddError("Failed to get data source", fmt.Sprintf("Expected exactly one data source, got: %d.", len(input.SyncData)))

		return diagnostics
	}

	f.DataSource = types.StringValue(input.SyncData[0].DataSource.Id)
	f.FilterPolicy = types.StringPointerValue(input.PolicyRule)
	f.WhatLocked = types.BoolValue(slices.ContainsFunc(input.Locks, func(data raitoType.AccessControlLocksAccessControlLockData) bool {
		return data.LockKey == raitoType.AccessControlLockWhatlock
	}))

	return diagnostics
}

func (f *FilterResourceModel) UpdateOwners(owners types.Set) {
	f.Owners = owners
}

type FilterResource struct {
	AccessProviderResource[FilterResourceModel, *FilterResourceModel]
}

func NewFilterResource() resource.Resource {
	return &FilterResource{
		AccessProviderResource: AccessProviderResource[FilterResourceModel, *FilterResourceModel]{
			readHooks: []ReadHook[FilterResourceModel, *FilterResourceModel]{
				readFilterResourceTable,
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

func (f *FilterResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_filter"
}

func (f *FilterResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := f.schema("filter")
	attributes["data_source"] = schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The ID of the data source of the filter",
		MarkdownDescription: "The ID of the data source of the filter",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(3),
		},
	}
	attributes["table"] = schema.StringAttribute{
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "The full name of the table that should be filtered",
		MarkdownDescription: "The full name of the table that should be filtered",
	}
	attributes["what_locked"] = schema.BoolAttribute{
		Required:            false,
		Optional:            true,
		Computed:            true,
		Sensitive:           false,
		Description:         "Indicates whether it should lock the what. Should be set to true if table is set.",
		MarkdownDescription: "Indicates whether it should lock the what. Should be set to true if table is set.",
	}
	attributes["filter_policy"] = schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The filter policy that defines how the data is filtered. The policy syntax is defined by the data source.",
		MarkdownDescription: "The filter policy that defines how the data is filtered. The policy syntax is defined by the data source.",
	}

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "The filter access control resource",
		MarkdownDescription: "The resource for representing a Raito [Row-level Filter](https://docs.raito.io/docs/cloud/access_management/row_filters) access control.",
		Version:             1,
	}
}

func readFilterResourceTable(ctx context.Context, client *sdk.CollibraClient, data *FilterResourceModel) (diagnostics diag.Diagnostics) {
	if !data.Table.IsNull() {
		cancelCtx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		whatItems := client.AccessControl().GetAccessControlWhatDataObjectList(cancelCtx, data.Id.ValueString())

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

			data.Table = types.StringValue(whatItem.DataObject.FullName)
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
