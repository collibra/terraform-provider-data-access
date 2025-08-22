package internal

import (
	"context"
	"errors"
	"fmt"
	"regexp"

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
	"github.com/raito-io/sdk-go"
	"github.com/raito-io/sdk-go/services"
	raitoTypes "github.com/raito-io/sdk-go/types"

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
	RaitoUser         types.Bool   `tfsdk:"raito_user"`
}

func (m *UserResourceModel) ToUserInput() raitoTypes.UserInput {
	return raitoTypes.UserInput{
		Name:  m.Name.ValueStringPointer(),
		Email: m.Email.ValueStringPointer(),
		Type:  (*raitoTypes.UserType)(m.Type.ValueStringPointer()),
	}
}

type UserResource struct {
	client *sdk.RaitoClient
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

func (u *UserResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data UserResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	user, err := u.client.User().GetUserByEmail(ctx, data.Email.ValueString())
	if err != nil {
		var notFoundErr *raitoTypes.ErrNotFound
		if !errors.As(err, &notFoundErr) {
			response.Diagnostics.AddError("Failed to check if user already exists", err.Error())

			return
		}
	}

	if user != nil {
		// Update user
		user, err = u.client.User().UpdateUser(ctx, user.Id, data.ToUserInput())
		if err != nil {
			response.Diagnostics.AddError("Failed to update user", err.Error())

			return
		}
	} else {
		// Create user
		user, err = u.client.User().CreateUser(ctx, data.ToUserInput())
		if err != nil {
			response.Diagnostics.AddError("Failed to create user", err.Error())

			return
		}
	}

	data.Id = types.StringValue(user.Id)
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)

	if data.RaitoUser.ValueBool() && !user.IsRaitoUser {
		options := make([]func(options *services.InviteAsRaitoUserOptions), 0, 1)

		if data.Password.IsNull() && data.PasswordWo.IsNull() {
			options = append(options, services.WithInviteAsRaitoUserNoPassword())
		}

		user, err = u.client.User().InviteAsRaitoUser(ctx, user.Id, options...)
		if err != nil {
			response.Diagnostics.AddError("Failed to invite user as Raito user", err.Error())

			return
		}
	} else if user.IsRaitoUser && !data.RaitoUser.ValueBool() {
		user, err = u.client.User().RemoveAsRaitoUser(ctx, user.Id)
		if err != nil {
			response.Diagnostics.AddError("Failed to revoke user as Raito user", err.Error())
		}
	}

	data.RaitoUser = types.BoolValue(user.IsRaitoUser)
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)

	if !data.Password.IsNull() || !data.PasswordWo.IsNull() {
		var password string

		if !data.PasswordWo.IsNull() {
			password = data.PasswordWo.ValueString()
		} else {
			password = data.Password.ValueString()
		}

		_, err = u.client.User().SetUserPassword(ctx, user.Id, password)
		if err != nil {
			response.Diagnostics.AddError("Failed to set user password", err.Error())

			return
		}
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
		Id:        types.StringValue(user.Id),
		Name:      types.StringValue(user.Name),
		Email:     types.StringPointerValue(user.Email),
		Type:      types.StringValue(string(user.Type)),
		Password:  stateData.Password,
		RaitoUser: types.BoolValue(user.IsRaitoUser),
	}

	response.Diagnostics.Append(response.State.Set(ctx, &actualData)...)
}

func (u *UserResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var planData UserResourceModel
	var stateData UserResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &planData)...)
	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Update user
	user, err := u.client.User().UpdateUser(ctx, planData.Id.ValueString(), planData.ToUserInput())
	if err != nil {
		response.Diagnostics.AddError("Failed to update user", err.Error())

		return
	}

	if (planData.RaitoUser.ValueBool() && !stateData.RaitoUser.ValueBool()) || (!planData.Password.IsNull() && stateData.Password.IsNull() && planData.RaitoUser.ValueBool()) {
		user, err = u.client.User().InviteAsRaitoUser(ctx, user.Id)
		if err != nil {
			response.Diagnostics.AddError("Failed to invite user as Raito user", err.Error())

			return
		}

		stateData.RaitoUser = types.BoolValue(user.IsRaitoUser)
		response.Diagnostics.Append(response.State.Set(ctx, &stateData)...)
	} else if !planData.RaitoUser.ValueBool() && stateData.RaitoUser.ValueBool() {
		user, err = u.client.User().RemoveAsRaitoUser(ctx, user.Id)
		if err != nil {
			response.Diagnostics.AddError("Failed to revoke user as Raito user", err.Error())

			return
		}

		stateData.RaitoUser = types.BoolValue(user.IsRaitoUser)
		response.Diagnostics.Append(response.State.Set(ctx, &stateData)...)
	}

	if !planData.Password.IsNull() && planData.Password.ValueString() != stateData.Password.ValueString() {
		user, err = u.client.User().SetUserPassword(ctx, user.Id, planData.Password.ValueString())
		if err != nil {
			response.Diagnostics.AddError("Failed to set user password", err.Error())

			return
		}
	} else if !planData.PasswordWo.IsNull() && !planData.PasswordWoVersion.IsNull() && !stateData.PasswordWoVersion.Equal(planData.PasswordWoVersion) {
		user, err = u.client.User().SetUserPassword(ctx, user.Id, planData.PasswordWo.ValueString())
		if err != nil {
			response.Diagnostics.AddError("Failed to set user password_wo", err.Error())

			return
		}
	}

	planData.RaitoUser = types.BoolValue(user.IsRaitoUser)
	response.Diagnostics.Append(response.State.Set(ctx, &planData)...)
}

func (u *UserResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var stateData UserResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)

	if response.Diagnostics.HasError() {
		return
	}

	if stateData.RaitoUser.ValueBool() {
		_, err := u.client.User().RemoveAsRaitoUser(ctx, stateData.Id.ValueString())
		if err != nil {
			var notFoundErr *raitoTypes.ErrNotFound
			if errors.As(err, &notFoundErr) {
				response.State.RemoveResource(ctx)
			} else {
				response.Diagnostics.AddError("Failed to remove user", err.Error())
			}

			return
		}

		tmpU, err := u.client.User().GetUser(ctx, stateData.Id.ValueString())
		if err != nil {
			response.State.RemoveResource(ctx)

			return
		}

		_ = tmpU
	}

	err := u.client.User().DeleteUser(ctx, stateData.Id.ValueString())
	if err != nil {
		var notFoundErr *raitoTypes.ErrNotFound
		if errors.As(err, &notFoundErr) {
			response.State.RemoveResource(ctx)
		} else {
			response.Diagnostics.AddError("Failed to delete user "+stateData.Id.ValueString(), err.Error())
		}

		return
	}

	response.State.RemoveResource(ctx)
}

func (u *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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
