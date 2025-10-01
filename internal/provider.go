package internal

import (
	"context"

	"github.com/collibra/access-governance-go-sdk"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure RaitoCloudProvider satisfies various provider interfaces.
var _ provider.Provider = &RaitoCloudProvider{}

// RaitoCloudProvider defines the provider implementation.
type RaitoCloudProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// RaitoCloudProviderModel describes the provider data model.
type RaitoCloudProviderModel struct {
	Url    types.String `tfsdk:"url"`
	User   types.String `tfsdk:"user"`
	Secret types.String `tfsdk:"secret"`
}

func (p *RaitoCloudProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "raito"
	resp.Version = p.version
}

func (p *RaitoCloudProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Sensitive:           false,
				Description:         "The base url of your Collibra instance (i.e. https://<your>.collibra.com)",
				MarkdownDescription: "The base url of your Collibra instance (i.e. https://<your>.collibra.com)",
			},
			"user": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Sensitive:           false,
				Description:         "The username to use to sign in to your Raito Cloud instance",
				MarkdownDescription: "The username to use to sign in to your Raito Cloud instance",
			},
			"secret": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Sensitive:           true,
				Description:         "The password to use to sign in to your Raito Cloud instance",
				MarkdownDescription: "The password to use to sign in to your Raito Cloud instance",
			},
		},
	}
}

func (p *RaitoCloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data RaitoCloudProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := sdk.NewClient(data.User.ValueString(), data.Secret.ValueString(), data.Url.ValueString())

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *RaitoCloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDataSourceResource,
		NewIdentityStoreResource,
		NewGlobalRoleAssignmentResource,
		NewGrantCategoryResource,
		NewGrantResource,
		NewFilterResource,
		NewMaskResource,
	}
}

func (p *RaitoCloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataSourceDataSource,
		NewGrantCategoryDataSource,
		NewIdentityStoreDataSource,
		NewUserDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RaitoCloudProvider{
			version: version,
		}
	}
}
