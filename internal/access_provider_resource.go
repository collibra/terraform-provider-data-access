package internal

import (
	"context"
	"errors"
	"fmt"
	"regexp"

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
	"github.com/raito-io/sdk-go"
	"github.com/raito-io/sdk-go/services"
	raitoType "github.com/raito-io/sdk-go/types"
	"github.com/raito-io/sdk-go/types/models"

	"github.com/raito-io/terraform-provider-raito/internal/types/abac_expression"
	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

const (
	lockMsg = "Locked by terraform"
)

type AccessProviderResourceModel struct {
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

type AccessProviderModel[T any] interface {
	*T
	GetAccessProviderResourceModel() *AccessProviderResourceModel
	SetAccessProviderResourceModel(model *AccessProviderResourceModel)
	ToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) diag.Diagnostics
	FromAccessProvider(ctx context.Context, client *sdk.RaitoClient, input *raitoType.AccessProvider) diag.Diagnostics
	UpdateOwners(owners types.Set)
}

type ReadHook[T any, ApModel AccessProviderModel[T]] func(ctx context.Context, client *sdk.RaitoClient, data ApModel) diag.Diagnostics
type ValidationHook[T any, ApModel AccessProviderModel[T]] func(ctx context.Context, data ApModel) diag.Diagnostics
type PlanModifierHook[T any, ApModel AccessProviderModel[T]] func(ctx context.Context, data ApModel) (ApModel, diag.Diagnostics)

type AccessProviderResource[T any, ApModel AccessProviderModel[T]] struct {
	client *sdk.RaitoClient

	readHooks         []ReadHook[T, ApModel]
	validationHooks   []ValidationHook[T, ApModel]
	planModifierHooks []PlanModifierHook[T, ApModel]
}

func (a *AccessProviderResource[T, ApModel]) schema(typeName string) map[string]schema.Attribute {
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
			MarkdownDescription: fmt.Sprintf("The state of the %s Possible values are: [%q, %q]", typeName, models.AccessProviderStateActive.String(), models.AccessProviderStateInactive.String()),
			Validators: []validator.String{
				stringvalidator.OneOf(models.AccessProviderStateActive.String(), models.AccessProviderStateInactive.String()),
			},
			Default: stringdefault.StaticString(models.AccessProviderStateActive.String()),
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
						MarkdownDescription: "The email address of the user. This cannot be set if `group` or `access_control` is set.",
						Validators: []validator.String{
							stringvalidator.RegexMatches(regexp.MustCompile(`.+@.+\..+`), "value must be a valid email address"),
						},
					},
					"group": schema.StringAttribute{
						Required:            false,
						Optional:            true,
						Computed:            false,
						Sensitive:           false,
						Description:         "The ID of the group in Raito Cloud",
						MarkdownDescription: "The ID of the group in Raito Cloud. This cannot be set if `user` or `access_control` is set.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(3),
						},
					},
					"access_control": schema.StringAttribute{
						Required:            false,
						Optional:            true,
						Computed:            false,
						Sensitive:           false,
						Description:         "The ID of the access control in Raito Cloud",
						MarkdownDescription: "The ID of the access control in Raito Cloud. Cannot be set if `user` or `group` is set.",
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
			Description:         "Indicates if who should be locked. This should be true if who users, who groups, or who_abac_rule is set.",
			MarkdownDescription: "Indicates if who should be locked. This should be true if who users, who groups, or who_abac_rule is set.",
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

func (a *AccessProviderResource[T, ApModel]) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data T

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	a.create(ctx, &data, response)
}

func (a *AccessProviderResource[T, ApModel]) create(ctx context.Context, data ApModel, response *resource.CreateResponse) {
	input := raitoType.AccessProviderInput{}

	apResourceModel := data.GetAccessProviderResourceModel()

	state := apResourceModel.State
	owners := apResourceModel.Owners

	response.Diagnostics.Append(data.ToAccessProviderInput(ctx, a.client, &input)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Create the access provider
	ap, err := a.client.AccessProvider().CreateAccessProvider(ctx, input)
	if err != nil {
		response.Diagnostics.AddError("Failed to create access provider", err.Error())

		return
	}

	response.Diagnostics.Append(data.FromAccessProvider(ctx, a.client, ap)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	if response.Diagnostics.HasError() {
		return
	}

	ap, diagnostics := a.updateState(ctx, data, state, ap)

	response.Diagnostics.Append(diagnostics...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(data.FromAccessProvider(ctx, a.client, ap)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(a.createUpdateOwners(ctx, data, owners, ap, &response.State)...)
}

func (a *AccessProviderResource[T, ApModel]) createUpdateOwners(ctx context.Context, data ApModel, owners types.Set, ap *raitoType.AccessProvider, state *tfsdk.State) (diagnostics diag.Diagnostics) {
	if !owners.IsNull() && !owners.IsUnknown() {
		ownerElements := owners.Elements()

		ownerIds := make([]string, len(ownerElements))
		for i, ownerElement := range ownerElements {
			ownerIds[i] = ownerElement.(types.String).ValueString()
		}

		_, err := a.client.Role().UpdateRoleAssigneesOnAccessProvider(ctx, ap.Id, ownerRole, ownerIds...)
		if err != nil {
			diagnostics.AddError("Failed to update owners of access provider", err.Error())

			return diagnostics
		}
	} else {
		ownerSet, ownerDiagnostics := a.readOwners(ctx, ap.Id)
		diagnostics.Append(ownerDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		data.UpdateOwners(ownerSet)
		diagnostics.Append(state.Set(ctx, data)...)
	}

	return diagnostics
}

func (a *AccessProviderResource[T, ApModel]) updateState(ctx context.Context, data ApModel, state types.String, ap *raitoType.AccessProvider) (_ *raitoType.AccessProvider, diagnostics diag.Diagnostics) {
	if state.Equal(data.GetAccessProviderResourceModel().State) {
		return ap, diagnostics
	}

	var err error

	if data.GetAccessProviderResourceModel().State.ValueString() == models.AccessProviderStateActive.String() {
		ap, err = a.client.AccessProvider().DeactivateAccessProvider(ctx, ap.Id)
		if err != nil {
			diagnostics.AddError("Failed to activate access provider", err.Error())

			return ap, diagnostics
		}
	} else if data.GetAccessProviderResourceModel().State.ValueString() == models.AccessProviderStateInactive.String() {
		ap, err = a.client.AccessProvider().ActivateAccessProvider(ctx, ap.Id)
		if err != nil {
			diagnostics.AddError("Failed to deactivate access provider", err.Error())

			return ap, diagnostics
		}
	} else {
		diagnostics.AddError("Invalid state", fmt.Sprintf("Invalid state: %s", data.GetAccessProviderResourceModel().State.ValueString()))

		return ap, diagnostics
	}

	return ap, diagnostics
}

func (a *AccessProviderResource[T, ApModel]) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data T

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	a.read(ctx, &data, response, a.readHooks...)
}

func (a *AccessProviderResource[T, ApModel]) read(ctx context.Context, data ApModel, response *resource.ReadResponse, hooks ...ReadHook[T, ApModel]) {
	apModel := data.GetAccessProviderResourceModel()

	// Get the access provider
	ap, err := a.client.AccessProvider().GetAccessProvider(ctx, apModel.Id.ValueString())
	if err != nil {
		notFoundErr := &raitoType.ErrNotFound{}
		if errors.As(err, &notFoundErr) {
			response.State.RemoveResource(ctx)

			return
		}

		response.Diagnostics.AddError("Failed to read access provider", err.Error())

		return
	}

	if ap.State == models.AccessProviderStateDeleted {
		response.State.RemoveResource(ctx)

		return
	}

	response.Diagnostics.Append(data.FromAccessProvider(ctx, a.client, ap)...)

	if response.Diagnostics.HasError() {
		return
	}

	apModel = data.GetAccessProviderResourceModel()

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
				} else if !attributes["group"].IsNull() {
					definedPromises.Add(_groupPrefix(attributes["group"].(types.String).ValueString()))
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
				"group":            types.StringType,
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

	if !apModel.Who.IsNull() && ap.WhoAbacRule != nil {
		apModel.WhoAbacRule = jsontypes.NewNormalizedPointerValue(ap.WhoAbacRule.RuleJson)
	}

	// Set all global access provider attributes
	data.SetAccessProviderResourceModel(apModel)

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

func (a *AccessProviderResource[T, ApModel]) readWhoItems(ctx context.Context, apModel *AccessProviderResourceModel, response *resource.ReadResponse, definedPromises set.Set[string], stateWhoItems []attr.Value) ([]attr.Value, bool) {
	// Get all who-items. Ignore implemented promises.
	cancelCtx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	whoItems := a.client.AccessProvider().GetAccessProviderWhoList(cancelCtx, apModel.Id.ValueString())
	for whoItem := range whoItems {
		if whoItem.HasError() {
			response.Diagnostics.AddError("Failed to read who-item from access provider", whoItem.GetError().Error())

			return nil, true
		}

		var user, group, whoAp *string

		item := whoItem.GetItem()
		switch benificiaryItem := item.Item.(type) {
		case *raitoType.AccessProviderWhoListItemItemUser:
			user = benificiaryItem.Email
		case *raitoType.AccessProviderWhoListItemItemGroup:
			group = &benificiaryItem.Id
		case *raitoType.AccessProviderWhoListItemItemAccessProvider:
			whoAp = &benificiaryItem.Id
		default:
			response.Diagnostics.AddError("Invalid who-item", fmt.Sprintf("Invalid who-item: %T", benificiaryItem))

			return nil, true
		}

		if item.Type == raitoType.AccessWhoItemTypeWhogrant {
			if (user != nil && definedPromises.Contains(_userPrefix(*user))) || (group != nil && definedPromises.Contains(_groupPrefix(*group))) || (whoAp != nil && definedPromises.Contains(_accessControlPrefix(*whoAp))) {
				continue
			}
		} else if item.PromiseDuration == nil {
			response.Diagnostics.AddError("Invalid who-item detected.", "Invalid who-item. Promise duration not set on promise who-item")
		}

		stateWhoItems = append(stateWhoItems, types.ObjectValueMust(
			map[string]attr.Type{
				"user":             types.StringType,
				"group":            types.StringType,
				"access_control":   types.StringType,
				"promise_duration": types.Int64Type,
			}, map[string]attr.Value{
				"user":             types.StringPointerValue(user),
				"group":            types.StringPointerValue(group),
				"access_control":   types.StringPointerValue(whoAp),
				"promise_duration": types.Int64PointerValue(item.PromiseDuration),
			}))
	}

	return stateWhoItems, false
}

func (a *AccessProviderResource[T, ApModel]) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data T

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	a.update(ctx, &data, response)
}

func (a *AccessProviderResource[T, ApModel]) update(ctx context.Context, data ApModel, response *resource.UpdateResponse) {
	input := raitoType.AccessProviderInput{}

	apResourceModel := data.GetAccessProviderResourceModel()

	id := apResourceModel.Id.ValueString()
	state := apResourceModel.State
	owners := apResourceModel.Owners

	response.Diagnostics.Append(data.ToAccessProviderInput(ctx, a.client, &input)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Check for implemented promises
	definedPromises := set.Set[string]{}

	for _, whoItem := range input.WhoItems {
		if whoItem.Type != nil && *whoItem.Type == raitoType.AccessWhoItemTypeWhopromise {
			if whoItem.User != nil {
				definedPromises.Add(_userPrefix(*whoItem.User))
			} else if whoItem.Group != nil {
				definedPromises.Add(_groupPrefix(*whoItem.Group))
			} else if whoItem.AccessProvider != nil {
				definedPromises.Add(_accessControlPrefix(*whoItem.AccessProvider))
			}
		}
	}

	if a.updateGetWhoItems(ctx, id, response, definedPromises, input) {
		return
	}

	// Update access provider
	ap, err := a.client.AccessProvider().UpdateAccessProvider(ctx, id, input, services.WithAccessProviderOverrideLocks())
	if err != nil {
		response.Diagnostics.AddError("Failed to update access provider", err.Error())

		return
	}

	response.Diagnostics.Append(data.FromAccessProvider(ctx, a.client, ap)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	if response.Diagnostics.HasError() {
		return
	}

	ap, diagnostics := a.updateState(ctx, data, state, ap)

	response.Diagnostics.Append(diagnostics...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(data.FromAccessProvider(ctx, a.client, ap)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Update owners
	response.Diagnostics.Append(a.createUpdateOwners(ctx, data, owners, ap, &response.State)...)
}

func (a *AccessProviderResource[T, ApModel]) updateGetWhoItems(ctx context.Context, id string, response *resource.UpdateResponse, definedPromises set.Set[string], input raitoType.AccessProviderInput) bool {
	cancelCtx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	whoItemChannel := a.client.AccessProvider().GetAccessProviderWhoList(cancelCtx, id)
	for whoItem := range whoItemChannel {
		if whoItem.HasError() {
			response.Diagnostics.AddError("Failed to read who-item from access provider", whoItem.GetError().Error())

			return true
		}

		item := whoItem.GetItem()

		if item.Type == raitoType.AccessWhoItemTypeWhogrant {
			var key string
			var user, group, whoAp *string

			switch beneficiaryItem := item.Item.(type) {
			case *raitoType.AccessProviderWhoListItemItemUser:
				if beneficiaryItem.Email == nil {
					continue
				}

				key = _userPrefix(*beneficiaryItem.Email)
				user = &beneficiaryItem.Id
			case *raitoType.AccessProviderWhoListItemItemGroup:
				key = _groupPrefix(beneficiaryItem.Id)
				group = &beneficiaryItem.Id
			case *raitoType.AccessProviderWhoListItemItemAccessProvider:
				key = _accessControlPrefix(beneficiaryItem.Id)
				whoAp = &beneficiaryItem.Id
			default:
				continue
			}

			if definedPromises.Contains(key) {
				input.WhoItems = append(input.WhoItems, raitoType.WhoItemInput{
					Type:           utils.Ptr(raitoType.AccessWhoItemTypeWhogrant),
					User:           user,
					Group:          group,
					AccessProvider: whoAp,
					ExpiresAfter:   item.ExpiresAfter,
					ExpiresAt:      item.ExpiresAt,
				})
			}
		}
	}

	return false
}

func (a *AccessProviderResource[T, ApModel]) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data T

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	apModel := ApModel(&data)

	err := a.client.AccessProvider().DeleteAccessProvider(ctx, apModel.GetAccessProviderResourceModel().Id.ValueString(), services.WithAccessProviderOverrideLocks())
	if err != nil {
		response.Diagnostics.AddError("Failed to delete access provider", err.Error())

		return
	}

	response.State.RemoveResource(ctx)
}

func (a *AccessProviderResource[T, ApModel]) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	a.client = client
}

func (a *AccessProviderResource[T, ApModel]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (a *AccessProviderResource[T, ApModel]) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var data T

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	apModel := ApModel(&data)

	apResourceModel := apModel.GetAccessProviderResourceModel()

	who := &apResourceModel.Who
	whoAbac := &apResourceModel.WhoAbacRule

	whoGroupsOrUsersDefined := false
	whoAccessProvidersDefined := false

	if !who.IsNull() && !whoAbac.IsNull() {
		response.Diagnostics.AddError(
			"Cannot specify both who and who_abac",
			"Please specify only one of who or who_abac",
		)
	} else if !who.IsNull() { // For each who-item check if exactly one of user, group or access_control is set.
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

			attrFn("user", &whoGroupsOrUsersDefined)
			attrFn("group", &whoGroupsOrUsersDefined)
			attrFn("access_control", &whoAccessProvidersDefined)

			if attributesFound != 1 {
				response.Diagnostics.AddError(
					"Invalid who-item. Exactly one of user, group or access_control must be set.",
					fmt.Sprintf("Expected exactly one of user, group or access_control, got: %d.", attributesFound),
				)

				break
			}
		}
	}

	if whoGroupsOrUsersDefined || !whoAbac.IsNull() {
		if !apResourceModel.WhoLocked.IsNull() && !apResourceModel.WhoLocked.ValueBool() {
			response.Diagnostics.AddError("Who must be locked", "Who must be locked if who users, who groups or who_abac_rule is set.")
		}
	}

	if whoAccessProvidersDefined {
		if !apResourceModel.InheritanceLocked.IsNull() && !apResourceModel.InheritanceLocked.ValueBool() {
			response.Diagnostics.AddError("Inheritance must be locked", "Inheritance must be locked if who access providers are set.")
		}
	}

	for _, validatorHook := range a.validationHooks {
		response.Diagnostics.Append(validatorHook(ctx, apModel)...)
	}
}

func (a *AccessProviderResource[T, ApModel]) readOwners(ctx context.Context, apId string) (_ types.Set, diagnostics diag.Diagnostics) {
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	roleAssignments := a.client.Role().ListRoleAssignmentsOnAccessProvider(cancelCtx, apId, services.WithRoleAssignmentListFilter(&raitoType.RoleAssignmentFilterInput{
		Role: utils.Ptr(ownerRole),
	}))

	var ownerIds []attr.Value

	for roleAssignment := range roleAssignments {
		if roleAssignment.HasError() {
			diagnostics.AddError("Failed to list role assignments on access provider", roleAssignment.GetError().Error())

			return basetypes.SetValue{}, diagnostics
		}

		switch to := roleAssignment.GetItem().To.(type) {
		case *raitoType.RoleAssignmentToUser:
			ownerIds = append(ownerIds, types.StringValue(to.Id))
		case *raitoType.RoleAssignmentToGroup:
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

func (a *AccessProviderResource[T, ApModel]) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
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
	apResourceModel := apModel.GetAccessProviderResourceModel()

	whoGroupsOrUsersDefined := false
	whoAccessProvidersDefined := false

	if !apResourceModel.Who.IsNull() {
		for _, whoItem := range apResourceModel.Who.Elements() {
			whoItemAttribute := whoItem.(types.Object)

			attributes := whoItemAttribute.Attributes()

			attrFn := func(key string, indicator *bool) {
				if attribute, found := attributes[key]; found && !attribute.IsNull() {
					*indicator = true
				}
			}

			attrFn("user", &whoGroupsOrUsersDefined)
			attrFn("group", &whoGroupsOrUsersDefined)
			attrFn("access_control", &whoAccessProvidersDefined)
		}
	}

	if whoGroupsOrUsersDefined || !apResourceModel.WhoAbacRule.IsNull() {
		apResourceModel.WhoLocked = types.BoolValue(true)
	} else if apResourceModel.WhoLocked.IsUnknown() {
		apResourceModel.WhoLocked = types.BoolValue(false)
	}

	if whoAccessProvidersDefined {
		apResourceModel.InheritanceLocked = types.BoolValue(true)
	} else if apResourceModel.InheritanceLocked.IsUnknown() {
		apResourceModel.InheritanceLocked = types.BoolValue(false)
	}

	apModel.SetAccessProviderResourceModel(apResourceModel)

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

func (a *AccessProviderResourceModel) ToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) (diagnostics diag.Diagnostics) {
	result.Name = a.Name.ValueStringPointer()
	result.Description = a.Description.ValueStringPointer()
	result.Locks = append(result.Locks,
		raitoType.AccessProviderLockDataInput{
			LockKey: raitoType.AccessProviderLockNamelock,
			Details: &raitoType.AccessProviderLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		},
	)

	result.WhoType = utils.Ptr(raitoType.WhoAndWhatTypeStatic)

	if !a.Who.IsNull() && !a.Who.IsUnknown() {
		diagnostics.Append(a.whoElementsToAccessProviderInput(ctx, client, result)...)
	} else if !a.WhoAbacRule.IsNull() && !a.WhoAbacRule.IsUnknown() {
		result.WhoType = utils.Ptr(raitoType.WhoAndWhatTypeDynamic)
		diagnostics.Append(a.whoAbacRuleToAccessProviderInput(result)...)
	}

	if a.WhoLocked.ValueBool() {
		result.Locks = append(result.Locks,
			raitoType.AccessProviderLockDataInput{
				LockKey: raitoType.AccessProviderLockWholock,
				Details: &raitoType.AccessProviderLockDetailsInput{
					Reason: utils.Ptr(lockMsg),
				},
			},
		)
	}

	if a.InheritanceLocked.ValueBool() {
		result.Locks = append(result.Locks,
			raitoType.AccessProviderLockDataInput{
				LockKey: raitoType.AccessProviderLockInheritancelock,
				Details: &raitoType.AccessProviderLockDetailsInput{
					Reason: utils.Ptr(lockMsg),
				},
			},
		)
	}

	if !a.Owners.IsNull() {
		result.Locks = append(result.Locks, raitoType.AccessProviderLockDataInput{
			LockKey: raitoType.AccessProviderLockOwnerlock,
			Details: &raitoType.AccessProviderLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		})
	}

	return diagnostics
}

func (a *AccessProviderResourceModel) whoElementsToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) (diagnostics diag.Diagnostics) {
	whoItems := a.Who.Elements()

	result.WhoItems = make([]raitoType.WhoItemInput, 0, len(whoItems))

	for _, whoItem := range whoItems {
		whoObject := whoItem.(types.Object)
		whoAttributes := whoObject.Attributes()

		raitoWhoItem := raitoType.WhoItemInput{
			Type: utils.Ptr(raitoType.AccessWhoItemTypeWhogrant),
		}

		if promiseDurationAttribute, found := whoAttributes["promise_duration"]; found && !promiseDurationAttribute.IsNull() {
			promiseDurationInt := promiseDurationAttribute.(types.Int64)
			raitoWhoItem.PromiseDuration = promiseDurationInt.ValueInt64Pointer()
			raitoWhoItem.Type = utils.Ptr(raitoType.AccessWhoItemTypeWhopromise)
		}

		if userAttribute, found := whoAttributes["user"]; found && !userAttribute.IsNull() {
			userString := userAttribute.(types.String)

			userInformation, err := client.User().GetUserByEmail(ctx, userString.ValueString())
			if err != nil {
				diagnostics.AddError("Failed to get user", err.Error())

				continue
			}

			raitoWhoItem.User = &userInformation.Id
		} else if groupAttribute, found := whoAttributes["group"]; found && !groupAttribute.IsNull() {
			raitoWhoItem.Group = groupAttribute.(types.String).ValueStringPointer()
		} else if accessControlAttribute, found := whoAttributes["access_control"]; found && !accessControlAttribute.IsNull() {
			raitoWhoItem.AccessProvider = accessControlAttribute.(types.String).ValueStringPointer()
		} else {
			diagnostics.AddError("Failed to get who-item", "No user, group, or access control set")

			continue
		}

		result.WhoItems = append(result.WhoItems, raitoWhoItem)
	}

	return diagnostics
}

func (a *AccessProviderResourceModel) whoAbacRuleToAccessProviderInput(result *raitoType.AccessProviderInput) (diagnostics diag.Diagnostics) {
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

	result.WhoAbacRule = &raitoType.WhoAbacRuleInput{
		Rule: *rule,
		Type: raitoType.AccessWhoItemTypeWhogrant,
	}

	return diagnostics
}

func (a *AccessProviderResourceModel) FromAccessProvider(ap *raitoType.AccessProvider) (diagnostics diag.Diagnostics) {
	a.Id = types.StringValue(ap.Id)
	a.Name = types.StringValue(ap.Name)
	a.Description = types.StringValue(ap.Description)
	a.State = types.StringValue(ap.State.String())

	a.WhoLocked = types.BoolValue(false)
	a.InheritanceLocked = types.BoolValue(false)

	for _, lock := range ap.Locks {
		switch lock.LockKey {
		case raitoType.AccessProviderLockWholock:
			a.WhoLocked = types.BoolValue(true)
		case raitoType.AccessProviderLockInheritancelock:
			a.InheritanceLocked = types.BoolValue(true)
		default:
		}
	}

	return diagnostics
}

func _userPrefix(u string) string {
	return "user:" + u
}

func _groupPrefix(g string) string {
	return "group:" + g
}

func _accessControlPrefix(a string) string {
	return "access_control:" + a
}

type AccessProviderWhatAbacParser struct {
	ResourceFixedDoType []string
}

func (p AccessProviderWhatAbacParser) ToAccessProviderInput(ctx context.Context, whatAbacRule types.Object, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) (diagnostics diag.Diagnostics) {
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

func (p AccessProviderWhatAbacParser) ToWhatAbacRuleObject(ctx context.Context, client *sdk.RaitoClient, ap *raitoType.AccessProvider) (_ types.Object, diagnostics diag.Diagnostics) {
	objectTypes := map[string]attr.Type{
		"permissions":        types.SetType{ElemType: types.StringType},
		"global_permissions": types.SetType{ElemType: types.StringType},
		"scope":              types.SetType{ElemType: types.StringType},
		"rule":               jsontypes.NormalizedType{},
	}

	if len(p.ResourceFixedDoType) > 0 {
		objectTypes["do_types"] = types.SetType{ElemType: types.StringType}
	}

	permissions, pDiagnostics := utils.SliceToStringSet(ctx, ap.WhatAbacRule.Permissions)
	diagnostics.Append(pDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	globalPermissions, gpDiagnostics := utils.SliceToStringSet(ctx, ap.WhatAbacRule.GlobalPermissions)
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

		scopeItems = append(scopeItems, types.StringValue(scopeItem.MustGetItem().FullName))
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
