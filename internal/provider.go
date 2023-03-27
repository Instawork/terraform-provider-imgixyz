package internal

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("IMGIX_TOKEN", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{},
		DataSourcesMap: map[string]*schema.Resource{
			"imgixyz_source": dataSourceSource(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	token := d.Get("token").(string)
	var diags diag.Diagnostics
	if token != "" {
		c := &ImgixClient{}
		c.SetAuthToken(token)
		return c, diags
	}
	diags = append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  "Unable to create imgix client",
		Detail:   "Unable to create an imgix client without a token provided",
	})
	return nil, diags
}
