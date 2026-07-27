package internal

import (
	"context"

	sdk "github.com/collibra/data-access-go-sdk"
	dataAccessType "github.com/collibra/data-access-go-sdk/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//
// Model
//

type GroupResourceModel struct {
	// AccessControlResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	State             types.String `tfsdk:"state"`
	Who               types.Set    `tfsdk:"who"`
	WhoAbacRules      types.Set    `tfsdk:"who_abac_rules"`
	WhoLocked         types.Bool   `tfsdk:"who_locked"`
	InheritanceLocked types.Bool   `tfsdk:"inheritance_locked"`

	// GroupResourceModel properties
	DataSources types.Set `tfsdk:"data_sources"`
	Owners      types.Set `tfsdk:"owners"`
}

func (m *GroupResourceModel) GetAccessControlResourceModel() *AccessControlResourceModel {
	return &AccessControlResourceModel{
		Id:                m.Id,
		Name:              m.Name,
		Description:       m.Description,
		State:             m.State,
		Who:               m.Who,
		WhoAbacRules:      m.WhoAbacRules,
		WhoLocked:         m.WhoLocked,
		InheritanceLocked: m.InheritanceLocked,
	}
}

func (m *GroupResourceModel) SetAccessControlResourceModel(ac *AccessControlResourceModel) {
	m.Id = ac.Id
	m.Name = ac.Name
	m.Description = ac.Description
	m.State = ac.State
	m.Who = ac.Who
	m.WhoAbacRules = ac.WhoAbacRules
	m.WhoLocked = ac.WhoLocked
	m.InheritanceLocked = ac.InheritanceLocked
}

func (m *GroupResourceModel) UpdateOwners(owners types.Set) {
	m.Owners = owners
}

func (m *GroupResourceModel) GetOwners() (types.Set, bool) {
	return m.Owners, true
}

type GroupResource struct {
	AccessControlResource[GroupResourceModel, *GroupResourceModel]
}

func NewGroupResource() resource.Resource {
	return &GroupResource{
		AccessControlResource[GroupResourceModel, *GroupResourceModel]{
			readHooks:         []ReadHook[GroupResourceModel, *GroupResourceModel]{},
			validationHooks:   []ValidationHook[GroupResourceModel, *GroupResourceModel]{},
			planModifierHooks: []PlanModifierHook[GroupResourceModel, *GroupResourceModel]{},
		},
	}
}

func (g GroupResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_group"
}

func (g GroupResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := g.schema("group")

	attributes["owners"] = schema.SetAttribute{
		ElementType:         types.StringType,
		Required:            false,
		Optional:            true,
		Computed:            true,
		Sensitive:           false,
		Description:         "The user IDs of the owners of the role.",
		MarkdownDescription: "The user IDs of the owners of the role.",
		Validators: []validator.Set{
			setvalidator.ValueStringsAre(
				stringvalidator.LengthAtLeast(3),
			),
		},
		Default: nil,
	}
	attributes["data_sources"] = schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"data_source": schema.StringAttribute{
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "The ID of the data source of the role.",
					MarkdownDescription: "The ID of the data source of the role.",
					Validators: []validator.String{
						stringvalidator.LengthAtLeast(3),
					},
				},
				"type": schema.StringAttribute{
					Required:            false,
					Optional:            true,
					Computed:            true,
					Sensitive:           false,
					Description:         "The implementation type of the role for this data source.",
					MarkdownDescription: "The implementation type of the role for this data source.",
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
		Description:         "The data sources that the role applies to.",
		MarkdownDescription: "The data sources that the role applies to. See the nested schema below.",
		Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
	}

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "The resource for representing a group in Collibra Data Access.",
		MarkdownDescription: "The resource for representing a group in Collibra Data Access.",
		Version:             1,
	}
}

//
// Actions
//

func (m *GroupResourceModel) ToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *dataAccessType.AccessControlInput) diag.Diagnostics {
	diagnostics := m.GetAccessControlResourceModel().ToAccessControlInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	result.Action = new(dataAccessType.AccessControlActionGroup)

	return diagnostics
}

func (m *GroupResourceModel) FromAccessControl(ctx context.Context, client *sdk.CollibraClient, ac *dataAccessType.AccessControl) diag.Diagnostics {
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

	return diagnostics
}
