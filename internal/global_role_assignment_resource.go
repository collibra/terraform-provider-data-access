package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk-go"
	"github.com/raito-io/sdk-go/services"
	types2 "github.com/raito-io/sdk-go/types"

	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*GlobalRoleAssignmentResource)(nil)

const roleIdSuffix = "Role"
const _separator = "#"

type GlobalRoleAssignmentModel struct {
	Id   types.String `tfsdk:"id"`
	Role types.String `tfsdk:"role"`
	User types.String `tfsdk:"user"`
}

func roleId(roleName string) string {
	return roleName + roleIdSuffix
}

func (m *GlobalRoleAssignmentModel) GetRoleId() string {
	return roleId(m.Role.ValueString())
}

func _generateUniqueId(role, user string) string {
	return role + _separator + user
}

func _getRoleAndUserFromId(id string) (role, user string) {
	parts := strings.SplitN(id, _separator, 2)
	role = parts[0]
	user = parts[1]

	return
}

type GlobalRoleAssignmentResource struct {
	client *sdk.RaitoClient
}

func (g *GlobalRoleAssignmentResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_global_role_assignment"
}

func (g *GlobalRoleAssignmentResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Generate ID of GlobalRoleAssignment",
				MarkdownDescription: "Generate ID of GlobalRoleAssignment",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "Global role name",
				MarkdownDescription: "Global role name",
				Validators: []validator.String{
					stringvalidator.OneOf("Admin", "Creator", "Observer", "Integrator", "AccessCreator"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "User id",
				MarkdownDescription: "User id",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Description:         "Global Role Assignment",
		MarkdownDescription: "Global Role Assignment",
		Version:             1,
	}
}

func (g *GlobalRoleAssignmentResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data GlobalRoleAssignmentModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	_, err := g.client.Role().AssignGlobalRole(ctx, data.GetRoleId(), data.User.ValueString())
	if err != nil {
		response.Diagnostics.AddError("failed to assign global role", err.Error())

		return
	}

	data.Id = types.StringValue(_generateUniqueId(data.Role.ValueString(), data.User.ValueString()))

	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	if response.Diagnostics.HasError() {
		return
	}
}

func (g *GlobalRoleAssignmentResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var stateData GlobalRoleAssignmentModel

	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Read role assignment
	roleName, userId := _getRoleAndUserFromId(stateData.Id.ValueString())

	cancelCtx, cancel := context.WithCancel(ctx)

	defer cancel()

	roleAssignmentChannel := g.client.Role().ListRoleAssignments(cancelCtx, services.WithRoleAssignmentListFilter(
		&types2.RoleAssignmentFilterInput{
			Role: utils.Ptr(roleId(roleName)),
			User: &userId,
		}))

	var ra *types2.RoleAssignment

	for roleAssignment := range roleAssignmentChannel {
		if roleAssignment.HasError() {
			response.Diagnostics.AddError("Failed to list role assignment", roleAssignment.GetError().Error())

			return
		} else if ra != nil {
			response.Diagnostics.AddError("Multiple role assignment found", "Multiple role assignment found")

			return
		} else if roleAssignment.GetItem() == nil {
			continue
		}

		ra = roleAssignment.GetItem()
	}

	if ra == nil {
		response.State.RemoveResource(ctx)

		return
	}

	stateData.User = types.StringValue(ra.To.(*types2.RoleAssignmentToUser).Id)
	raRoleId := ra.Role.Role.GetId()
	stateData.Role = types.StringValue(strings.TrimSuffix(raRoleId, roleIdSuffix))

	response.Diagnostics.Append(response.State.Set(ctx, stateData)...)
}

func (g *GlobalRoleAssignmentResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	response.Diagnostics.AddError("Not able to update role assignment", "Not able to update role assignment")
}

func (g *GlobalRoleAssignmentResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data GlobalRoleAssignmentModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	_, err := g.client.Role().UnassignGlobalRole(ctx, data.GetRoleId(), data.User.ValueString())
	if err != nil {
		response.Diagnostics.AddError("failed to unassign global role", err.Error())

		return
	}

	response.State.RemoveResource(ctx)
}

func (g *GlobalRoleAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.RaitoClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sdk.RaitoClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	if client == nil {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *sdk.RaitoClient, not to be nil.",
		)

		return
	}

	g.client = client
}

func (g *GlobalRoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func NewGlobalRoleAssignmentResource() resource.Resource {
	return &GlobalRoleAssignmentResource{}
}
