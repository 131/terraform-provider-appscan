Terraform Appscan Provider
==================

The Appscan Provider allows Terraform to manage [HCL's appscan](https://www.hcl-software.com/appscan) resources.

- Website: [registry.terraform.io](https://registry.terraform.io/providers/131/appscan/latest/docs)

Requirements
------------

- [Terraform](https://www.terraform.io/downloads.html) 0.12.x
- [Go](https://golang.org/doc/install) 1.20 (to build the provider plugin)


Using the provider
----------------------

Please see the documentation in the [Terraform registry](https://registry.terraform.io/providers/131/appscan/latest/docs).

Then create a Terraform configuration using this exact provider:

```sh
# Configure the AppScan Provider
terraform {
  required_providers {
    appscan = {
      source = "131/appscan"
    }
  }
}

provider "appscan" {
  key_id     = [appscan key_id]   # or via env APPSCAN_KEY_ID
  key_secret = [appscan key_secret] # or via env APPSCAN_KEY_ID
}

EOF

# Initialize your project
$ terraform init
...

# Apply your resources & datasources
$ terraform apply
...
```



# Credits
* [Francois Leurent](https://github.com/131)