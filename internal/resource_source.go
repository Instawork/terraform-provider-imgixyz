package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// With the resource.Resource implementation
func NewSourceResource() resource.Resource {
	return &SourceResource{}
}

// Ensure the implementation satisfies the resource.ResourceWithConfigure interface.
var _ resource.ResourceWithConfigure = &SourceResource{}

type SourceResource struct {
	client *ImgixClient
}

func resourceDeployObjectType(computed, required bool) schema.Block {
	return schema.SingleNestedBlock{
		Attributes: map[string]schema.Attribute{
			"annotation":    schema.StringAttribute{Required: required, Computed: computed},
			"type":          schema.StringAttribute{Required: required, Computed: computed},
			"s3_bucket":     schema.StringAttribute{Optional: required, Computed: computed},
			"s3_prefix":     schema.StringAttribute{Optional: required, Computed: computed},
			"s3_access_key": schema.StringAttribute{Optional: required, Computed: computed},
			"s3_secret_key": schema.StringAttribute{
				Optional:      required,
				Computed:      computed,
				PlanModifiers: []planmodifier.String{UseStateAfterSetModifier()},
			},
			"imgix_subdomains": schema.ListAttribute{ElementType: types.StringType, Required: required, Computed: computed},
		},
	}
}

func (d *SourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source"
}

func (r SourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name":    schema.StringAttribute{Required: true},
			"enabled": schema.BoolAttribute{Required: true},
		},
		Blocks: map[string]schema.Block{
			"deployment": resourceDeployObjectType(false, true),
		},
	}
}

func (d *SourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// This isn't always called so don't panic yet
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ImgixClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ImgixClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (r SourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Prevent panic if the provider has not been configured.
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured HTTP Client",
			"Expected configured HTTP client. Please report this issue to the provider developers.",
		)
		return
	}

	// Read Terraform plan data into the model
	data := new(SourceModel)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert from Terraform data model into API data model
	localSource := new(ImgixSource)
	diags := convertSourceModelToSource(ctx, data, localSource)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If upsert_by_name = true, try to "create" an existing resource by syncing state
	// and updating enabled = true
	var source *ImgixSource
	if r.client.upsertByName {
		tflog.Debug(ctx, "Using the source name to try to find an existing resource")
		s, err := r.client.GetSourceByName(data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Upsert Resource",
				"An unexpected error occurred while creating the resource. "+
					"Please report this issue to the provider developers.\n\n"+
					"Client Error: "+err.Error(),
			)
			return
		}
		source = s
	}

	// Call out to our api and create the resource
	if source == nil {
		s, err := r.client.CreateSource(localSource)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create Resource",
				"An unexpected error occurred while creating the resource. "+
					"Please report this issue to the provider developers.\n\n"+
					"Client Error: "+err.Error(),
			)
			return
		}
		source = s
	} else {
		resp.Diagnostics.AddWarning(
			"Found Existing Resource",
			"Imgix doesn't allow deletion of sources but we found an existing source by name and `upsert_by_name = true`.\n"+
				"We've imported the existing resource and have updated the source with any changed attributes.",
		)
		// Update our data in remote to sync with what we imported
		localSource.ID = source.ID
		s, err := r.client.UpdateSource(localSource)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Sync Existing Resource",
				"An unexpected error occurred while creating the resource create request. "+
					"Please report this issue to the provider developers.\n\n"+
					"Client Error: "+err.Error(),
			)
			return
		}
		source = s
	}

	// Convert our remote struct into our terraform model
	state := new(SourceModel)
	diags = convertSourceToSourceModel(ctx, source, data, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *SourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Prevent panic if the provider has not been configured.
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured HTTP Client",
			"Expected configured HTTP client. Please report this issue to the provider developers.",
		)
		return
	}

	// Read Terraform state data into the model
	data := new(SourceModel)
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch our remote data
	source, err := r.client.GetSourceByID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch source by ID", err.Error())
		return
	}

	// Convert our remote data to local
	state := new(SourceModel)
	diag := convertSourceToSourceModel(ctx, source, data, state)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set our state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func updateSourceEnabledAttribute(client *ImgixClient, sourceID string, enabled bool) error {
	_, err := client.UpdateSource(&ImgixSource{ID: sourceID, Enabled: &enabled})
	return err
}

func (r SourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Prevent panic if the provider has not been configured.
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured HTTP Client",
			"Expected configured HTTP client. Please report this issue to the provider developers.",
		)
		return
	}

	// Read Terraform plan into the model
	oldState := new(SourceModel)
	resp.Diagnostics.Append(req.State.Get(ctx, &oldState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read Terraform plan into the model
	plan := new(SourceModel)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = oldState.ID

	// Convert from Terraform data model into API data model
	source := new(ImgixSource)
	diags := convertSourceModelToSource(ctx, plan, source)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Imgix won't allow updates to a disabled source, gather the state and determine the actions
	enabledState := oldState.Enabled.ValueBoolPointer()
	enabledPlan := plan.Enabled.ValueBool()
	currentlyDisabled := enabledState != nil && !*enabledState
	isUpdatingDisabledSource := currentlyDisabled && !enabledPlan
	shouldUpdateThenEnable := currentlyDisabled && enabledPlan
	enabledPlanPtr := plan.Enabled.ValueBoolPointer()
	shouldUpdateThenDisable := oldState.Enabled.ValueBool() && enabledPlanPtr != nil && !*enabledPlanPtr

	// Don't allow updating fields unless we are already enabled or setting `enabled = true`
	if isUpdatingDisabledSource {
		resp.Diagnostics.AddError(
			"Unable to Update Resource When Disabled",
			"Imgix doesn't allow updates to attributes when `enabled = false`, this is a no-op.\n"+
				"Please set `enabled = true` to update any attributes.",
		)
		return
	}

	// We need to enable the source first, then update attributes because Imgix doesn't allow updates to a disabled source
	if shouldUpdateThenEnable {
		err := updateSourceEnabledAttribute(r.client, source.ID, true)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Enable Resource",
				"An unexpected error occurred while enabling the resource during update request. "+
					"Please report this issue to the provider developers.\n\n"+
					"Client Error: "+err.Error(),
			)
			return
		}
	}

	// Don't pass down our placeholder to update
	if source.Deployment.S3SecretKey == SECRET_KEY_PLACEHOLDER {
		source.Deployment.S3SecretKey = ""
	}

	// We have to update our data before we disable if we have other things planned, so set `enabled = true` for now
	if shouldUpdateThenDisable {
		b := true
		source.Enabled = &b
	}

	// Update our data in remote
	_, err := r.client.UpdateSource(source)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Resource",
			"An unexpected error occurred while creating the resource update request. "+
				"Please report this issue to the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	// Now that we've update the attributes, we can disable the source
	if shouldUpdateThenDisable {
		err := updateSourceEnabledAttribute(r.client, source.ID, false)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Enable Resource",
				"An unexpected error occurred while disabling the resource during update request. "+
					"Please report this issue to the provider developers.\n\n"+
					"Client Error: "+err.Error(),
			)
			return
		}
	}

	// Fetch our remote data again to be safe
	source, err = r.client.GetSourceByID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch source by ID", err.Error())
		return
	}

	// Convert our remote data to local
	state := new(SourceModel)
	diag := convertSourceToSourceModel(ctx, source, plan, state)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set our state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r SourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Prevent panic if the provider has not been configured.
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured HTTP Client",
			"Expected configured HTTP client. Please report this issue to the provider developers.",
		)
		return
	}

	// Read Terraform prior state data into the model
	data := new(SourceModel)
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the resource
	err := r.client.DeleteSourceByID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Resource",
			"An unexpected error occurred while deleting the source. "+
				"Please report this issue to the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}
}

func convertSourceModelToSource(ctx context.Context, sourceModel *SourceModel, targetSource *ImgixSource) diag.Diagnostics {
	if sourceModel.ID.ValueString() != "" {
		targetSource.ID = sourceModel.ID.ValueString()
	}
	targetSource.Enabled = sourceModel.Enabled.ValueBoolPointer()
	targetSource.Name = sourceModel.Name.ValueString()
	targetSource.Deployment.Type = sourceModel.Deployment.Type.ValueString()
	targetSource.Deployment.Annotation = sourceModel.Deployment.Annotation.ValueString()
	targetSource.Deployment.S3Bucket = sourceModel.Deployment.S3Bucket.ValueString()
	targetSource.Deployment.S3Prefix = sourceModel.Deployment.S3Prefix.ValueStringPointer()
	targetSource.Deployment.S3AccessKey = sourceModel.Deployment.S3AccessKey.ValueString()
	targetSource.Deployment.S3SecretKey = sourceModel.Deployment.S3SecretKey.ValueString()
	var domains []string
	diags := sourceModel.Deployment.ImgixSubdomains.ElementsAs(ctx, &domains, true)
	targetSource.Deployment.ImgixSubdomains = domains
	return diags
}
