---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "imgixyz_source Data Source - terraform-provider-imgixyz"
subcategory: ""
description: |-
  
---

# imgixyz_source (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `deployment` (Block, Read-only) (see [below for nested schema](#nestedblock--deployment))
- `enabled` (Boolean)
- `id` (String) The ID of this resource.
- `name` (String)

<a id="nestedblock--deployment"></a>
### Nested Schema for `deployment`

Read-Only:

- `annotation` (String)
- `imgix_subdomains` (List of String)
- `s3_access_key` (String, Sensitive)
- `s3_bucket` (String)
- `s3_prefix` (String)
- `s3_secret_key` (String, Sensitive)
- `type` (String)


