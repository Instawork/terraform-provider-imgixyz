package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// With the datasource.DataSource implementation
func NewSourceDataSource() datasource.DataSource {
	return &SourceDataSource{}
}

// Ensure the implementation satisfies the datasource.DataSourceWithConfigure interface.
var _ datasource.DataSourceWithConfigure = &SourceDataSource{}

type SourceDataSource struct {
	client *ImgixClient
}

type SourceDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Deployment       types.Object `tfsdk:"deployment"`
	DeploymentStatus types.String `tfsdk:"deployment_status"`
	Enabled          types.Bool   `tfsdk:"enabled"`
}

var deployObjectType = schema.ObjectAttribute{
	Computed: true,
	AttributeTypes: map[string]attr.Type{
		"annotation":       types.StringType,
		"type":             types.StringType,
		"s3_bucket":        types.StringType,
		"s3_prefix":        types.StringType,
		"s3_access_key":    types.StringType,
		"s3_secret_key":    types.StringType,
		"imgix_subdomains": types.ListType{ElemType: types.StringType},
	},
}

func (d *SourceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source"
}

func (d *SourceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// This isn't always called so don't panic yet
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ImgixClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ImgixClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *SourceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Required: true},
			"name":              schema.StringAttribute{Computed: true},
			"deployment":        deployObjectType,
			"deployment_status": schema.StringAttribute{Computed: true},
			"enabled":           schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *SourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Prevent panic if the provider has not been configured.
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured HTTP Client",
			"Expected configured HTTP client. Please report this issue to the provider developers.",
		)
		return
	}

	// Read Terraform configuration data into the model
	data := new(SourceDataSourceModel)
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch our remote data
	source, err := d.client.GetSourceByID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch source by ID", err.Error())
		return
	}

	// Set domains
	var imgixSubdomains []attr.Value
	for _, a := range source.Deployment.ImgixSubdomains {
		imgixSubdomains = append(imgixSubdomains, types.StringValue(a))
	}

	// Typically data sources will make external calls, however this example
	// hardcodes setting the id attribute to a specific value for brevity.
	data.ID = types.StringValue(source.ID)
	data.Name = types.StringValue(source.Name)
	data.Enabled = types.BoolValue(source.Enabled)
	data.DeploymentStatus = types.StringValue(source.DeploymentStatus)
	deployment, diag := types.ObjectValue(deployObjectType.AttributeTypes, map[string]attr.Value{
		"type":             types.StringValue(source.Deployment.Type),
		"annotation":       types.StringValue(source.Deployment.Annotation),
		"s3_bucket":        types.StringValue(source.Deployment.S3Bucket),
		"s3_prefix":        types.StringValue(source.Deployment.S3Prefix),
		"s3_access_key":    types.StringValue(source.Deployment.S3AccessKey),
		"s3_secret_key":    types.StringValue(source.Deployment.S3SecretKey),
		"imgix_subdomains": types.ListValueMust(types.StringType, imgixSubdomains),
	})
	data.Deployment = deployment
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
