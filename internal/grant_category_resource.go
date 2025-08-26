package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk-go"
	raitoType "github.com/raito-io/sdk-go/types"
)

var _ resource.Resource = (*GrantCategoryResource)(nil)

type GrantCategoryResourceModel struct {
	Id                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	Icon                     types.String `tfsdk:"icon"`
	IsSystem                 types.Bool   `tfsdk:"is_system"`
	IsDefault                types.Bool   `tfsdk:"is_default"`
	CanCreate                types.Bool   `tfsdk:"can_create"`
	AllowDuplicateNames      types.Bool   `tfsdk:"allow_duplicate_names"`
	MultiDataSource          types.Bool   `tfsdk:"multi_data_source"`
	DefaultTypePerDataSource types.Set    `tfsdk:"default_type_per_data_source"`
	AllowedWhoItems          types.Object `tfsdk:"allowed_who_items"`
	AllowedWhatItems         types.Object `tfsdk:"allowed_what_items"`
}

func (m *GrantCategoryResourceModel) ToGrantCategoryInput() raitoType.GrantCategoryInput {
	defaultTypePerDataSourceValues := m.DefaultTypePerDataSource.Elements()
	defaultTypePerDS := make([]raitoType.GrantCategoryTypeForDataSourceInput, 0, len(defaultTypePerDataSourceValues))

	for _, v := range defaultTypePerDataSourceValues {
		vObject := v.(types.Object)
		attributes := vObject.Attributes()

		typeForDataSource := raitoType.GrantCategoryTypeForDataSourceInput{
			DataSource: attributes["data_source"].(types.String).ValueString(),
			Type:       attributes["type"].(types.String).ValueString(),
		}

		defaultTypePerDS = append(defaultTypePerDS, typeForDataSource)
	}

	allowedWhoCategoriesValues := m.AllowedWhoItems.Attributes()["categories"].(types.Set).Elements()
	allowedWhoCategories := make([]string, 0, len(allowedWhoCategoriesValues))

	for _, v := range allowedWhoCategoriesValues {
		allowedWhoCategories = append(allowedWhoCategories, v.(types.String).ValueString())
	}

	input := raitoType.GrantCategoryInput{
		Name:                     m.Name.ValueStringPointer(),
		Description:              m.Description.ValueStringPointer(),
		Icon:                     m.Icon.ValueStringPointer(),
		CanCreate:                m.CanCreate.ValueBoolPointer(),
		AllowDuplicateNames:      m.AllowDuplicateNames.ValueBoolPointer(),
		MultiDataSource:          m.MultiDataSource.ValueBoolPointer(),
		DefaultTypePerDataSource: defaultTypePerDS,
		AllowedWhoItems: &raitoType.GrantCategoryAllowedWhoItemsInput{
			User:        m.AllowedWhoItems.Attributes()["user"].(types.Bool).ValueBool(),
			Group:       m.AllowedWhoItems.Attributes()["group"].(types.Bool).ValueBool(),
			Inheritance: m.AllowedWhoItems.Attributes()["inheritance"].(types.Bool).ValueBool(),
			Self:        m.AllowedWhoItems.Attributes()["self"].(types.Bool).ValueBool(),
			Categories:  allowedWhoCategories,
		},
		AllowedWhatItems: &raitoType.GrantCategoryAllowedWhatItemsInput{
			DataObject: m.AllowedWhatItems.Attributes()["data_object"].(types.Bool).ValueBool(),
		},
	}

	return input
}

type GrantCategoryResource struct {
	client *sdk.RaitoClient
}

func NewGrantCategoryResource() resource.Resource {
	return &GrantCategoryResource{}
}

func (g *GrantCategoryResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_grant_category"
}

func (g *GrantCategoryResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the grant category",
				MarkdownDescription: "The ID of the grant category",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "The name of the grant category",
				MarkdownDescription: "The name of the grant category",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(3)},
			},
			"description": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "The description of the grant category",
				MarkdownDescription: "The description of the grant category",
				Default:             stringdefault.StaticString(""),
			},
			"icon": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "The icon of the grant category",
				MarkdownDescription: "The icon of the grant category",
			},
			"is_system": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Whether the grant category is a system category",
				MarkdownDescription: "Whether the grant category is a system category",
			},
			"is_default": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Whether the grant category is a default category",
				MarkdownDescription: "Whether the grant category is a default category",
			},
			"can_create": schema.BoolAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Whether the user can create grants in this category",
				MarkdownDescription: "Whether the user can create grants in this category",
				Default:             booldefault.StaticBool(true),
			},
			"allow_duplicate_names": schema.BoolAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Whether the user can create grants with duplicate names in this category",
				MarkdownDescription: "Whether the user can create grants with duplicate names in this category",
				Default:             booldefault.StaticBool(true),
			},
			"multi_data_source": schema.BoolAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Whether the grant category supports multiple data sources",
				MarkdownDescription: "Whether the grant category supports multiple data sources",
				Default:             booldefault.StaticBool(true),
			},
			"default_type_per_data_source": schema.SetNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"data_source": schema.StringAttribute{
							Required:            true,
							Optional:            false,
							Computed:            false,
							Sensitive:           false,
							Description:         "The data source for which the default type is set",
							MarkdownDescription: "The data source for which the default type is set",
						},
						"type": schema.StringAttribute{
							Required:            true,
							Optional:            false,
							Computed:            false,
							Sensitive:           false,
							Description:         "The default type for the data source",
							MarkdownDescription: "The default type for the data source",
						},
					},
				},
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "The default category for each data source, type pair",
				MarkdownDescription: "The default category for each data source, type pair",
				Default:             setdefault.StaticValue(types.SetValueMust(types.ObjectType{AttrTypes: map[string]attr.Type{"data_source": types.StringType, "type": types.StringType}}, []attr.Value{})),
			},
			"allowed_who_items": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"user": schema.BoolAttribute{
						Required:            false,
						Optional:            true,
						Computed:            true,
						Sensitive:           false,
						Description:         "Whether the user is allowed as WHO item for the grants of this category",
						MarkdownDescription: "Whether the user is allowed as WHO item for the grants of this category",
						Default:             booldefault.StaticBool(true),
					},
					"group": schema.BoolAttribute{
						Required:            false,
						Optional:            true,
						Computed:            true,
						Sensitive:           false,
						Description:         "Whether the group is allowed as WHO item for the grants of this category",
						MarkdownDescription: "Whether the group is allowed as WHO item for the grants of this category",
						Default:             booldefault.StaticBool(true),
					},
					"inheritance": schema.BoolAttribute{
						Required:            false,
						Optional:            true,
						Computed:            true,
						Sensitive:           false,
						Description:         "Whether the inheritance is allowed as WHO item for the grants of this category",
						MarkdownDescription: "Whether the inheritance is allowed as WHO item for the grants of this category",
						Default:             booldefault.StaticBool(true),
					},
					"self": schema.BoolAttribute{
						Required:            false,
						Optional:            true,
						Computed:            true,
						Sensitive:           false,
						Description:         "Whether the self is allowed as WHO item for the grants of this category",
						MarkdownDescription: "Whether the self is allowed as WHO item for the grants of this category",
						Default:             booldefault.StaticBool(true),
					},
					"categories": schema.SetAttribute{
						ElementType:         types.StringType,
						Required:            false,
						Optional:            true,
						Computed:            true,
						Sensitive:           false,
						Description:         "The allowed WHO items for the grants of this category",
						MarkdownDescription: "The allowed WHO items for the grants of this category",
						Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
					},
				},
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "The allowed WHO items for the grants of this category",
				MarkdownDescription: "The allowed WHO items for the grants of this category",
				Default: objectdefault.StaticValue(types.ObjectValueMust(map[string]attr.Type{
					"user":        types.BoolType,
					"group":       types.BoolType,
					"inheritance": types.BoolType,
					"self":        types.BoolType,
					"categories":  types.SetType{ElemType: types.StringType},
				}, map[string]attr.Value{
					"user":        types.BoolValue(true),
					"group":       types.BoolValue(true),
					"inheritance": types.BoolValue(true),
					"self":        types.BoolValue(true),
					"categories":  types.SetValueMust(types.StringType, []attr.Value{}),
				})),
			},
			"allowed_what_items": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"data_object": schema.BoolAttribute{
						Required:            false,
						Optional:            true,
						Computed:            true,
						Sensitive:           false,
						Description:         "The allowed WHAT items for the grants of this category",
						MarkdownDescription: "The allowed WHAT items for the grants of this category",
						Default:             booldefault.StaticBool(true),
					},
				},
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "The allowed WHAT items for the grants of this category",
				MarkdownDescription: "The allowed WHAT items for the grants of this category",
				Default: objectdefault.StaticValue(types.ObjectValueMust(map[string]attr.Type{
					"data_object": types.BoolType,
				},
					map[string]attr.Value{
						"data_object": types.BoolValue(true),
					},
				)),
			},
		},
		Description:         "The grant category resource allows you to manage grant categories in Raito.",
		MarkdownDescription: "The grant category resource allows you to manage grant categories in Raito.",
		Version:             1,
	}
}

func (g *GrantCategoryResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data GrantCategoryResourceModel

	// Read terraform plan data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	grantCategoryResult, err := g.client.GrantCategory().CreateGrantCategory(ctx, data.ToGrantCategoryInput())
	if err != nil {
		response.Diagnostics.AddError("Failed to create grant category", err.Error())

		return
	}

	data.Id = types.StringValue(grantCategoryResult.Id)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	response.Diagnostics.Append(setGrantCategoryResourceData(grantCategoryResult, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (g *GrantCategoryResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var stateData GrantCategoryResourceModel

	// Read terraform plan stateData into model
	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)

	if response.Diagnostics.HasError() {
		return
	}

	category, err := g.client.GrantCategory().GetGrantCategory(ctx, stateData.Id.ValueString())
	if err != nil {
		var notFoundErr *raitoType.ErrNotFound
		if errors.As(err, &notFoundErr) {
			response.State.RemoveResource(ctx)
		} else {
			response.Diagnostics.AddError("Failed to get grant category", err.Error())
		}

		return
	}

	response.Diagnostics.Append(setGrantCategoryResourceData(category, &stateData)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, stateData)...)
}

func (g *GrantCategoryResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data GrantCategoryResourceModel

	// Read terraform plan data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Update grant category
	gc, err := g.client.GrantCategory().UpdateGrantCategory(ctx, data.Id.ValueString(), data.ToGrantCategoryInput())
	if err != nil {
		response.Diagnostics.AddError("Failed to update grant category", err.Error())

		return
	}

	response.Diagnostics.Append(setGrantCategoryResourceData(gc, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (g *GrantCategoryResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data GrantCategoryResourceModel

	// Read terraform plan stateData into model
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	err := g.client.GrantCategory().DeleteGrantCategory(ctx, data.Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Failed to delete grant category", err.Error())

		return
	}

	response.State.RemoveResource(ctx)
}

func (g *GrantCategoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (g *GrantCategoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func setGrantCategoryResourceData(data *raitoType.GrantCategoryDetails, resp *GrantCategoryResourceModel) (diags diag.Diagnostics) {
	resp.Id = types.StringValue(data.Id)
	resp.Name = types.StringValue(data.Name)
	resp.Description = types.StringValue(data.Description)
	resp.Icon = types.StringValue(data.Icon)
	resp.IsSystem = types.BoolValue(data.IsSystem)
	resp.IsDefault = types.BoolValue(data.IsDefault)
	resp.CanCreate = types.BoolValue(data.CanCreate)
	resp.AllowDuplicateNames = types.BoolValue(data.AllowDuplicateNames)
	resp.MultiDataSource = types.BoolValue(data.MultiDataSource)

	// Default types per DS
	defaultTypesPerDs := make([]attr.Value, 0, len(data.DefaultTypePerDataSource))

	for _, v := range data.DefaultTypePerDataSource {
		objectValue, diag := types.ObjectValue(map[string]attr.Type{
			"data_source": types.StringType,
			"type":        types.StringType,
		},
			map[string]attr.Value{
				"data_source": types.StringValue(v.GetDataSource()),
				"type":        types.StringValue(v.GetType()),
			})

		diags.Append(diag...)

		if diags.HasError() {
			return diags
		}

		defaultTypesPerDs = append(defaultTypesPerDs, objectValue)
	}

	defaultTypesPerDsAttr, diag := types.SetValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"data_source": types.StringType,
			"type":        types.StringType,
		},
	}, defaultTypesPerDs)

	diags.Append(diag...)

	if diags.HasError() {
		return diags
	}

	resp.DefaultTypePerDataSource = defaultTypesPerDsAttr

	// Allowed WHO items
	whoCategoryValues := make([]attr.Value, 0, len(data.AllowedWhoItems.Categories))
	for _, v := range data.AllowedWhoItems.Categories {
		whoCategoryValues = append(whoCategoryValues, types.StringValue(v))
	}

	whoCategories, diag := types.SetValue(types.StringType, whoCategoryValues)

	diags.Append(diag...)

	if diags.HasError() {
		return diags
	}

	allowedWhoItems, diag := types.ObjectValue(
		map[string]attr.Type{
			"user":        types.BoolType,
			"group":       types.BoolType,
			"inheritance": types.BoolType,
			"self":        types.BoolType,
			"categories":  types.SetType{ElemType: types.StringType},
		},
		map[string]attr.Value{
			"user":        types.BoolValue(data.AllowedWhoItems.User),
			"group":       types.BoolValue(data.AllowedWhoItems.Group),
			"inheritance": types.BoolValue(data.AllowedWhoItems.Inheritance),
			"self":        types.BoolValue(data.AllowedWhoItems.Self),
			"categories":  whoCategories,
		},
	)

	diags.Append(diag...)

	if diags.HasError() {
		return diags
	}

	resp.AllowedWhoItems = allowedWhoItems

	// Allowed WHAT items
	allowedWhatItems, diag := types.ObjectValue(
		map[string]attr.Type{
			"data_object": types.BoolType,
		},
		map[string]attr.Value{
			"data_object": types.BoolValue(data.AllowedWhatItems.DataObject),
		},
	)

	diags.Append(diag...)

	if diags.HasError() {
		return diags
	}

	resp.AllowedWhatItems = allowedWhatItems

	return diags
}
