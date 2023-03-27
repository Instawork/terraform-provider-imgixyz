package internal

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceSource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSourceCreate,
		ReadContext:   resourceSourceRead,
		UpdateContext: resourceSourceUpdate,
		DeleteContext: resourceSourceDelete,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"deployment": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"deployment_status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceSourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ImgixClient)
	var diags diag.Diagnostics

	// create source
	deployment := d.Get("deployment").(map[string]interface{})
	source := new(ImgixSource)
	source.Name = d.Get("name").(string)
	source.Deployment.Type = deployment["type"].(string)
	source.Deployment.S3Bucket = deployment["s3_bucket"].(string)
	source.Deployment.S3AccessKey = deployment["s3_access_key"].(string)
	source.Deployment.S3SecretKey = deployment["s3_secret_key"].(string)
	source.Enabled = d.Get("enabled").(bool)
	source, err := c.CreateSource(source)
	if err != nil {
		return diag.FromErr(err)
	}

	// set the source and refresh via id
	d.SetId(source.ID)
	resourceSourceRead(ctx, d, m)
	return diags
}

func resourceSourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return DataSourceSourceRead(ctx, d, m)
}

func resourceSourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// TODO: Implement
	// c := m.(*ImgixClient)
	// if d.HasChange("items") {
	// 	d.Set("last_updated", time.Now().Format(time.RFC850))
	// }
	return resourceSourceRead(ctx, d, m)
}

func resourceSourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// TODO: Implement
	// c := m.(*ImgixClient)
	var diags diag.Diagnostics
	// sourceID := d.Id()
	// TODO: delete via api
	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	// d.SetId("")
	return diags
}
