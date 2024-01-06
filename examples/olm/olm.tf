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

resource "olm_v0_instance" "test" {
  version = "v0.26.0"
}