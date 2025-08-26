package internal

import (
	"context"

	"github.com/raito-io/sdk-go"

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
	Domain      types.String `tfsdk:"domain"`
	User        types.String `tfsdk:"user"`
	Secret      types.String `tfsdk:"secret"`
	UrlOverride types.String `tfsdk:"url_override"`
}

func (p *RaitoCloudProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "raito"
	resp.Version = p.version
}

func (p *RaitoCloudProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Sensitive:           false,
				Description:         "The subdomain of your Raito Cloud instance (i.e. https://<this_part>.raito.cloud)",
				MarkdownDescription: "The subdomain of your Raito Cloud instance (i.e. https://<this_part>.raito.cloud)",
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
			"url_override": schema.StringAttribute{
				Required:    false,
				Optional:    true,
				Sensitive:   false,
				Description: "If set, this URL is used as address for the Raito Cloud API. Only used for testing purposes.",
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

	var options []func(options *sdk.ClientOptions)

	if !data.UrlOverride.IsNull() {
		options = append(options, sdk.WithUrlOverride(data.UrlOverride.ValueString()))
	}

	client := sdk.NewClient(ctx, data.Domain.ValueString(), data.User.ValueString(), data.Secret.ValueString(), options...)

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
		NewUserResource,
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
