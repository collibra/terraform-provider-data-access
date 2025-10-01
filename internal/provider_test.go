package internal

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"raito": providerserver.NewProtocol6WithError(New("test")()),
}

var providerConfig = `
variable "raito_user" {
	type = string
}

variable "raito_secret" {
    type = string
}

variable "raito_url" {
    type = string
}

provider "raito" {
  user         = var.raito_user
  secret       = var.raito_secret
  url          = var.raito_url
}
`

func AccProviderPreCheck(t *testing.T) {
	if v := os.Getenv("TF_VAR_raito_user"); v == "" {
		t.Fatal("TF_VAR_raito_user must be set for acceptance testing")
	}

	if v := os.Getenv("TF_VAR_raito_secret"); v == "" {
		t.Fatal("TF_VAR_raito_secret must be set for acceptance testing")
	}

	if v := os.Getenv("TF_VAR_raito_url"); v == "" {
		t.Fatal("TF_VAR_raito_url must be set for acceptance testing")
	}
}
