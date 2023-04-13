---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "imgixyz_source Resource - terraform-provider-imgixyz"
subcategory: ""
description: |-
  
---

# imgixyz_source (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `enabled` (Boolean)
- `name` (String)
- `deployment` (Block) (see [below for nested schema](#nestedblock--deployment))


### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--deployment"></a>
### Nested Schema for `deployment`

Required:

- `annotation` (String)
- `imgix_subdomains` (List of String)
- `type` (String)

Optional:

- `s3_access_key` (String, Sensitive)
- `s3_bucket` (String)
- `s3_prefix` (String)
- `s3_secret_key` (String, Sensitive)

