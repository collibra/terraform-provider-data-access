package internal

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/collibra/access-governance-go-sdk"
	raitoTypes "github.com/collibra/access-governance-go-sdk/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*UserResource)(nil)

type UserResourceModel struct {
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Email             types.String `tfsdk:"email"`
	Type              types.String `tfsdk:"type"`
	Password          types.String `tfsdk:"password"`
	PasswordWo        types.String `tfsdk:"password_wo"`
	PasswordWoVersion types.Int32  `tfsdk:"password_wo_version"`
}

func (m *UserResourceModel) ToUserInput() raitoTypes.UserInput {
	return raitoTypes.UserInput{
		Name:  m.Name.ValueStringPointer(),
		Email: m.Email.ValueStringPointer(),
		Type:  (*raitoTypes.UserType)(m.Type.ValueStringPointer()),
	}
}

type UserResource struct {
	client *sdk.CollibraClient
}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

func (u *UserResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_user"
}

func (u *UserResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the user",
				MarkdownDescription: "The ID of the user",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "The name of the user",
				MarkdownDescription: "The name of the user",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(3)},
			},
			"email": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "The email of the user",
				MarkdownDescription: "The email of the user",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(3)},
			},
			"type": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "The type of the user (Human or Machine)",
				MarkdownDescription: "The type of the user (Human or Machine)",
				Default:             stringdefault.StaticString(string(raitoTypes.UserTypeHuman)),
				Validators: []validator.String{
					stringvalidator.OneOf(utils.StringArray(raitoTypes.AllUserType)...),
				},
			},
			"password": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            false,
				Sensitive:           true,
				WriteOnly:           false,
				Description:         "The password of the user, if set the user will be created as Raito User. Preferably use password_wo.",
				MarkdownDescription: "The password of the user, if set the user will be created as Raito User. Preferably use password_wo.",
				Validators: []validator.String{
					stringvalidator.All(
						stringvalidator.LengthAtLeast(8),
						stringvalidator.RegexMatches(regexp.MustCompile(".*[a-z].*"), "requires at least one lowercase letter"),
						stringvalidator.RegexMatches(regexp.MustCompile(".*[A-Z].*"), " requires at least one uppercase letter"),
						stringvalidator.RegexMatches(regexp.MustCompile(`.*\d.*`), "requires at least one number"),
						stringvalidator.RegexMatches(regexp.MustCompile(".*[!@#$%^&*].*"), "requires at least one special character"),
						stringvalidator.PreferWriteOnlyAttribute(
							path.MatchRoot("password_wo"),
						),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password_wo": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            false,
				Sensitive:           true,
				WriteOnly:           true,
				Description:         "The password of the user, if set the user will be created as Raito User",
				MarkdownDescription: "The password of the user, if set the user will be created as Raito User",
				Validators: []validator.String{
					stringvalidator.All(
						stringvalidator.LengthAtLeast(8),
						stringvalidator.RegexMatches(regexp.MustCompile(".*[a-z].*"), "requires at least one lowercase letter"),
						stringvalidator.RegexMatches(regexp.MustCompile(".*[A-Z].*"), " requires at least one uppercase letter"),
						stringvalidator.RegexMatches(regexp.MustCompile(`.*\d.*`), "requires at least one number"),
						stringvalidator.RegexMatches(regexp.MustCompile(".*[!@#$%^&*].*"), "requires at least one special character"),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password_wo_version": schema.Int32Attribute{
				Required:            false,
				Optional:            true,
				Computed:            false,
				Sensitive:           false,
				Description:         "Version of the password_wo. This is used to force the password to be updated.",
				MarkdownDescription: "Version of the password_wo. This is used to force the password to be updated.",
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
				Default: nil,
			},
			"raito_user": schema.BoolAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Indicates if a user is a Raito User",
				MarkdownDescription: "Indicates if a user is a Raito User",
				Default:             booldefault.StaticBool(true),
			},
		},
		Description:         "User resource",
		MarkdownDescription: "The resource for representing a [User](https://docs.raito.io/docs/cloud/admin/user_management) in Raito.",
		Version:             1,
	}
}

func (u *UserResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var stateData UserResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Read user
	user, err := u.client.User().GetUser(ctx, stateData.Id.ValueString())
	if err != nil {
		var notFoundErr *raitoTypes.ErrNotFound
		if errors.As(err, &notFoundErr) {
			response.State.RemoveResource(ctx)
		} else {
			response.Diagnostics.AddError("Failed to read user", err.Error())
		}

		return
	}

	if response.Diagnostics.HasError() {
		return
	}

	actualData := UserResourceModel{
		Id:                types.StringValue(user.Id),
		Name:              types.StringValue(user.Name),
		Email:             types.StringPointerValue(user.Email),
		Type:              types.StringValue(string(user.Type)),
		Password:          stateData.Password,
		PasswordWoVersion: stateData.PasswordWoVersion,
	}

	response.Diagnostics.Append(response.State.Set(ctx, &actualData)...)
}

func (u *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.CollibraClient)

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

	u.client = client
}

func (u *UserResource) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var data UserResourceModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	isRaitoUser := data.RaitoUser.IsNull() || data.RaitoUser.ValueBool()

	if !data.Password.IsNull() && !isRaitoUser {
		response.Diagnostics.AddError("Password cannot be set if the user is not a Raito user", "Password cannot be set if the user is not a Raito user")
	}
}
