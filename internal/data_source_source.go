package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const SECRET_KEY_PLACEHOLDER = "IMGIX_HIDES_KEYS"

// With the datasource.DataSource implementation
func NewSourceDataSource() datasource.DataSource {
	return &SourceDataSource{}
}

// Ensure the implementation satisfies the datasource.DataSourceWithConfigure interface.
var _ datasource.DataSourceWithConfigure = &SourceDataSource{}

type SourceDataSource struct {
	client *ImgixClient
}

type SourceModel struct {
	ID         types.String     `tfsdk:"id"`
	Name       types.String     `tfsdk:"name"`
	Deployment *DeploymentModel `tfsdk:"deployment"`
	Enabled    types.Bool       `tfsdk:"enabled"`
}

type DeploymentModel struct {
	Annotation      types.String `tfsdk:"annotation"`
	Type            types.String `tfsdk:"type"`
	S3Bucket        types.String `tfsdk:"s3_bucket"`
	S3Prefix        types.String `tfsdk:"s3_prefix"`
	S3AccessKey     types.String `tfsdk:"s3_access_key"`
	S3SecretKey     types.String `tfsdk:"s3_secret_key"`
	ImgixSubdomains types.List   `tfsdk:"imgix_subdomains"`
}

func dataDeployObjectType(computed, required bool) schema.Block {
	return schema.SingleNestedBlock{
		Attributes: map[string]schema.Attribute{
			"annotation":       schema.StringAttribute{Required: required, Computed: computed},
			"type":             schema.StringAttribute{Required: required, Computed: computed},
			"s3_bucket":        schema.StringAttribute{Optional: required, Computed: computed},
			"s3_prefix":        schema.StringAttribute{Optional: required, Computed: computed},
			"s3_access_key":    schema.StringAttribute{Optional: required, Computed: computed, Sensitive: true},
			"s3_secret_key":    schema.StringAttribute{Optional: required, Computed: computed, Sensitive: true},
			"imgix_subdomains": schema.ListAttribute{ElementType: types.StringType, Required: required, Computed: computed},
		},
	}
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
			"id":      schema.StringAttribute{Required: true},
			"name":    schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{Computed: true},
		},
		Blocks: map[string]schema.Block{
			"deployment": dataDeployObjectType(true, false),
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
	data := new(SourceModel)
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the deployment block
	deployment := new(DeploymentModel)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("deployment"), deployment)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Deployment = deployment

	// Fetch our remote data
	source, err := d.client.GetSourceByID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch source by ID", err.Error())
		return
	}

	// Convert our remote data into our local model
	state := new(SourceModel)
	diag := convertSourceToSourceModel(ctx, source, data, state)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func convertSourceToSourceModel(ctx context.Context, source *ImgixSource, readSourceModel, targetSourceModel *SourceModel) diag.Diagnostics {
	targetSourceModel.ID = types.StringValue(source.ID)
	targetSourceModel.Name = types.StringValue(source.Name)
	targetSourceModel.Enabled = types.BoolValue(*source.Enabled)

	// Set domains
	var imgixSubdomains []attr.Value
	for _, a := range source.Deployment.ImgixSubdomains {
		imgixSubdomains = append(imgixSubdomains, types.StringValue(a))
	}
	subdomains, diag := types.ListValue(types.StringType, imgixSubdomains)
	if diag.HasError() {
		return diag
	}

	// Set deployment
	targetSourceModel.Deployment = &DeploymentModel{
		Type:            types.StringValue(source.Deployment.Type),
		Annotation:      types.StringValue(source.Deployment.Annotation),
		S3Bucket:        types.StringValue(source.Deployment.S3Bucket),
		S3AccessKey:     types.StringValue(source.Deployment.S3AccessKey),
		ImgixSubdomains: subdomains,
	}

	// Set optional fields
	if source.Deployment.S3Prefix != nil && *source.Deployment.S3Prefix != "" {
		targetSourceModel.Deployment.S3Prefix = types.StringValue(*source.Deployment.S3Prefix)
	}

	// Imgix won't return the s3_secret_key after creation so we need to stick a fake value in there
	if source.Deployment.S3SecretKey != "" {
		tflog.Debug(ctx, "setting s3_secret_key from remote source")
		targetSourceModel.Deployment.S3SecretKey = types.StringValue(source.Deployment.S3SecretKey)
	} else if readSourceModel.Deployment != nil && readSourceModel.Deployment.S3SecretKey.ValueString() != "" {
		tflog.Debug(ctx, "s3_secret_key is set during create, setting value to local one")
		// NOTE: There is a bug in UnmarshalManyPayload that quotes our string
		value := strings.ReplaceAll(readSourceModel.Deployment.S3SecretKey.String(), "\"", "")
		targetSourceModel.Deployment.S3SecretKey = types.StringValue(value)
	} else {
		tflog.Debug(ctx, "s3_secret_key isn't returned, setting value to unknown")
		targetSourceModel.Deployment.S3SecretKey = types.StringValue(SECRET_KEY_PLACEHOLDER)
	}

	return nil
}
