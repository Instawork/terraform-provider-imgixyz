# terraform-provider-imgixyz

Terraform provider for imgix with an open license

![GitHub release (latest by date)](https://img.shields.io/github/v/release/Instawork/terraform-provider-imgixyz)
![GitHub license](https://img.shields.io/github/license/Instawork/terraform-provider-imgixyz)

## Motivation

You can read our blog post on the subject [here](https://engineering.instawork.com/license-to-code-why-one-text-file-can-ruin-your-week-133745e487d2) but the TL;DR is make sure you add a `LICENSE` to your projects!

## Usage

You can download and look at the docs at our [Terraform Registry Page](https://registry.terraform.io/providers/Instawork/imgixyz)

A very simple example:

```terraform
terraform {
  required_version = "~> 1.3.0"
  required_providers {
    imgixyz = {
      source  = "Instawork/imgixyz"
      version = "0.0.11"
    }
  }
}

provider "imgixyz" {
  token          = "IMGIX_TOKEN_GOES_HERE"
  upsert_by_name = true
}

resource "imgixyz_source" "profile" {
  name    = "imgix-dev-profile"
  enabled = true
  deployment {
    type             = "s3"
    annotation       = "Deployment created by Terraform for dev environment."
    imgix_subdomains = ["imgix-dev-profile"]
    s3_access_key = "AWS_ACCESS_KEY_HERE"
    s3_secret_key = "AWS_SECRET_KEY_HERE"
    s3_bucket     = "imgix-dev-profile-images"
  }
}
```

## What is supported?

We currently only support a very small subset of the `/sources` API.

**Fields that are supported:**

- name
- enabled
- deployment
  - type
  - annotation
  - imgix_subdomains
  - s3_access_key
  - s3_secret_key
  - s3_bucket
  - s3_prefix

These fields are the only ones required to get up and running with Imgix + AWS.

## Contribution

You can read the API docs for Imgix Management [here](https://docs.imgix.com/apis/management)

Feel free to open a PR [here](https://github.com/Instawork/terraform-provider-imgixyz/pulls) and add support for more types of resources!
