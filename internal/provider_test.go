package internal

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Collibra user https://access-governance-e2e-1.collibra.tech/profile/0199b8bd-c778-7b8b-b84f-dbc6a7801fab
const TestUser1Email = "terraform-acc-test-1@collibra.com"

// The access governance user id for Collibra user https://access-governance-e2e-1.collibra.tech/profile/0199b8bd-c778-7b8b-b84f-dbc6a7801fab
const TestUser1Id = "jS5llU1fm18LdcCmLy95w"

// Collibra user https://access-governance-e2e-1.collibra.tech/profile/0199b8be-1f09-7e7a-93e6-86c5bab29c4c
const TestUser2Email = "terraform-acc-test-2@collibra.com"

// The access governance user id for Collibra user https://access-governance-e2e-1.collibra.tech/profile/0199b8be-1f09-7e7a-93e6-86c5bab29c4c
const TestUser2Id = "E6cwZLI9F6ys-WYso7HoD"

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"collibra-access-governance": providerserver.NewProtocol6WithError(New("test")()),
}

var providerConfig = `
variable "collibra_user" {
	type = string
}

variable "collibra_secret" {
    type = string
}

variable "collibra_url" {
    type = string
}

provider "collibra-access-governance" {
  user         = var.collibra_user
  secret       = var.collibra_secret
  url          = var.collibra_url
}
`

func AccProviderPreCheck(t *testing.T) {
	if v := os.Getenv("TF_VAR_collibra_user"); v == "" {
		t.Fatal("TF_VAR_collibra_user must be set for acceptance testing")
	}

	if v := os.Getenv("TF_VAR_collibra_secret"); v == "" {
		t.Fatal("TF_VAR_collibra_secret must be set for acceptance testing")
	}

	if v := os.Getenv("TF_VAR_collibra_url"); v == "" {
		t.Fatal("TF_VAR_collibra_url must be set for acceptance testing")
	}
}
