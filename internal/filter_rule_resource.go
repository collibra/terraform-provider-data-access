package internal

import (
	"context"

	sdk "github.com/collibra/data-access-go-sdk"
	dataAccessType "github.com/collibra/data-access-go-sdk/types"
	"github.com/collibra/terraform-provider-data-access/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//
// Model
//

type FilterRuleResourceModel struct {
	// AccessControlResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	State             types.String `tfsdk:"state"`
	Who               types.Set    `tfsdk:"who"`
	WhoAbacRules      types.Set    `tfsdk:"who_abac_rules"`
	WhoLocked         types.Bool   `tfsdk:"who_locked"`
	InheritanceLocked types.Bool   `tfsdk:"inheritance_locked"`

	// GrantResourceModel properties.
	FilterPolicy types.String `tfsdk:"filter_policy"`
}

func (f *FilterRuleResourceModel) GetAccessControlResourceModel() *AccessControlResourceModel {
	return &AccessControlResourceModel{
		Id:                f.Id,
		Name:              f.Name,
		Description:       f.Description,
		State:             f.State,
		Who:               f.Who,
		WhoAbacRules:      f.WhoAbacRules,
		WhoLocked:         f.WhoLocked,
		InheritanceLocked: f.InheritanceLocked,
	}
}

func (f *FilterRuleResourceModel) SetAccessControlResourceModel(m *AccessControlResourceModel) {
	f.Id = m.Id
	f.Name = m.Name
	f.Description = m.Description
	f.State = m.State
	f.Who = m.Who
	f.WhoAbacRules = m.WhoAbacRules
	f.WhoLocked = m.WhoLocked
	f.InheritanceLocked = m.InheritanceLocked
}

func (f *FilterRuleResourceModel) UpdateOwners(_ types.Set) {
	// Do nothing no owners
}

func (f *FilterRuleResourceModel) GetOwners() (types.Set, bool) {
	return types.Set{}, false
}

type FilterRuleResource struct {
	AccessControlResource[FilterRuleResourceModel, *FilterRuleResourceModel]
}

func NewFilterRuleResource() resource.Resource {
	return &FilterRuleResource{
		AccessControlResource: AccessControlResource[FilterRuleResourceModel, *FilterRuleResourceModel]{},
	}
}

// Schema

func (f *FilterRuleResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_filter_rule"
}

func (f *FilterRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := f.schema("filter_rule")
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
		Description:         "The filter rule access control resource",
		MarkdownDescription: "The resource for representing a Row-level Filter Rule access control. This should be used in combination with a Filter.",
		Version:             1,
	}
}

//
// Actions
//

func (f *FilterRuleResourceModel) ToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *dataAccessType.AccessControlInput) diag.Diagnostics {
	diagnostics := f.GetAccessControlResourceModel().ToAccessControlInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	result.Action = utils.Ptr(dataAccessType.AccessControlActionFilterrule)
	result.PolicyRule = f.FilterPolicy.ValueStringPointer()

	return diagnostics
}

func (f *FilterRuleResourceModel) FromAccessControl(_ context.Context, _ *sdk.CollibraClient, input *dataAccessType.AccessControl) diag.Diagnostics {
	apResourceModel := f.GetAccessControlResourceModel()

	diagnostics := apResourceModel.FromAccessControl(input)
	if diagnostics.HasError() {
		return diagnostics
	}

	f.SetAccessControlResourceModel(apResourceModel)

	f.FilterPolicy = types.StringPointerValue(input.PolicyRule)

	return diagnostics
}
