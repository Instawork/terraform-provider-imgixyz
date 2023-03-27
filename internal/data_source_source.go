package internal

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: DataSourceSourceRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"deployment": {
				Computed: true,
				Type:     schema.TypeMap,
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
				Computed: true,
			},
		},
	}
}

func DataSourceSourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ImgixClient)
	var diags diag.Diagnostics
	sourceID := d.Get("id").(string)
	source, err := c.GetSourceByID(sourceID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(sourceID)
	d.Set("name", source.Name)
	d.Set("deployment_status", source.DeploymentStatus)
	d.Set("deployment", source.Deployment)
	d.Set("enabled", source.Enabled)
	return diags
}
