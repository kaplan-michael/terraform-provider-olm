resource "olm_v0_operator" "test" {
  name             = "cert-manager"
  channel          = "stable"
  namespace        = "operators"
  source           = "operatorhubio-catalog"
  source_namespace = "olm"
}