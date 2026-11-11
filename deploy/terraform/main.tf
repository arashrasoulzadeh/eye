terraform {
  required_version = ">= 1.5.0"
  required_providers { kubernetes = { source = "hashicorp/kubernetes", version = "~> 2.0" } helm = { source = "hashicorp/helm", version = "~> 2.0" } }
}

variable "namespace" { type = string default = "eye" }
variable "admin_key" { type = string sensitive = true }
variable "api_key_salt" { type = string sensitive = true }
variable "database_url" { type = string sensitive = true default = "" }
variable "chart_path" { type = string default = "../helm/eye" }
variable "admin_host" { type = string default = "eye.example.com" }
variable "api_host" { type = string default = "eye-api.example.com" }

resource "kubernetes_namespace" "eye" { metadata { name = var.namespace } }

resource "helm_release" "eye" {
  name = "eye"
  namespace = kubernetes_namespace.eye.metadata[0].name
  chart = var.chart_path
  values = [yamlencode({ server = { adminKey = var.admin_key, apiKeySalt = var.api_key_salt, databaseURL = var.database_url }, ingress = { enabled = true, adminHost = var.admin_host, apiHost = var.api_host } })]
}
