package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk-go"
	"github.com/raito-io/sdk-go/services"
	types2 "github.com/raito-io/sdk-go/types"
)

var _ datasource.DataSource = (*IdentityStoreDataSource)(nil)

type IdentityStoreDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Master      types.Bool   `tfsdk:"master"`
	IsNative    types.Bool   `tfsdk:"is_native"`
	Owners      types.Set    `tfsdk:"owners"`
}

type IdentityStoreDataSource struct {
	client *sdk.RaitoClient
}

func NewIdentityStoreDataSource() datasource.DataSource {
	return &IdentityStoreDataSource{}
}

func (i *IdentityStoreDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_identitystore"
}

func (i *IdentityStoreDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the identity store",
				MarkdownDescription: "The ID of the identity store",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "The name of the identity store",
				MarkdownDescription: "The name of the identity store",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(3)},
			},
			"description": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The description of the identity store",
				MarkdownDescription: "The description of the identity store",
			},
			"master": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "True, if this is a master identity store",
				MarkdownDescription: "`True`, if this is a master identity store. Default: `false`",
			},
			"is_native": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "True, if this is a native identity store",
				MarkdownDescription: "True, if this is a native identity store",
			},
			"owners": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The IDs of the owners of the identity store",
				MarkdownDescription: "The IDs of the owners of the identity store",
			},
		},
		Description:         "Find a identity store by name",
		MarkdownDescription: "Find a Raito [Identity Store](https://docs.raito.io/docs/cloud/identity_stores) by name.",
	}
}

func (i *IdentityStoreDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data IdentityStoreDataSourceModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	isChan := i.client.IdentityStore().ListIdentityStores(cancelCtx, services.WithListIdentityStoresFilter(&types2.IdentityStoreFilterInput{
		Search: &name,
	}))

	for is := range isChan {
		if is.HasError() {
			response.Diagnostics.AddError("Failed to list identity stores", is.GetError().Error())

			return
		}

		isItem := is.GetItem()

		if isItem.Name != name {
			continue
		}

		data.Id = types.StringValue(isItem.Id)
		data.Name = types.StringValue(isItem.Name)
		data.Description = types.StringValue(isItem.Description)
		data.Master = types.BoolValue(isItem.Master)
		data.IsNative = types.BoolValue(isItem.Native)

		owners, diagn := getOwners(ctx, isItem.Id, i.client)
		response.Diagnostics.Append(diagn...)

		if response.Diagnostics.HasError() {
			return
		}

		data.Owners = owners

		response.Diagnostics.Append(response.State.Set(ctx, data)...)

		return
	}
}

func (i *IdentityStoreDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	i.client = client
}
