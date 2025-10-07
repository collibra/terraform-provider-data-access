package internal

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/collibra/access-governance-go-sdk"
	"github.com/collibra/access-governance-go-sdk/services"
	accessGovernanceType "github.com/collibra/access-governance-go-sdk/types"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/raito-io/golang-set/set"

	"github.com/collibra/access-governance-terraform-provider/internal/types/abac_expression"
	"github.com/collibra/access-governance-terraform-provider/internal/utils"
)

const (
	lockMsg = "Locked by terraform"
)

type AccessControlResourceModel struct {
	Id                types.String
	Name              types.String
	Description       types.String
	State             types.String
	Who               types.Set
	WhoAbacRule       jsontypes.Normalized
	WhoLocked         types.Bool
	InheritanceLocked types.Bool

	Owners types.Set
}

type AccessControlModel[T any] interface {
	*T
	GetAccessControlResourceModel() *AccessControlResourceModel
	SetAccessControlResourceModel(model *AccessControlResourceModel)
	ToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *accessGovernanceType.AccessControlInput) diag.Diagnostics
	FromAccessControl(ctx context.Context, client *sdk.CollibraClient, input *accessGovernanceType.AccessControl) diag.Diagnostics
	UpdateOwners(owners types.Set)
}

type ReadHook[T any, ApModel AccessControlModel[T]] func(ctx context.Context, client *sdk.CollibraClient, data ApModel) diag.Diagnostics
type ValidationHook[T any, ApModel AccessControlModel[T]] func(ctx context.Context, data ApModel) diag.Diagnostics
type PlanModifierHook[T any, ApModel AccessControlModel[T]] func(ctx context.Context, data ApModel) (ApModel, diag.Diagnostics)

type AccessControlResource[T any, ApModel AccessControlModel[T]] struct {
	client *sdk.CollibraClient

	readHooks         []ReadHook[T, ApModel]
	validationHooks   []ValidationHook[T, ApModel]
	planModifierHooks []PlanModifierHook[T, ApModel]
}

func (a *AccessControlResource[T, ApModel]) schema(typeName string) map[string]schema.Attribute {
	defaultSchema := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Required:            false,
			Optional:            false,
			Computed:            true,
			Sensitive:           false,
			Description:         fmt.Sprintf("The ID of the %s.", typeName),
			MarkdownDescription: fmt.Sprintf("The ID of the %s", typeName),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			Optional:            false,
			Computed:            false,
			Sensitive:           false,
			Description:         fmt.Sprintf("The name of the %s", typeName),
			MarkdownDescription: fmt.Sprintf("The name of the %s", typeName),
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(3),
			},
		},
		"description": schema.StringAttribute{
			Required:            false,
			Optional:            true,
			Computed:            true,
			Sensitive:           false,
			Description:         fmt.Sprintf("The description of the %s", typeName),
			MarkdownDescription: fmt.Sprintf("The description of the %s", typeName),
			Default:             stringdefault.StaticString(""),
		},
		"state": schema.StringAttribute{
			Required:            false,
			Optional:            true,
			Computed:            true,
			Sensitive:           false,
			Description:         fmt.Sprintf("The state of the %s", typeName),
			MarkdownDescription: fmt.Sprintf("The state of the %s Possible values are: [%q, %q]", typeName, string(accessGovernanceType.AccessControlStateActive), string(accessGovernanceType.AccessControlStateInactive)),
			Validators: []validator.String{
				stringvalidator.OneOf(string(accessGovernanceType.AccessControlStateActive), string(accessGovernanceType.AccessControlStateInactive)),
			},
			Default: stringdefault.StaticString(string(accessGovernanceType.AccessControlStateActive)),
		},
		"who": schema.SetNestedAttribute{
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"user": schema.StringAttribute{
						Required:            false,
						Optional:            true,
						Computed:            false,
						Sensitive:           false,
						Description:         "The email address of user",
						MarkdownDescription: "The email address of the user. This cannot be set if `access_control` is set.",
						Validators: []validator.String{
							stringvalidator.RegexMatches(regexp.MustCompile(`.+@.+\..+`), "value must be a valid email address"),
						},
					},
					"access_control": schema.StringAttribute{
						Required:            false,
						Optional:            true,
						Computed:            false,
						Sensitive:           false,
						Description:         "The ID of the access control in Raito Cloud",
						MarkdownDescription: "The ID of the access control in Raito Cloud. Cannot be set if `user` is set.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(3),
						},
					},
					"promise_duration": schema.Int64Attribute{
						Required:            false,
						Optional:            true,
						Computed:            false,
						Sensitive:           false,
						Description:         "Specify this to indicate that this who-item is a promise instead of a direct grant. This is specified as the number of seconds that access should be granted when requested.",
						MarkdownDescription: "Specify this to indicate that this who-item is a promise instead of a direct grant. This is specified as the number of seconds that access should be granted when requested.",
						Validators: []validator.Int64{
							int64validator.AtLeast(1),
						},
					},
				},
				CustomType:    nil,
				Validators:    nil,
				PlanModifiers: nil,
			},
			Required:            false,
			Optional:            true,
			Computed:            false,
			Sensitive:           false,
			Description:         fmt.Sprintf("The who-items associated with the %s", typeName),
			MarkdownDescription: fmt.Sprintf("The who-items associated with the %s. When this is not set (nil), the who-list will not be overridden. This is typically used when this should be managed from Raito Cloud.", typeName),
		},
		"who_abac_rule": schema.StringAttribute{
			CustomType:          jsontypes.NormalizedType{},
			Required:            false,
			Optional:            true,
			Computed:            false,
			Sensitive:           false,
			Description:         fmt.Sprintf("json representation of the abac rule for who-items associated with the %s", typeName),
			MarkdownDescription: fmt.Sprintf("json representation of the abac rule for who-items associated with the %s", typeName),
		},
		"who_locked": schema.BoolAttribute{
			Required:            false,
			Optional:            true,
			Computed:            true,
			Sensitive:           false,
			Description:         "Indicates if who should be locked. This should be true if who users or who_abac_rule is set.",
			MarkdownDescription: "Indicates if who should be locked. This should be true if who users or who_abac_rule is set.",
			Validators:          nil,
		},
		"inheritance_locked": schema.BoolAttribute{
			Required:            false,
			Optional:            true,
			Computed:            true,
			Sensitive:           false,
			Description:         "Indicates if who should be locked. This should be true if who access providers are set.",
			MarkdownDescription: "Indicates if who should be locked. This should be true if who access providers are set.",
			Validators:          nil,
		},
		"owners": schema.SetAttribute{
			ElementType:         types.StringType,
			Required:            false,
			Optional:            true,
			Computed:            true,
			Sensitive:           false,
			Description:         fmt.Sprintf("User id of the owners of this %s", typeName),
			MarkdownDescription: fmt.Sprintf("User id of the owners of this %s", typeName),
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(
					stringvalidator.LengthAtLeast(3),
				),
			},
			Default: nil,
		},
	}

	return defaultSchema
}

func (a *AccessControlResource[T, ApModel]) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data T

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	a.create(ctx, &data, response)
}

func (a *AccessControlResource[T, ApModel]) create(ctx context.Context, data ApModel, response *resource.CreateResponse) {
	input := accessGovernanceType.AccessControlInput{}

	apResourceModel := data.GetAccessControlResourceModel()

	state := apResourceModel.State
	owners := apResourceModel.Owners

	response.Diagnostics.Append(data.ToAccessControlInput(ctx, a.client, &input)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Create the access provider
	ac, err := a.client.AccessControl().CreateAccessControl(ctx, input)
	if err != nil {
		response.Diagnostics.AddError("Failed to create access provider", err.Error())

		return
	}

	response.Diagnostics.Append(data.FromAccessControl(ctx, a.client, ac)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	if response.Diagnostics.HasError() {
		return
	}

	ac, diagnostics := a.updateState(ctx, data, state, ac)

	response.Diagnostics.Append(diagnostics...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(data.FromAccessControl(ctx, a.client, ac)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(a.createUpdateOwners(ctx, data, owners, ac, &response.State)...)
}

func (a *AccessControlResource[T, ApModel]) createUpdateOwners(ctx context.Context, data ApModel, owners types.Set, ac *accessGovernanceType.AccessControl, state *tfsdk.State) (diagnostics diag.Diagnostics) {
	if !owners.IsNull() && !owners.IsUnknown() {
		ownerElements := owners.Elements()

		ownerIds := make([]string, len(ownerElements))
		for i, ownerElement := range ownerElements {
			ownerIds[i] = ownerElement.(types.String).ValueString()
		}

		_, err := a.client.Role().UpdateRoleAssigneesOnAccessControl(ctx, ac.Id, ownerRole, ownerIds...)
		if err != nil {
			diagnostics.AddError("Failed to update owners of access provider", err.Error())

			return diagnostics
		}
	} else {
		ownerSet, ownerDiagnostics := a.readOwners(ctx, ac.Id)
		diagnostics.Append(ownerDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		data.UpdateOwners(ownerSet)
		diagnostics.Append(state.Set(ctx, data)...)
	}

	return diagnostics
}

func (a *AccessControlResource[T, ApModel]) updateState(ctx context.Context, data ApModel, state types.String, ac *accessGovernanceType.AccessControl) (_ *accessGovernanceType.AccessControl, diagnostics diag.Diagnostics) {
	if state.Equal(data.GetAccessControlResourceModel().State) {
		return ac, diagnostics
	}

	var err error

	if data.GetAccessControlResourceModel().State.ValueString() == string(accessGovernanceType.AccessControlStateActive) {
		ac, err = a.client.AccessControl().DeactivateAccessControl(ctx, ac.Id)
		if err != nil {
			diagnostics.AddError("Failed to activate access provider", err.Error())

			return ac, diagnostics
		}
	} else if data.GetAccessControlResourceModel().State.ValueString() == string(accessGovernanceType.AccessControlStateInactive) {
		ac, err = a.client.AccessControl().ActivateAccessControl(ctx, ac.Id)
		if err != nil {
			diagnostics.AddError("Failed to deactivate access provider", err.Error())

			return ac, diagnostics
		}
	} else {
		diagnostics.AddError("Invalid state", fmt.Sprintf("Invalid state: %s", data.GetAccessControlResourceModel().State.ValueString()))

		return ac, diagnostics
	}

	return ac, diagnostics
}

func (a *AccessControlResource[T, ApModel]) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data T

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	a.read(ctx, &data, response, a.readHooks...)
}

func (a *AccessControlResource[T, ApModel]) read(ctx context.Context, data ApModel, response *resource.ReadResponse, hooks ...ReadHook[T, ApModel]) {
	apModel := data.GetAccessControlResourceModel()

	// Get the access provider
	ac, err := a.client.AccessControl().GetAccessControl(ctx, apModel.Id.ValueString())
	if err != nil {
		notFoundErr := &accessGovernanceType.ErrNotFound{}
		if errors.As(err, &notFoundErr) {
			response.State.RemoveResource(ctx)

			return
		}

		response.Diagnostics.AddError("Failed to read access provider", err.Error())

		return
	}

	if ac.State == accessGovernanceType.AccessControlStateDeleted {
		response.State.RemoveResource(ctx)

		return
	}

	response.Diagnostics.Append(data.FromAccessControl(ctx, a.client, ac)...)

	if response.Diagnostics.HasError() {
		return
	}

	apModel = data.GetAccessControlResourceModel()

	// If who in initial state is not nil, get all who-items
	if !apModel.Who.IsNull() {
		definedPromises := set.Set[string]{}

		// Search al promises defined in the terraform state
		for _, whoItem := range apModel.Who.Elements() {
			whoItemObject := whoItem.(types.Object)
			attributes := whoItemObject.Attributes()

			if !attributes["promise_duration"].IsNull() {
				if !attributes["user"].IsNull() {
					definedPromises.Add(_userPrefix(attributes["user"].(types.String).ValueString()))
				} else if !attributes["access_control"].IsNull() {
					definedPromises.Add(_accessControlPrefix(attributes["access_control"].(types.String).ValueString()))
				}
			}
		}

		stateWhoItems := make([]attr.Value, 0)

		stateWhoItems, done := a.readWhoItems(ctx, apModel, response, definedPromises, stateWhoItems)
		if done {
			return
		}

		who, whoDiag := types.SetValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"user":             types.StringType,
				"access_control":   types.StringType,
				"promise_duration": types.Int64Type,
			},
		}, stateWhoItems)

		response.Diagnostics.Append(whoDiag...)

		if response.Diagnostics.HasError() {
			return
		}

		apModel.Who = who
	}

	if !apModel.Who.IsNull() && ac.WhoAbacRule != nil {
		apModel.WhoAbacRule = jsontypes.NewNormalizedPointerValue(ac.WhoAbacRule.RuleJson)
	}

	// Set all global access provider attributes
	data.SetAccessControlResourceModel(apModel)

	// Read owners
	ownersSet, ownerDiagnostics := a.readOwners(ctx, apModel.Id.ValueString())
	response.Diagnostics.Append(ownerDiagnostics...)

	if response.Diagnostics.HasError() {
		return
	}

	data.UpdateOwners(ownersSet)

	// Execute action specific hooks
	for _, hook := range hooks {
		response.Diagnostics.Append(hook(ctx, a.client, data)...)

		if response.Diagnostics.HasError() {
			return
		}
	}

	// Set new state of the access provider
	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (a *AccessControlResource[T, ApModel]) readWhoItems(ctx context.Context, apModel *AccessControlResourceModel, response *resource.ReadResponse, definedPromises set.Set[string], stateWhoItems []attr.Value) ([]attr.Value, bool) {
	whoItems := a.client.AccessControl().GetAccessControlWhoList(ctx, apModel.Id.ValueString())
	for whoItem, err := range whoItems {
		if err != nil {
			response.Diagnostics.AddError("Failed to read who-item from access provider", err.Error())

			return nil, true
		}

		var user, whoAp *string

		switch benificiaryItem := whoItem.Item.(type) {
		case *accessGovernanceType.AccessWhoItemItemUser:
			user = benificiaryItem.Email
		case *accessGovernanceType.AccessWhoItemItemAccessControl:
			whoAp = &benificiaryItem.Id
		default:
			response.Diagnostics.AddError("Invalid who-item", fmt.Sprintf("Invalid who-item: %T", benificiaryItem))

			return nil, true
		}

		if whoItem.Type == accessGovernanceType.AccessWhoItemTypeWhogrant {
			if (user != nil && definedPromises.Contains(_userPrefix(*user))) || (whoAp != nil && definedPromises.Contains(_accessControlPrefix(*whoAp))) {
				continue
			}
		} else if whoItem.PromiseDuration == nil {
			response.Diagnostics.AddError("Invalid who-item detected.", "Invalid who-item. Promise duration not set on promise who-item")
		}

		stateWhoItems = append(stateWhoItems, types.ObjectValueMust(
			map[string]attr.Type{
				"user":             types.StringType,
				"access_control":   types.StringType,
				"promise_duration": types.Int64Type,
			}, map[string]attr.Value{
				"user":             types.StringPointerValue(user),
				"access_control":   types.StringPointerValue(whoAp),
				"promise_duration": types.Int64PointerValue(whoItem.PromiseDuration),
			}))
	}

	return stateWhoItems, false
}

func (a *AccessControlResource[T, ApModel]) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data T

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	a.update(ctx, &data, response)
}

func (a *AccessControlResource[T, ApModel]) update(ctx context.Context, data ApModel, response *resource.UpdateResponse) {
	input := accessGovernanceType.AccessControlInput{}

	apResourceModel := data.GetAccessControlResourceModel()

	id := apResourceModel.Id.ValueString()
	state := apResourceModel.State
	owners := apResourceModel.Owners

	response.Diagnostics.Append(data.ToAccessControlInput(ctx, a.client, &input)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Check for implemented promises
	definedPromises := set.Set[string]{}

	for _, whoItem := range input.WhoItems {
		if whoItem.Type != nil && *whoItem.Type == accessGovernanceType.AccessWhoItemTypeWhopromise {
			if whoItem.User != nil {
				definedPromises.Add(_userPrefix(*whoItem.User))
			} else if whoItem.AccessControl != nil {
				definedPromises.Add(_accessControlPrefix(*whoItem.AccessControl))
			}
		}
	}

	if a.updateGetWhoItems(ctx, id, response, definedPromises, input) {
		return
	}

	// Update access provider
	ac, err := a.client.AccessControl().UpdateAccessControl(ctx, id, input, services.WithAccessControlOverrideLocks())
	if err != nil {
		response.Diagnostics.AddError("Failed to update access provider", err.Error())

		return
	}

	response.Diagnostics.Append(data.FromAccessControl(ctx, a.client, ac)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	if response.Diagnostics.HasError() {
		return
	}

	ac, diagnostics := a.updateState(ctx, data, state, ac)

	response.Diagnostics.Append(diagnostics...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(data.FromAccessControl(ctx, a.client, ac)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Update owners
	response.Diagnostics.Append(a.createUpdateOwners(ctx, data, owners, ac, &response.State)...)
}

func (a *AccessControlResource[T, ApModel]) updateGetWhoItems(ctx context.Context, id string, response *resource.UpdateResponse, definedPromises set.Set[string], input accessGovernanceType.AccessControlInput) bool {
	whoItems := a.client.AccessControl().GetAccessControlWhoList(ctx, id)
	for whoItem, err := range whoItems {
		if err != nil {
			response.Diagnostics.AddError("Failed to read who-item from access provider", err.Error())

			return true
		}

		if whoItem.Type == accessGovernanceType.AccessWhoItemTypeWhogrant {
			var key string
			var user, whoAp *string

			switch beneficiaryItem := whoItem.Item.(type) {
			case *accessGovernanceType.AccessWhoItemItemUser:
				if beneficiaryItem.Email == nil {
					continue
				}

				key = _userPrefix(*beneficiaryItem.Email)
				user = &beneficiaryItem.Id
			case *accessGovernanceType.AccessWhoItemItemAccessControl:
				key = _accessControlPrefix(beneficiaryItem.Id)
				whoAp = &beneficiaryItem.Id
			default:
				continue
			}

			if definedPromises.Contains(key) {
				input.WhoItems = append(input.WhoItems, accessGovernanceType.WhoItemInput{
					Type:          utils.Ptr(accessGovernanceType.AccessWhoItemTypeWhogrant),
					User:          user,
					AccessControl: whoAp,
					ExpiresAt:     whoItem.ExpiresAt,
				})
			}
		}
	}

	return false
}

func (a *AccessControlResource[T, ApModel]) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data T

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	apModel := ApModel(&data)

	err := a.client.AccessControl().DeleteAccessControl(ctx, apModel.GetAccessControlResourceModel().Id.ValueString(), services.WithAccessControlOverrideLocks())
	if err != nil {
		response.Diagnostics.AddError("Failed to delete access provider", err.Error())

		return
	}

	response.State.RemoveResource(ctx)
}

func (a *AccessControlResource[T, ApModel]) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.CollibraClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sdk.CollibraClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	if client == nil {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *sdk.CollibraClient, not to be nil.",
		)

		return
	}

	a.client = client
}

func (a *AccessControlResource[T, ApModel]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (a *AccessControlResource[T, ApModel]) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var data T

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	apModel := ApModel(&data)

	apResourceModel := apModel.GetAccessControlResourceModel()

	who := &apResourceModel.Who
	whoAbac := &apResourceModel.WhoAbacRule

	whoUsersDefined := false
	whoAccessControlsDefined := false

	if !who.IsNull() && !whoAbac.IsNull() {
		response.Diagnostics.AddError(
			"Cannot specify both who and who_abac",
			"Please specify only one of who or who_abac",
		)
	} else if !who.IsNull() { // For each who-item check if exactly one of user or access_control is set.
		for _, whoItem := range who.Elements() {
			whoItemAttribute := whoItem.(types.Object)

			attributes := whoItemAttribute.Attributes()

			attributesFound := 0

			attrFn := func(key string, indicator *bool) {
				if attribute, found := attributes[key]; found && !attribute.IsNull() {
					attributesFound++
					*indicator = true
				}
			}

			attrFn("user", &whoUsersDefined)
			attrFn("access_control", &whoAccessControlsDefined)

			if attributesFound != 1 {
				response.Diagnostics.AddError(
					"Invalid who-item. Exactly one of user or access_control must be set.",
					fmt.Sprintf("Expected exactly one of user or access_control, got: %d.", attributesFound),
				)

				break
			}
		}
	}

	if whoUsersDefined || !whoAbac.IsNull() {
		if !apResourceModel.WhoLocked.IsNull() && !apResourceModel.WhoLocked.ValueBool() {
			response.Diagnostics.AddError("Who must be locked", "Who must be locked if who users or who_abac_rule is set.")
		}
	}

	if whoAccessControlsDefined {
		if !apResourceModel.InheritanceLocked.IsNull() && !apResourceModel.InheritanceLocked.ValueBool() {
			response.Diagnostics.AddError("Inheritance must be locked", "Inheritance must be locked if who access providers are set.")
		}
	}

	for _, validatorHook := range a.validationHooks {
		response.Diagnostics.Append(validatorHook(ctx, apModel)...)
	}
}

func (a *AccessControlResource[T, ApModel]) readOwners(ctx context.Context, apId string) (_ types.Set, diagnostics diag.Diagnostics) {
	roleAssignments := a.client.Role().ListRoleAssignmentsOnAccessControl(ctx, apId, services.WithRoleAssignmentListFilter(&accessGovernanceType.RoleAssignmentFilterInput{
		Role: utils.Ptr(ownerRole),
	}))

	var ownerIds []attr.Value

	for roleAssignment, err := range roleAssignments {
		if err != nil {
			diagnostics.AddError("Failed to list role assignments on access provider", err.Error())

			return basetypes.SetValue{}, diagnostics
		}

		switch to := roleAssignment.To.(type) {
		case *accessGovernanceType.RoleAssignmentToUser:
			ownerIds = append(ownerIds, types.StringValue(to.Id))
		default:
			diagnostics.AddError("Unexpected role assignment type", fmt.Sprintf("Unexpected role assignment type %T", to))

			return basetypes.SetValue{}, diagnostics
		}
	}

	ownerSet, diagOwners := types.SetValue(types.StringType, ownerIds)
	diagnostics.Append(diagOwners...)

	if diagnostics.HasError() {
		return basetypes.SetValue{}, diagnostics
	}

	return ownerSet, diagnostics
}

func (a *AccessControlResource[T, ApModel]) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		resp.Plan = req.Plan

		return
	}

	var data T

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apModel := ApModel(&data)
	apResourceModel := apModel.GetAccessControlResourceModel()

	whoUsersDefined := false
	whoAccessControlsDefined := false

	if !apResourceModel.Who.IsNull() {
		for _, whoItem := range apResourceModel.Who.Elements() {
			whoItemAttribute := whoItem.(types.Object)

			attributes := whoItemAttribute.Attributes()

			attrFn := func(key string, indicator *bool) {
				if attribute, found := attributes[key]; found && !attribute.IsNull() {
					*indicator = true
				}
			}

			attrFn("user", &whoUsersDefined)
			attrFn("access_control", &whoAccessControlsDefined)
		}
	}

	if whoUsersDefined || !apResourceModel.WhoAbacRule.IsNull() {
		apResourceModel.WhoLocked = types.BoolValue(true)
	} else if apResourceModel.WhoLocked.IsUnknown() {
		apResourceModel.WhoLocked = types.BoolValue(false)
	}

	if whoAccessControlsDefined {
		apResourceModel.InheritanceLocked = types.BoolValue(true)
	} else if apResourceModel.InheritanceLocked.IsUnknown() {
		apResourceModel.InheritanceLocked = types.BoolValue(false)
	}

	apModel.SetAccessControlResourceModel(apResourceModel)

	for _, planModifierHook := range a.planModifierHooks {
		updatedModel, planModifierDiag := planModifierHook(ctx, apModel)
		resp.Diagnostics.Append(planModifierDiag...)

		if resp.Diagnostics.HasError() {
			return
		}

		apModel = updatedModel
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, apModel)...)
}

func (a *AccessControlResourceModel) ToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *accessGovernanceType.AccessControlInput) (diagnostics diag.Diagnostics) {
	result.Name = a.Name.ValueStringPointer()
	result.Description = a.Description.ValueStringPointer()
	result.Locks = append(result.Locks,
		accessGovernanceType.AccessControlLockDataInput{
			LockKey: accessGovernanceType.AccessControlLockNamelock,
			Details: &accessGovernanceType.AccessControlLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		},
	)

	result.WhoType = utils.Ptr(accessGovernanceType.WhoAndWhatTypeStatic)

	if !a.Who.IsNull() && !a.Who.IsUnknown() {
		diagnostics.Append(a.whoElementsToAccessControlInput(ctx, client, result)...)
	} else if !a.WhoAbacRule.IsNull() && !a.WhoAbacRule.IsUnknown() {
		result.WhoType = utils.Ptr(accessGovernanceType.WhoAndWhatTypeDynamic)
		diagnostics.Append(a.whoAbacRuleToAccessControlInput(result)...)
	}

	if a.WhoLocked.ValueBool() {
		result.Locks = append(result.Locks,
			accessGovernanceType.AccessControlLockDataInput{
				LockKey: accessGovernanceType.AccessControlLockWholock,
				Details: &accessGovernanceType.AccessControlLockDetailsInput{
					Reason: utils.Ptr(lockMsg),
				},
			},
		)
	}

	if a.InheritanceLocked.ValueBool() {
		result.Locks = append(result.Locks,
			accessGovernanceType.AccessControlLockDataInput{
				LockKey: accessGovernanceType.AccessControlLockInheritancelock,
				Details: &accessGovernanceType.AccessControlLockDetailsInput{
					Reason: utils.Ptr(lockMsg),
				},
			},
		)
	}

	if !a.Owners.IsNull() {
		result.Locks = append(result.Locks, accessGovernanceType.AccessControlLockDataInput{
			LockKey: accessGovernanceType.AccessControlLockOwnerlock,
			Details: &accessGovernanceType.AccessControlLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		})
	}

	return diagnostics
}

func (a *AccessControlResourceModel) whoElementsToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *accessGovernanceType.AccessControlInput) (diagnostics diag.Diagnostics) {
	whoItems := a.Who.Elements()

	result.WhoItems = make([]accessGovernanceType.WhoItemInput, 0, len(whoItems))

	for _, whoItem := range whoItems {
		whoObject := whoItem.(types.Object)
		whoAttributes := whoObject.Attributes()

		accessGovernanceWhoItem := accessGovernanceType.WhoItemInput{
			Type: utils.Ptr(accessGovernanceType.AccessWhoItemTypeWhogrant),
		}

		if promiseDurationAttribute, found := whoAttributes["promise_duration"]; found && !promiseDurationAttribute.IsNull() {
			promiseDurationInt := promiseDurationAttribute.(types.Int64)
			accessGovernanceWhoItem.PromiseDuration = promiseDurationInt.ValueInt64Pointer()
			accessGovernanceWhoItem.Type = utils.Ptr(accessGovernanceType.AccessWhoItemTypeWhopromise)
		}

		if userAttribute, found := whoAttributes["user"]; found && !userAttribute.IsNull() {
			userString := userAttribute.(types.String)

			userInformation, err := client.User().GetUserByEmail(ctx, userString.ValueString())
			if err != nil {
				diagnostics.AddError("Failed to get user", err.Error())

				continue
			}

			accessGovernanceWhoItem.User = &userInformation.Id
		} else if accessControlAttribute, found := whoAttributes["access_control"]; found && !accessControlAttribute.IsNull() {
			accessGovernanceWhoItem.AccessControl = accessControlAttribute.(types.String).ValueStringPointer()
		} else {
			diagnostics.AddError("Failed to get who-item", "No user or access control set")

			continue
		}

		result.WhoItems = append(result.WhoItems, accessGovernanceWhoItem)
	}

	return diagnostics
}

func (a *AccessControlResourceModel) whoAbacRuleToAccessControlInput(result *accessGovernanceType.AccessControlInput) (diagnostics diag.Diagnostics) {
	var abacBeRule abac_expression.BinaryExpression

	diagnostics.Append(a.WhoAbacRule.Unmarshal(&abacBeRule)...)

	if diagnostics.HasError() {
		return diagnostics
	}

	rule, err := abacBeRule.ToGqlInput()
	if err != nil {
		diagnostics.AddError("Failed to convert abac-rule to gql", err.Error())

		return
	}

	result.WhoAbacRule = &accessGovernanceType.WhoAbacRuleInput{
		Rule: *rule,
		Type: accessGovernanceType.AccessWhoItemTypeWhogrant,
	}

	return diagnostics
}

func (a *AccessControlResourceModel) FromAccessControl(ac *accessGovernanceType.AccessControl) (diagnostics diag.Diagnostics) {
	a.Id = types.StringValue(ac.Id)
	a.Name = types.StringValue(ac.Name)
	a.Description = types.StringValue(ac.Description)
	a.State = types.StringValue(string(ac.State))

	a.WhoLocked = types.BoolValue(false)
	a.InheritanceLocked = types.BoolValue(false)

	for _, lock := range ac.Locks {
		switch lock.LockKey {
		case accessGovernanceType.AccessControlLockWholock:
			a.WhoLocked = types.BoolValue(true)
		case accessGovernanceType.AccessControlLockInheritancelock:
			a.InheritanceLocked = types.BoolValue(true)
		default:
		}
	}

	return diagnostics
}

func _userPrefix(u string) string {
	return "user:" + u
}

func _accessControlPrefix(a string) string {
	return "access_control:" + a
}

type AccessControlWhatAbacParser struct {
	ResourceFixedDoType []string
}

func (p AccessControlWhatAbacParser) ToAccessControlInput(ctx context.Context, whatAbacRule types.Object, client *sdk.CollibraClient, result *accessGovernanceType.AccessControlInput) (diagnostics diag.Diagnostics) {
	attributes := whatAbacRule.Attributes()

	var doTypes []string

	if len(p.ResourceFixedDoType) > 0 {
		var doDiagnostics diag.Diagnostics

		doTypes, doDiagnostics = utils.StringSetToSlice(ctx, attributes["do_types"].(types.Set))
		diagnostics.Append(doDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}
	} else {
		doTypes = p.ResourceFixedDoType
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

	scopeAttr := attributes["scope"]

	scope := make([]string, 0)

	if !scopeAttr.IsNull() && !scopeAttr.IsUnknown() {
		scopeFullnameItems, scopeDiagnostics := utils.StringSetToSlice(ctx, attributes["scope"].(types.Set))
		diagnostics.Append(scopeDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		for _, scopeFullnameItem := range scopeFullnameItems {
			// Assume that currently only 1 dataSource is provided
			dataSource := result.DataSources[0].DataSource

			id, err := client.DataObject().GetDataObjectIdByName(ctx, scopeFullnameItem, dataSource)
			if err != nil {
				diagnostics.AddError("Failed to get data object id", err.Error())

				return diagnostics
			}

			scope = append(scope, id)
		}
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

	result.WhatType = utils.Ptr(accessGovernanceType.WhoAndWhatTypeDynamic)
	result.WhatAbacRule = &accessGovernanceType.WhatAbacRuleInput{
		DoTypes:           doTypes,
		Permissions:       permissions,
		GlobalPermissions: globalPermissions,
		Scope:             scope,
		Rule:              *abacInput,
	}

	return diagnostics
}

func (p AccessControlWhatAbacParser) ToWhatAbacRuleObject(ctx context.Context, client *sdk.CollibraClient, ac *accessGovernanceType.AccessControl) (_ types.Object, diagnostics diag.Diagnostics) {
	objectTypes := map[string]attr.Type{
		"permissions":        types.SetType{ElemType: types.StringType},
		"global_permissions": types.SetType{ElemType: types.StringType},
		"scope":              types.SetType{ElemType: types.StringType},
		"rule":               jsontypes.NormalizedType{},
	}

	if len(p.ResourceFixedDoType) > 0 {
		objectTypes["do_types"] = types.SetType{ElemType: types.StringType}
	}

	permissions, pDiagnostics := utils.SliceToStringSet(ctx, ac.WhatAbacRule.Permissions)
	diagnostics.Append(pDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	globalPermissions, gpDiagnostics := utils.SliceToStringSet(ctx, ac.WhatAbacRule.GlobalPermissions)
	diagnostics.Append(gpDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	doTypes, dtDiagnostics := utils.SliceToStringSet(ctx, ac.WhatAbacRule.DoTypes)
	diagnostics.Append(dtDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	abacRule := jsontypes.NewNormalizedPointerValue(ac.WhatAbacRule.RuleJson)

	var scopeItems []attr.Value //nolint:prealloc

	for scopeItem, err := range client.AccessControl().GetAccessControlAbacWhatScope(ctx, ac.Id) {
		if err != nil {
			diagnostics.AddError("Failed to load access provider abac scope", err.Error())

			return types.ObjectNull(objectTypes), diagnostics
		}

		scopeItems = append(scopeItems, types.StringValue(scopeItem.FullName))
	}

	objectValue := map[string]attr.Value{
		"do_types":           doTypes,
		"permissions":        permissions,
		"global_permissions": globalPermissions,
		"rule":               abacRule,
	}

	if len(p.ResourceFixedDoType) > 0 {
		scope, scopeDiagnostics := types.SetValue(types.StringType, scopeItems)
		diagnostics.Append(scopeDiagnostics...)

		if diagnostics.HasError() {
			return types.ObjectNull(objectTypes), diagnostics
		}

		objectValue["scope"] = scope
	}

	object, whatAbacDiagnostics := types.ObjectValue(objectTypes, objectValue)

	diagnostics.Append(whatAbacDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	return object, diagnostics
}
