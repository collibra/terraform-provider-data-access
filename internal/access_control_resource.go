package internal

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	sdk "github.com/collibra/data-access-go-sdk"
	"github.com/collibra/data-access-go-sdk/services"
	dataAccessType "github.com/collibra/data-access-go-sdk/types"
	"github.com/collibra/go-set/set"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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

	"github.com/collibra/terraform-provider-data-access/internal/types/abac_expression"
	"github.com/collibra/terraform-provider-data-access/internal/utils"
)

const (
	lockMsg = "Locked by terraform"
)

//
// Model
//

type AccessControlResourceModel struct {
	Id                types.String
	Name              types.String
	Description       types.String
	State             types.String
	Who               types.Set
	WhoAbacRules      types.Set
	WhoLocked         types.Bool
	InheritanceLocked types.Bool
}

type DataObjectReferenceModel struct {
	Type types.String `tfsdk:"type"`
	Path types.List   `tfsdk:"path"`
}

type AccessControlModel[T any] interface {
	*T
	GetAccessControlResourceModel() *AccessControlResourceModel
	SetAccessControlResourceModel(model *AccessControlResourceModel)
	ToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *dataAccessType.AccessControlInput) diag.Diagnostics
	FromAccessControl(ctx context.Context, client *sdk.CollibraClient, input *dataAccessType.AccessControl) diag.Diagnostics
	UpdateOwners(owners types.Set)
	GetOwners() (types.Set, bool)
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

//
// Schema
//

var dataObjectReferenceTypeAttributes = map[string]schema.Attribute{
	"type": schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The type of the data object",
		MarkdownDescription: "The type of the data object",
		Default:             nil,
	},
	"path": schema.ListAttribute{
		ElementType:         types.StringType,
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The path of the data object",
		MarkdownDescription: "The path of the data object",
		Default:             nil,
	},
	"data_source": schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The ID of the data source the data object belongs to",
		MarkdownDescription: "The ID of the data source the data object belongs to",
		Default:             nil,
	},
}

var dataObjectReferenceTypeAttributeTypes = map[string]attr.Type{
	"type": types.StringType,
	"path": types.ListType{
		ElemType: types.StringType,
	},
	"data_source": types.StringType,
}

var dataObjectReferenceType = schema.ObjectAttribute{
	AttributeTypes:      dataObjectReferenceTypeAttributeTypes,
	Required:            true,
	Optional:            false,
	Computed:            false,
	Sensitive:           false,
	Description:         "The reference to the data object",
	MarkdownDescription: "The reference to the data object",
}

type accessControlSchemaOptions struct {
	excludeWho bool
}

func withAccessControlSchemaExcludeWho() func(options *accessControlSchemaOptions) {
	return func(options *accessControlSchemaOptions) {
		options.excludeWho = true
	}
}

func (a *AccessControlResource[T, ApModel]) schema(typeName string, ops ...func(options *accessControlSchemaOptions)) map[string]schema.Attribute {
	options := accessControlSchemaOptions{}

	for _, op := range ops {
		op(&options)
	}

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
			MarkdownDescription: fmt.Sprintf("The state of the %s Possible values are: [%q, %q]", typeName, string(dataAccessType.AccessControlStateActive), string(dataAccessType.AccessControlStateInactive)),
			Validators: []validator.String{
				stringvalidator.OneOf(string(dataAccessType.AccessControlStateActive), string(dataAccessType.AccessControlStateInactive)),
			},
			Default: stringdefault.StaticString(string(dataAccessType.AccessControlStateActive)),
		},
	}

	if !options.excludeWho {
		defaultSchema["who"] = schema.SetNestedAttribute{
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
						Description:         "The ID of the access control in Collibra Data Access",
						MarkdownDescription: "The ID of the access control in Collibra Data Access. Cannot be set if `user` is set.",
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
			MarkdownDescription: fmt.Sprintf("The who-items associated with the %s. When this is not set (nil), the who-list will not be overridden. This is typically used when this should be managed from Collibra Data Access.", typeName),
		}

		defaultSchema["who_abac_rules"] = schema.SetNestedAttribute{
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					// TODO we currently don't support the other attributes in a who abac rule.
					"id": schema.StringAttribute{
						Required:            true,
						Optional:            false,
						Computed:            false,
						Sensitive:           false,
						Description:         "A unique ID of the abac rule within this access control",
						MarkdownDescription: "A unique ID of the abac rule within this access control",
						Default:             nil,
					},
					"rule": schema.StringAttribute{
						CustomType:          jsontypes.NormalizedType{},
						Required:            true,
						Optional:            false,
						Computed:            false,
						Sensitive:           false,
						Description:         "The JSON representation of the abac rule",
						MarkdownDescription: "The JSON representation of the abac rule",
						Default:             nil,
					},
				},
			},
			Optional:            true,
			Required:            false,
			Computed:            false,
			Sensitive:           false,
			Description:         fmt.Sprintf("The abac rules for defining the dynamic who-items associated with the %s", typeName),
			MarkdownDescription: fmt.Sprintf("The abac rules for defining the dynamic who-items associated with the %s", typeName),
		}

		defaultSchema["who_locked"] = schema.BoolAttribute{
			Required:            false,
			Optional:            true,
			Computed:            true,
			Sensitive:           false,
			Description:         "Indicates if who should be locked. This should be true if who users or who_abac_rule is set.",
			MarkdownDescription: "Indicates if who should be locked. This should be true if who users or who_abac_rule is set.",
			Validators:          nil,
		}

		defaultSchema["inheritance_locked"] = schema.BoolAttribute{
			Required:            false,
			Optional:            true,
			Computed:            true,
			Sensitive:           false,
			Description:         "Indicates if who should be locked. This should be true if who access providers are set.",
			MarkdownDescription: "Indicates if who should be locked. This should be true if who access providers are set.",
			Validators:          nil,
		}
	}

	return defaultSchema
}

//
// Actions
//

// Create creates an AccessControl in Collibra from the given terraform model
func (a *AccessControlResource[T, ApModel]) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data T

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	a.create(ctx, &data, response)
}

// create creates an AccessControl in Collibra from the given terraform model
func (a *AccessControlResource[T, ApModel]) create(ctx context.Context, data ApModel, response *resource.CreateResponse) {
	input := dataAccessType.AccessControlInput{}

	apResourceModel := data.GetAccessControlResourceModel()

	state := apResourceModel.State

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

	if owners, ok := data.GetOwners(); ok {
		response.Diagnostics.Append(a.createUpdateOwners(ctx, data, owners, ac, &response.State)...)
	}
}

// createUpdateOwners updates the owners of an AccessControl in Collibra
func (a *AccessControlResource[T, ApModel]) createUpdateOwners(ctx context.Context, data ApModel, owners types.Set, ac *dataAccessType.AccessControl, state *tfsdk.State) (diagnostics diag.Diagnostics) {
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

// updateState updates the state of an AccessControl in Collibra
func (a *AccessControlResource[T, ApModel]) updateState(ctx context.Context, data ApModel, state types.String, ac *dataAccessType.AccessControl) (_ *dataAccessType.AccessControl, diagnostics diag.Diagnostics) {
	if state.Equal(data.GetAccessControlResourceModel().State) {
		return ac, diagnostics
	}

	var err error

	if data.GetAccessControlResourceModel().State.ValueString() == string(dataAccessType.AccessControlStateActive) {
		ac, err = a.client.AccessControl().DeactivateAccessControl(ctx, ac.Id)
		if err != nil {
			diagnostics.AddError("Failed to activate access provider", err.Error())

			return ac, diagnostics
		}
	} else if data.GetAccessControlResourceModel().State.ValueString() == string(dataAccessType.AccessControlStateInactive) {
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

// Read reads the AccessControl from Collibra and created the terraform model
func (a *AccessControlResource[T, ApModel]) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data T

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	a.read(ctx, &data, response, a.readHooks...)
}

// read reads the AccessControl from Collibra and created the terraform model
func (a *AccessControlResource[T, ApModel]) read(ctx context.Context, data ApModel, response *resource.ReadResponse, hooks ...ReadHook[T, ApModel]) {
	apModel := data.GetAccessControlResourceModel()

	// Get the access provider
	ac, err := a.client.AccessControl().GetAccessControl(ctx, apModel.Id.ValueString())
	if err != nil {
		notFoundErr := &dataAccessType.ErrNotFound{}
		if errors.As(err, &notFoundErr) {
			response.State.RemoveResource(ctx)

			return
		}

		response.Diagnostics.AddError("Failed to read access provider", err.Error())

		return
	}

	if ac.State == dataAccessType.AccessControlStateDeleted {
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

		stateWhoItems, done := a.whoItemsToTerraform(ctx, apModel, response, definedPromises, stateWhoItems)
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

	if len(ac.WhoAbacRules) > 0 {
		object, objectDiagnostics := a.abacWhoToTerraform(ac)
		response.Diagnostics.Append(objectDiagnostics...)

		if response.Diagnostics.HasError() {
			return
		}

		apModel.WhoAbacRules = object
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

// abacWhoToTerraform convert the WHO ABAC rules from the AccessControl to terraform types
func (a *AccessControlResource[T, ApModel]) abacWhoToTerraform(ac *dataAccessType.AccessControl) (_ types.Set, diagnostics diag.Diagnostics) {
	whoAbacRuleList := make([]attr.Value, 0, len(ac.WhoAbacRules))

	whoAbacRuleType := map[string]attr.Type{
		"rule": jsontypes.NormalizedType{},
		"id":   types.StringType,
	}
	whoAbacRulesType := types.ObjectType{AttrTypes: whoAbacRuleType}

	for _, rule := range ac.WhoAbacRules {
		abacRule := jsontypes.NewNormalizedPointerValue(rule.RuleJson)

		whoAbacRuleList = append(whoAbacRuleList, types.ObjectValueMust(whoAbacRuleType, map[string]attr.Value{
			"rule": abacRule,
			"id":   types.StringValue(rule.Id),
		}))
	}

	whoAbacRules, whoAbacRulesDiag := types.SetValue(whoAbacRulesType, whoAbacRuleList)

	diagnostics.Append(whoAbacRulesDiag...)

	if diagnostics.HasError() {
		return types.SetNull(whoAbacRulesType), diagnostics
	}

	return whoAbacRules, diagnostics
}

// whoItemsToTerraform converts the who-items from the access control to terraform types
func (a *AccessControlResource[T, ApModel]) whoItemsToTerraform(ctx context.Context, apModel *AccessControlResourceModel, response *resource.ReadResponse, definedPromises set.Set[string], stateWhoItems []attr.Value) ([]attr.Value, bool) {
	whoItems := a.client.AccessControl().GetAccessControlWhoList(ctx, apModel.Id.ValueString())
	for whoItem, err := range whoItems {
		if err != nil {
			response.Diagnostics.AddError("Failed to read who-item from access provider", err.Error())

			return nil, true
		}

		var user, whoAp *string

		switch benificiaryItem := whoItem.Item.(type) {
		case *dataAccessType.AccessWhoItemItemUser:
			user = benificiaryItem.Email
		case *dataAccessType.AccessWhoItemItemAccessControl:
			whoAp = &benificiaryItem.Id
		default:
			response.Diagnostics.AddError("Invalid who-item", fmt.Sprintf("Invalid who-item: %T", benificiaryItem))

			return nil, true
		}

		if whoItem.Type == dataAccessType.AccessWhoItemTypeWhogrant {
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

// Update updates the AccessControl in Collibra from the given terraform model
func (a *AccessControlResource[T, ApModel]) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data T

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	a.update(ctx, &data, response)
}

// update updates the AccessControl in Collibra from the given terraform model and updates the terraform accordingly after
func (a *AccessControlResource[T, ApModel]) update(ctx context.Context, data ApModel, response *resource.UpdateResponse) {
	input := dataAccessType.AccessControlInput{}

	apResourceModel := data.GetAccessControlResourceModel()

	id := apResourceModel.Id.ValueString()
	state := apResourceModel.State

	response.Diagnostics.Append(data.ToAccessControlInput(ctx, a.client, &input)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Check for implemented promises
	definedPromises := set.Set[string]{}

	for _, whoItem := range input.WhoItems {
		if whoItem.Type != nil && *whoItem.Type == dataAccessType.AccessWhoItemTypeWhopromise {
			if whoItem.User != nil {
				definedPromises.Add(_userPrefix(*whoItem.User))
			} else if whoItem.AccessControl != nil {
				definedPromises.Add(_accessControlPrefix(*whoItem.AccessControl))
			}
		}
	}

	if a.retainExistingWhoGrantsForPromises(ctx, id, response, definedPromises, input) {
		return
	}

	// Update access control
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
	if owners, ok := data.GetOwners(); ok {
		response.Diagnostics.Append(a.createUpdateOwners(ctx, data, owners, ac, &response.State)...)
	}
}

// retainExistingWhoGrantsForPromises gets the existing WHO-items from the AccessControl and only keeps the grants that correspond to a defined promise
func (a *AccessControlResource[T, ApModel]) retainExistingWhoGrantsForPromises(ctx context.Context, id string, response *resource.UpdateResponse, definedPromises set.Set[string], input dataAccessType.AccessControlInput) bool {
	whoItems := a.client.AccessControl().GetAccessControlWhoList(ctx, id)
	for whoItem, err := range whoItems {
		if err != nil {
			response.Diagnostics.AddError("Failed to read who-item from access provider", err.Error())

			return true
		}

		if whoItem.Type == dataAccessType.AccessWhoItemTypeWhogrant {
			var key string
			var user, whoAp *string

			switch beneficiaryItem := whoItem.Item.(type) {
			case *dataAccessType.AccessWhoItemItemUser:
				if beneficiaryItem.Email == nil {
					continue
				}

				key = _userPrefix(*beneficiaryItem.Email)
				user = &beneficiaryItem.Id
			case *dataAccessType.AccessWhoItemItemAccessControl:
				key = _accessControlPrefix(beneficiaryItem.Id)
				whoAp = &beneficiaryItem.Id
			default:
				continue
			}

			if definedPromises.Contains(key) {
				input.WhoItems = append(input.WhoItems, dataAccessType.WhoItemInput{
					Type:          utils.Ptr(dataAccessType.AccessWhoItemTypeWhogrant),
					User:          user,
					AccessControl: whoAp,
					ExpiresAt:     whoItem.ExpiresAt,
				})
			}
		}
	}

	return false
}

// Delete deletes the AccessControl in Collibra
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

// Configure configures the client to connect to Collibra
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
	whoAbac := &apResourceModel.WhoAbacRules

	whoUsersDefined := false
	whoAccessControlsDefined := false

	if !who.IsNull() { // For each who-item check if exactly one of user or access_control is set.
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

// readOwners fetches the owners of an AccessControl from Collibra and puts them in the Terraform model
func (a *AccessControlResource[T, ApModel]) readOwners(ctx context.Context, apId string) (_ types.Set, diagnostics diag.Diagnostics) {
	roleAssignments := a.client.Role().ListRoleAssignmentsOnAccessControl(ctx, apId, services.WithRoleAssignmentListFilter(&dataAccessType.RoleAssignmentFilterInput{
		Role: utils.Ptr(ownerRole),
	}))

	var ownerIds []attr.Value

	for roleAssignment, err := range roleAssignments {
		if err != nil {
			diagnostics.AddError("Failed to list role assignments on access provider", err.Error())

			return basetypes.SetValue{}, diagnostics
		}

		switch to := roleAssignment.To.(type) {
		case *dataAccessType.RoleAssignmentToUser:
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

	if whoUsersDefined || (!apResourceModel.WhoAbacRules.IsNull() && len(apResourceModel.WhoAbacRules.Elements()) > 0) {
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

type ToAccessControlInputOptions struct {
	LockOwners bool
}

func WithToAccessControlLockOwners(lock bool) func(options *ToAccessControlInputOptions) {
	return func(options *ToAccessControlInputOptions) {
		options.LockOwners = lock
	}
}

// ToAccessControlInput converts the Terraform model to the Collibra AccessControlInput model
func (a *AccessControlResourceModel) ToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *dataAccessType.AccessControlInput, ops ...func(options *ToAccessControlInputOptions)) (diagnostics diag.Diagnostics) {
	option := ToAccessControlInputOptions{}

	for _, op := range ops {
		op(&option)
	}

	result.Name = a.Name.ValueStringPointer()
	result.Description = a.Description.ValueStringPointer()
	result.Locks = append(result.Locks,
		dataAccessType.AccessControlLockDataInput{
			LockKey: dataAccessType.AccessControlLockNamelock,
			Details: &dataAccessType.AccessControlLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		},
	)

	// Handling the WHO
	if !a.Who.IsNull() && !a.Who.IsUnknown() {
		diagnostics.Append(a.whoElementsToAccessControlInput(ctx, client, result)...)
	}

	if !a.WhoAbacRules.IsNull() && !a.WhoAbacRules.IsUnknown() {
		diagnostics.Append(a.whoAbacRulesToAccessControlInput(result)...)
	}

	if a.WhoLocked.ValueBool() {
		result.Locks = append(result.Locks,
			dataAccessType.AccessControlLockDataInput{
				LockKey: dataAccessType.AccessControlLockWholock,
				Details: &dataAccessType.AccessControlLockDetailsInput{
					Reason: utils.Ptr(lockMsg),
				},
			},
		)
	}

	if a.InheritanceLocked.ValueBool() {
		result.Locks = append(result.Locks,
			dataAccessType.AccessControlLockDataInput{
				LockKey: dataAccessType.AccessControlLockInheritancelock,
				Details: &dataAccessType.AccessControlLockDetailsInput{
					Reason: utils.Ptr(lockMsg),
				},
			},
		)
	}

	if option.LockOwners {
		result.Locks = append(result.Locks, dataAccessType.AccessControlLockDataInput{
			LockKey: dataAccessType.AccessControlLockOwnerlock,
			Details: &dataAccessType.AccessControlLockDetailsInput{
				Reason: utils.Ptr(lockMsg),
			},
		})
	}

	return diagnostics
}

// whoElementsToAccessControlInput converts the WHO-items from the Terraform model to the Collibra AccessControlInput model
func (a *AccessControlResourceModel) whoElementsToAccessControlInput(ctx context.Context, client *sdk.CollibraClient, result *dataAccessType.AccessControlInput) (diagnostics diag.Diagnostics) {
	whoItems := a.Who.Elements()

	result.WhoItems = make([]dataAccessType.WhoItemInput, 0, len(whoItems))

	for _, whoItem := range whoItems {
		whoObject := whoItem.(types.Object)
		whoAttributes := whoObject.Attributes()

		dataAccessWhoItem := dataAccessType.WhoItemInput{
			Type: utils.Ptr(dataAccessType.AccessWhoItemTypeWhogrant),
		}

		if promiseDurationAttribute, found := whoAttributes["promise_duration"]; found && !promiseDurationAttribute.IsNull() {
			promiseDurationInt := promiseDurationAttribute.(types.Int64)
			dataAccessWhoItem.PromiseDuration = promiseDurationInt.ValueInt64Pointer()
			dataAccessWhoItem.Type = utils.Ptr(dataAccessType.AccessWhoItemTypeWhopromise)
		}

		if userAttribute, found := whoAttributes["user"]; found && !userAttribute.IsNull() {
			userString := userAttribute.(types.String)

			userInformation, err := client.User().GetUserByEmail(ctx, userString.ValueString())
			if err != nil {
				diagnostics.AddError("Failed to get user", err.Error())

				continue
			}

			dataAccessWhoItem.User = &userInformation.Id
		} else if accessControlAttribute, found := whoAttributes["access_control"]; found && !accessControlAttribute.IsNull() {
			dataAccessWhoItem.AccessControl = accessControlAttribute.(types.String).ValueStringPointer()
		} else {
			diagnostics.AddError("Failed to get who-item", "No user or access control set")

			continue
		}

		result.WhoItems = append(result.WhoItems, dataAccessWhoItem)
	}

	return diagnostics
}

// whoAbacRulesToAccessControlInput converts the WHO ABAC rules from the Terraform model to the Collibra AccessControlInput model
func (a *AccessControlResourceModel) whoAbacRulesToAccessControlInput(result *dataAccessType.AccessControlInput) (diagnostics diag.Diagnostics) {
	whoAbacRuleItems := a.WhoAbacRules.Elements()

	result.WhoAbacRules = make([]*dataAccessType.WhoAbacRuleInput, 0, len(whoAbacRuleItems))

	for _, whoAbacRuleItem := range whoAbacRuleItems {
		abacRuleObject := whoAbacRuleItem.(types.Object)
		attributes := abacRuleObject.Attributes()

		abacInput, ruleDiag := abacRuleToGqlInput(attributes, "rule")
		if ruleDiag.HasError() {
			return ruleDiag
		}

		result.WhoAbacRules = append(result.WhoAbacRules, &dataAccessType.WhoAbacRuleInput{
			Rule: *abacInput,
			Type: dataAccessType.AccessWhoItemTypeWhogrant,
			Id:   getOptionalString(attributes, "id"),
		})
	}

	return diagnostics
}

// FromAccessControl converts the Collibra AccessControl model to the Terraform model
func (a *AccessControlResourceModel) FromAccessControl(ac *dataAccessType.AccessControl) (diagnostics diag.Diagnostics) {
	a.Id = types.StringValue(ac.Id)
	a.Name = types.StringValue(ac.Name)
	a.Description = types.StringValue(ac.Description)
	a.State = types.StringValue(string(ac.State))

	a.WhoLocked = types.BoolValue(false)
	a.InheritanceLocked = types.BoolValue(false)

	for _, lock := range ac.Locks {
		switch lock.LockKey {
		case dataAccessType.AccessControlLockWholock:
			a.WhoLocked = types.BoolValue(true)
		case dataAccessType.AccessControlLockInheritancelock:
			a.InheritanceLocked = types.BoolValue(true)
		default:
		}
	}

	return diagnostics
}

// dataSourcesToAccessControlInput converts the data sources from the Terraform model to the Collibra AccessControlInput model
func dataSourcesToAccessControlInput(dataSources types.Set, result *dataAccessType.AccessControlInput) {
	if !dataSources.IsNull() && !dataSources.IsUnknown() {
		dataSourceElements := dataSources.Elements()

		result.DataSources = make([]dataAccessType.AccessControlDataSourceInput, 0, len(dataSourceElements))

		for _, dsElement := range dataSourceElements {
			dsAttributes := dsElement.(types.Object).Attributes()

			var apType *string

			if !dsAttributes["type"].(types.String).IsUnknown() {
				apType = dsAttributes["type"].(types.String).ValueStringPointer()
			}

			result.DataSources = append(result.DataSources, dataAccessType.AccessControlDataSourceInput{
				DataSource: dsAttributes["data_source"].(types.String).ValueString(),
				Type:       apType,
			})
		}
	}
}

// abacRuleToGqlInput converts a JSON ABAC rule from Terraform to a Collibra GraphQL input object
func abacRuleToGqlInput(attributes map[string]attr.Value, field string) (_ *dataAccessType.AbacComparisonExpressionInput, diagnostics diag.Diagnostics) {
	jsonRule := attributes[field].(jsontypes.Normalized)

	var abacRule abac_expression.BinaryExpression
	diagnostics.Append(jsonRule.Unmarshal(&abacRule)...)

	if diagnostics.HasError() {
		return nil, diagnostics
	}

	abacInput, err := abacRule.ToGqlInput()
	if err != nil {
		diagnostics.AddError("Failed to convert abac rule to gql input", err.Error())

		return nil, diagnostics
	}

	return abacInput, diagnostics
}

func getOptionalString(attributes map[string]attr.Value, field string) *string {
	var val *string
	if !attributes[field].IsNull() {
		val = utils.Ptr(attributes[field].(types.String).ValueString())
	}

	return val
}

func _userPrefix(u string) string {
	return "user:" + u
}

func _accessControlPrefix(a string) string {
	return "access_control:" + a
}
