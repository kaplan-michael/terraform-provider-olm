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

resource "olm_v0_operator" "test" {
  name             = "cert-manager"
  channel          = "stable"
  namespace        = "operators"
  source           = "operatorhubio-catalog"
  source_namespace = "olm"
  depends_on       = [olm_v0_instance.test]
}