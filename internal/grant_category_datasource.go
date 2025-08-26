package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk-go"
	types2 "github.com/raito-io/sdk-go/types"
)

var _ datasource.DataSource = (*GrantCategoryDataSource)(nil)

type GrantCategoryDataSourceModel struct {
	Id                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	IsSystem                 types.Bool   `tfsdk:"is_system"`
	IsDefault                types.Bool   `tfsdk:"is_default"`
	CanCreate                types.Bool   `tfsdk:"can_create"`
	AllowDuplicateNames      types.Bool   `tfsdk:"allow_duplicate_names"`
	MultiDataSource          types.Bool   `tfsdk:"multi_data_source"`
	DefaultTypePerDataSource types.Set    `tfsdk:"default_type_per_data_source"`
	AllowedWhoItems          types.Object `tfsdk:"allowed_who_items"`
	AllowedWhatItems         types.Object `tfsdk:"allowed_what_items"`
}

type GrantCategoryDataSource struct {
	client *sdk.RaitoClient
}

func NewGrantCategoryDataSource() datasource.DataSource {
	return &GrantCategoryDataSource{}
}

func (g *GrantCategoryDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_grant_category"
}

func (g *GrantCategoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the requested grant category",
				MarkdownDescription: "The ID of the requested grant category",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "The name of the requested grant category",
				MarkdownDescription: "The name of the requested grant category",
			},
			"description": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The description of the grant category",
				MarkdownDescription: "The description of the grant category",
			},
			"is_system": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Indicates if the grant category is a system category",
				MarkdownDescription: "Indicates if the grant category is a system category",
			},
			"is_default": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Indicates if the grant category is the default category",
				MarkdownDescription: "Indicates if the grant category is the default category",
			},
			"can_create": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Indicates if grants of this category can be created",
				MarkdownDescription: "Indicates if grants of this category can be created",
			},
			"allow_duplicate_names": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Indicates if duplicate names are allowed for grants of this category",
				MarkdownDescription: "Indicates if duplicate names are allowed for grants of this category",
			},
			"multi_data_source": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Indicates if APs of this category can have multiple data sources",
				MarkdownDescription: "Indicates if APs of this category can have multiple data sources",
			},
			"default_type_per_data_source": schema.SetNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"data_source": schema.StringAttribute{
							Required:            false,
							Optional:            false,
							Computed:            true,
							Sensitive:           false,
							Description:         "Data source ID for which the default type is the defined grant category",
							MarkdownDescription: "Data source ID for which the default type is the defined grant category",
						},
						"type": schema.StringAttribute{
							Required:            false,
							Optional:            false,
							Computed:            true,
							Sensitive:           false,
							Description:         "Types for which this grant category is the default for the defined data source",
							MarkdownDescription: "Types for which this grant category is the default for the defined data source",
						},
					},
				},
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "List of data sources and types for which the grant category is the default",
				MarkdownDescription: "List of data sources and types for which the grant category is the default",
			},
			"allowed_who_items": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"user": schema.BoolAttribute{
						Required:            false,
						Optional:            false,
						Computed:            true,
						Sensitive:           false,
						Description:         "Indicates if a user is allowed as a WHO item",
						MarkdownDescription: "Indicates if a user is allowed as a WHO item",
					},
					"group": schema.BoolAttribute{
						Required:            false,
						Optional:            false,
						Computed:            true,
						Sensitive:           false,
						Description:         "Indicates if a group is allowed as a WHO item",
						MarkdownDescription: "Indicates if a group is allowed as a WHO item",
					},
					"inheritance": schema.BoolAttribute{
						Required:            false,
						Optional:            false,
						Computed:            true,
						Sensitive:           false,
						Description:         "Indicates if inheritance is allowed as a WHO item",
						MarkdownDescription: "Indicates if inheritance is allowed as a WHO item",
					},
					"self": schema.BoolAttribute{
						Required:            false,
						Optional:            false,
						Computed:            true,
						Sensitive:           false,
						Description:         "Indicates if self is allowed as a WHO item",
						MarkdownDescription: "Indicates if self is allowed as a WHO item",
					},
					"categories": schema.SetAttribute{
						ElementType:         types.StringType,
						Required:            false,
						Optional:            false,
						Computed:            true,
						Sensitive:           false,
						Description:         "List of grant category IDs that are allowed as WHO items",
						MarkdownDescription: "List of grant category IDs that are allowed as WHO items",
					},
				},
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Allowed WHO items for the grant category",
				MarkdownDescription: "Allowed WHO items for the grant category",
			},
			"allowed_what_items": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"data_object": schema.BoolAttribute{
						Required:            false,
						Optional:            false,
						Computed:            true,
						Sensitive:           false,
						Description:         "Indicates if a data object is allowed as a WHAT item",
						MarkdownDescription: "Indicates if a data object is allowed as a WHAT item",
					},
				},
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Allowed WHAT items for the grant category",
				MarkdownDescription: "Allowed WHAT items for the grant category",
			},
		},
		Description:         "Find a grant category by name",
		MarkdownDescription: "Find a grant category by name",
	}
}

func (g *GrantCategoryDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data GrantCategoryDataSourceModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	grantCategories, err := g.client.GrantCategory().ListGrantCategories(ctx)
	if err != nil {
		response.Diagnostics.AddError("failed to list grant categories", err.Error())

		return
	}

	for i := range grantCategories {
		if grantCategories[i].Name == name {
			setGrantCategoryData(&grantCategories[i], &data, response.Diagnostics)

			if response.Diagnostics.HasError() {
				return
			}

			response.Diagnostics.Append(response.State.Set(ctx, data)...)

			return
		}
	}

	response.Diagnostics.AddError("grant category not found", "grant category not found")
}

func (g *GrantCategoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func setGrantCategoryData(data *types2.GrantCategoryDetails, resp *GrantCategoryDataSourceModel, diagnostic diag.Diagnostics) {
	resp.Id = types.StringValue(data.Id)
	resp.Description = types.StringValue(data.Description)
	resp.IsSystem = types.BoolValue(data.IsSystem)
	resp.IsDefault = types.BoolValue(data.IsDefault)
	resp.CanCreate = types.BoolValue(data.CanCreate)
	resp.AllowDuplicateNames = types.BoolValue(data.AllowDuplicateNames)
	resp.MultiDataSource = types.BoolValue(data.MultiDataSource)

	// Default types per DS
	defaultTypesPerDs := make([]attr.Value, 0, len(data.DefaultTypePerDataSource))

	for _, v := range data.DefaultTypePerDataSource {
		objectValue, diags := types.ObjectValue(map[string]attr.Type{
			"data_source": types.StringType,
			"type":        types.StringType,
		},
			map[string]attr.Value{
				"data_source": types.StringValue(v.GetDataSource()),
				"type":        types.StringValue(v.GetType()),
			})

		diagnostic.Append(diags...)

		if diagnostic.HasError() {
			return
		}

		defaultTypesPerDs = append(defaultTypesPerDs, objectValue)
	}

	defaultTypesPerDsAttr, diags := types.SetValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"data_source": types.StringType,
			"type":        types.StringType,
		},
	}, defaultTypesPerDs)

	diagnostic.Append(diags...)

	if diagnostic.HasError() {
		return
	}

	resp.DefaultTypePerDataSource = defaultTypesPerDsAttr

	// Allowed WHO items
	whoCategoryValues := make([]attr.Value, 0, len(data.AllowedWhoItems.Categories))
	for _, v := range data.AllowedWhoItems.Categories {
		whoCategoryValues = append(whoCategoryValues, types.StringValue(v))
	}

	whoCategories, diags := types.SetValue(types.StringType, whoCategoryValues)

	diagnostic.Append(diags...)

	if diagnostic.HasError() {
		return
	}

	allowedWhoItems, diags := types.ObjectValue(
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

	diagnostic.Append(diags...)

	if diagnostic.HasError() {
		return
	}

	resp.AllowedWhoItems = allowedWhoItems

	// Allowed WHAT items
	allowedWhatItems, diags := types.ObjectValue(
		map[string]attr.Type{
			"data_object": types.BoolType,
		},
		map[string]attr.Value{
			"data_object": types.BoolValue(data.AllowedWhatItems.DataObject),
		},
	)

	diagnostic.Append(diags...)

	if diagnostic.HasError() {
		return
	}

	resp.AllowedWhatItems = allowedWhatItems
}
