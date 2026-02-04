package internal

import (
	"context"
	"time"

	"github.com/collibra/data-access-go-sdk"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure CollibraDataAccessProvider satisfies various provider interfaces.
var _ provider.Provider = &CollibraDataAccessProvider{}

// CollibraDataAccessProvider defines the provider implementation.
type CollibraDataAccessProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// CollibraDataAccessProviderModel describes the provider data model.
type CollibraDataAccessProviderModel struct {
	Url    types.String `tfsdk:"url"`
	User   types.String `tfsdk:"user"`
	Secret types.String `tfsdk:"secret"`
}

func (p *CollibraDataAccessProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "collibra-data-access"
	resp.Version = p.version
}

func (p *CollibraDataAccessProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
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
				Description:         "The username to use to sign in to your Collibra instance",
				MarkdownDescription: "The username to use to sign in to your Collibra instance",
			},
			"secret": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Sensitive:           true,
				Description:         "The password to use to sign in to your Collibra instance",
				MarkdownDescription: "The password to use to sign in to your Collibra instance",
			},
		},
	}
}

func (p *CollibraDataAccessProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data CollibraDataAccessProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	clientOptions := []sdk.ClientOptions{
		sdk.WithRetryWaitMin(200 * time.Microsecond),
		sdk.WithRetryWaitMax(15 * time.Second),
		sdk.WithRetryMax(5),
		sdk.WithUsername(data.User.ValueString()),
		sdk.WithPassword(data.Secret.ValueString()),
	}

	client := sdk.NewClient(data.User.ValueString(), clientOptions...)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *CollibraDataAccessProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDataSourceResource,
		NewGrantCategoryResource,
		NewGrantResource,
		NewFilterResource,
		NewMaskResource,
	}
}

func (p *CollibraDataAccessProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataSourceDataSource,
		NewGrantCategoryDataSource,
		NewUserDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CollibraDataAccessProvider{
			version: version,
		}
	}
}
