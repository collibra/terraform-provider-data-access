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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/collibra/terraform-provider-data-access/internal/utils"
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
		Description:         "The user IDs of the owners of the group.",
		MarkdownDescription: "The user IDs of the owners of the group.",
		Validators: []validator.Set{
			setvalidator.ValueStringsAre(
				stringvalidator.LengthAtLeast(3),
			),
		},
		Default: nil,
	}
	attributes["data_sources"] = schema.SetAttribute{
		ElementType:         types.StringType,
		Required:            false,
		Optional:            true,
		Computed:            true,
		Sensitive:           false,
		Description:         "The data sources that the group applies to.",
		MarkdownDescription: "The data sources that the group applies to.",
		Validators: []validator.Set{
			setvalidator.ValueStringsAre(
				stringvalidator.LengthAtLeast(3),
			),
		},
		Default: nil,
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

	diagnostics.Append(dataSourceIdsToAccessControlInput(ctx, m.DataSources, result)...)

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

	dataSources, dsDiagnostics := dataSourceIdsFromAccessControl(ctx, ac)
	diagnostics.Append(dsDiagnostics...)

	if diagnostics.HasError() {
		return diagnostics
	}

	m.DataSources = dataSources

	return diagnostics
}

// dataSourceIdsToAccessControlInput converts a set of data source IDs from the Terraform model to the Collibra AccessControlInput model.
func dataSourceIdsToAccessControlInput(ctx context.Context, dataSources types.Set, result *dataAccessType.AccessControlInput) (diagnostics diag.Diagnostics) {
	if dataSources.IsNull() || dataSources.IsUnknown() {
		return diagnostics
	}

	dataSourceIds, diagnostics := utils.StringSetToSlice(ctx, dataSources)
	if diagnostics.HasError() {
		return diagnostics
	}

	result.DataSources = make([]dataAccessType.AccessControlDataSourceInput, 0, len(dataSourceIds))

	for _, id := range dataSourceIds {
		result.DataSources = append(result.DataSources, dataAccessType.AccessControlDataSourceInput{
			DataSource: id,
		})
	}

	return diagnostics
}

// dataSourceIdsFromAccessControl converts the data source IDs from the AccessControl Collibra model to a set in the Terraform model.
func dataSourceIdsFromAccessControl(ctx context.Context, ac *dataAccessType.AccessControl) (types.Set, diag.Diagnostics) {
	dataSourceIds := make([]string, 0, len(ac.SyncData))

	for i := range ac.SyncData {
		dataSourceIds = append(dataSourceIds, ac.SyncData[i].DataSource.Id)
	}

	return utils.SliceToStringSet(ctx, dataSourceIds)
}
