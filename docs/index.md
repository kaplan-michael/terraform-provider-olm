---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "olm Provider"
subcategory: ""
description: |-
  
---

# olm Provider



## Example Usage

```terraform
terraform {
  required_providers {
    olm = {
      source = "kaplan-michael/olm"
    }
  }
}

provider "olm" {
  kubeconfig = file("~/.kube/config")
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `ca_certificate` (String) Kubernetes API server CA certificate
- `client_certificate` (String) Kubernetes API server client certificate
- `client_key` (String) Kubernetes API server client key
- `host` (String) Kubernetes API server host
- `kubeconfig` (String, Sensitive) Kubeconfig raw file
