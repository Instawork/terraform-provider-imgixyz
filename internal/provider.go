package internal

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ImgixyzProvider{
			Version: version,
		}
	}
}

// Ensure the implementation satisfies the provider.Provider interface.
var _ provider.Provider = &ImgixyzProvider{}

type ImgixyzProvider struct {
	// Version is an example field that can be set with an actual provider
	// version on release, "dev" when the provider is built and ran locally,
	// and "test" when running acceptance testing.
	Version string
}

type ImgixyzProviderModel struct {
	Token        types.String `tfsdk:"token"`
	UpsertByName types.Bool   `tfsdk:"upsert_by_name"`
}

// Metadata satisfies the provider.Provider interface for ImgixyzProvider
func (p *ImgixyzProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "imgixyz"
}

// Schema satisfies the provider.Provider interface for ImgixyzProvider.
func (p *ImgixyzProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Imgix API Token which can be created on <https://dashboard.imgix.com/api-keys>",
			},
			"upsert_by_name": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Imgix does not support deleting a source. Therefore, enabling this will import existing source(s) by the name attribute during create.",
			},
		},
	}
}

// Configure satisfies the provider.Provider interface for ImgixyzProvider.
func (p *ImgixyzProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	token := os.Getenv("IMGIXYZ_TOKEN")
	var data ImgixyzProviderModel

	// Read configuration data into model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Check configuration data, which should take precedence over
	// environment variable data, if found.
	if data.Token.ValueString() != "" {
		token = data.Token.ValueString()
	}

	// Validate our token
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Token Configuration",
			"While configuring the provider, the token was not found in "+
				"the IMGIXYZ_TOKEN environment variable or provider "+
				"configuration block token attribute.",
		)
		// Not returning early allows the logic to collect all errors.
	}

	// Set our client on ResourceData to be accessed later
	client := NewImgixClient(token, data.UpsertByName.ValueBool())
	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources satisfies the provider.Provider interface for ImgixyzProvider.
func (p *ImgixyzProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSourceDataSource,
	}
}

// Resources satisfies the provider.Provider interface for ImgixyzProvider.
func (p *ImgixyzProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSourceResource,
	}
}
