package internal

import (
	"context"
	"fmt"

	"github.com/collibra/data-access-go-sdk"
	"github.com/collibra/data-access-go-sdk/services"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*DataSourceDataSource)(nil)

type DataSourceDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Parent      types.String `tfsdk:"parent"`
	Owners      types.Set    `tfsdk:"owners"`
}

type DataSourceDataSource struct {
	client *sdk.CollibraClient
}

func NewDataSourceDataSource() datasource.DataSource {
	return &DataSourceDataSource{}
}

func (d *DataSourceDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_datasource"
}

func (d *DataSourceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the requested data source",
				MarkdownDescription: "The ID of the requested data source",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "The name of the requested data source",
				MarkdownDescription: "The name of the requested data source",
				Validators:          nil,
			},
			"description": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The description of the data source",
				MarkdownDescription: "The description of the data source",
			},
			"sync_method": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The sync method of the data source. Should be set to ON_PREM for now.",
				MarkdownDescription: "The sync method of the data source. Should be set to `ON_PREM` for now.",
			},
			"parent": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the parent data source, if applicable",
				MarkdownDescription: "The ID of the parent data source, if applicable",
			},
			"owners": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The IDs of the owners of the data source",
				MarkdownDescription: "The IDs of the owners of the data source",
			},
		},
		Description:         "Find a data source based on the name",
		MarkdownDescription: "Find a Data Source based on the name",
	}
}

func (d *DataSourceDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data DataSourceDataSourceModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	dsSeq := d.client.DataSource().ListDataSources(ctx, services.WithDataSourceListSearch(&name))

	for dsItem, err := range dsSeq {
		if err != nil {
			response.Diagnostics.AddError("Failed to list data sources", err.Error())

			return
		}

		if dsItem.Name == name {
			var parentId *string
			if dsItem.Parent != nil {
				parentId = &dsItem.Parent.Id
			}

			data.Id = types.StringValue(dsItem.Id)
			data.Description = types.StringValue(dsItem.Description)
			data.Parent = types.StringPointerValue(parentId)

			owners, diagn := getOwners(ctx, dsItem.Id, d.client)
			response.Diagnostics.Append(diagn...)

			if response.Diagnostics.HasError() {
				return
			}

			data.Owners = owners

			response.Diagnostics.Append(response.State.Set(ctx, data)...)

			return
		}
	}
}

func (d *DataSourceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}
