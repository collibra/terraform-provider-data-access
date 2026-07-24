package internal

import (
	"context"
	"fmt"

	"github.com/collibra/data-access-go-sdk"
	types2 "github.com/collibra/data-access-go-sdk/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*GrantCategoryDataSource)(nil)

type GrantCategoryDataSourceModel struct {
	Id                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	NamePlural          types.String `tfsdk:"name_plural"`
	Description         types.String `tfsdk:"description"`
	IsSystem            types.Bool   `tfsdk:"is_system"`
	IsDefault           types.Bool   `tfsdk:"is_default"`
	CanCreate           types.Bool   `tfsdk:"can_create"`
	AllowDuplicateNames types.Bool   `tfsdk:"allow_duplicate_names"`
	MultiDataSource     types.Bool   `tfsdk:"multi_data_source"`
	AllowedWhoItems     types.Object `tfsdk:"allowed_who_items"`
	AllowedWhatItems    types.Object `tfsdk:"allowed_what_items"`
}

type GrantCategoryDataSource struct {
	client *sdk.CollibraClient
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
				Description:         "The ID of the role category.",
				MarkdownDescription: "The ID of the role category.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "The name of the requested role category.",
				MarkdownDescription: "The name of the requested role category.",
			},
			"name_plural": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The plural form of the display name of the role category.",
				MarkdownDescription: "The plural form of the display name of the role category.",
			},
			"description": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The description of the role category.",
				MarkdownDescription: "The description of the role category.",
			},
			"is_system": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Indicates whether the role category is a system category.",
				MarkdownDescription: "Indicates whether the role category is a system category.",
			},
			"is_default": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Indicates whether the role category is the default category.",
				MarkdownDescription: "Indicates whether the role category is the default category.",
			},
			"can_create": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Indicates whether a role of this category can be created.",
				MarkdownDescription: "Indicates whether a role of this category can be created.",
			},
			"allow_duplicate_names": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Indicates whether duplicate names are allowed for the roles in this category.",
				MarkdownDescription: "Indicates whether duplicate names are allowed for the roles in this category.",
			},
			"multi_data_source": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Indicates whether roles of this category can have multiple data sources.",
				MarkdownDescription: "Indicates whether roles of this category can have multiple data sources.",
			},
			"allowed_who_items": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"user": schema.BoolAttribute{
						Required:            false,
						Optional:            false,
						Computed:            true,
						Sensitive:           false,
						Description:         "Indicates whether a user is allowed in the Who components for the roles in this category.",
						MarkdownDescription: "Indicates whether a user is allowed in the Who components for the roles in this category.",
					},
					"inheritance": schema.BoolAttribute{
						Required:            false,
						Optional:            false,
						Computed:            true,
						Sensitive:           false,
						Description:         "Indicates whether inheritance is allowed in the Who components for the roles in this category.",
						MarkdownDescription: "Indicates whether inheritance is allowed in the Who components for the roles in this category.",
					},
					"self": schema.BoolAttribute{
						Required:            false,
						Optional:            false,
						Computed:            true,
						Sensitive:           false,
						Description:         "Indicates whether self is allowed in the Who components for the roles in this category.",
						MarkdownDescription: "Indicates whether self is allowed in the Who components for the roles in this category.",
					},
					"categories": schema.SetAttribute{
						ElementType:         types.StringType,
						Required:            false,
						Optional:            false,
						Computed:            true,
						Sensitive:           false,
						Description:         "The list of role category IDs that are allowed in the Who components for the roles in this category.",
						MarkdownDescription: "The list of role category IDs that are allowed in the Who components for the roles in this category.",
					},
				},
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The allowed items in the Who components for the roles in this category.",
				MarkdownDescription: "The allowed items in the Who components for the roles in this category. See the nested schema below.",
			},
			"allowed_what_items": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"data_object": schema.BoolAttribute{
						Required:            false,
						Optional:            false,
						Computed:            true,
						Sensitive:           false,
						Description:         "Indicates whether a data object is allowed in the What components for the roles in this category.",
						MarkdownDescription: "Indicates whether a data object is allowed in the What components for the roles in this category.",
					},
				},
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The allowed items in the What components for the roles in this category.",
				MarkdownDescription: "The allowed items in the What components for the roles in this category. See the nested schema below.",
			},
		},
		Description:         "The data source to get a role category in Collibra Data Access by name.",
		MarkdownDescription: "The data source to get a role category in Collibra Data Access by name.\n-> **Note:** In Collibra Data Access, grants are called roles, and grant categories are called role categories.",
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

	g.client = client
}

func setGrantCategoryData(data *types2.GrantCategoryDetails, resp *GrantCategoryDataSourceModel, diagnostic diag.Diagnostics) {
	resp.Id = types.StringValue(data.Id)
	resp.NamePlural = types.StringValue(data.NamePlural)
	resp.Description = types.StringValue(data.Description)
	resp.IsSystem = types.BoolValue(data.IsSystem)
	resp.IsDefault = types.BoolValue(data.IsDefault)
	resp.CanCreate = types.BoolValue(data.CanCreate)
	resp.AllowDuplicateNames = types.BoolValue(data.AllowDuplicateNames)
	resp.MultiDataSource = types.BoolValue(data.MultiDataSource)

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
			"inheritance": types.BoolType,
			"self":        types.BoolType,
			"categories":  types.SetType{ElemType: types.StringType},
		},
		map[string]attr.Value{
			"user":        types.BoolValue(data.AllowedWhoItems.User),
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
