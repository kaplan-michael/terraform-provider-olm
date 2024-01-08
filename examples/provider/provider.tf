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