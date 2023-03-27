package internal

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSourceRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: false,
			},
		},
	}
}

func dataSourceSourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ImgixClient)
	var diags diag.Diagnostics
	sourceID := strconv.Itoa(d.Get("id").(int))
	source, err := c.GetSourceByID(sourceID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(sourceID)
	d.Set("name", source.Name)
	return diags
}
