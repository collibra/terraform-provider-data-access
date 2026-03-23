package internal

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Collibra user https://data-access-e2e-dev.collibra.tech/data-access/resources/identities/jaGCmJHvrM15aHJc2eCBb
const TestUser1Email = "terraform-acc-test-1@collibra.com"

// The data access user id for Collibra user https://data-access-e2e-dev.collibra.tech/data-access/resources/identities/jaGCmJHvrM15aHJc2eCBb
const TestUser1Id = "jaGCmJHvrM15aHJc2eCBb"

// Collibra user https://data-access-e2e-dev.collibra.tech/data-access/resources/identities/1wWONZ059RupQoW5bib1z
const TestUser2Email = "terraform-acc-test-2@collibra.com"

// The data access user id for Collibra user https://data-access-e2e-dev.collibra.tech/data-access/resources/identities/1wWONZ059RupQoW5bib1z
const TestUser2Id = "1wWONZ059RupQoW5bib1z"

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"collibra-data-access": providerserver.NewProtocol6WithError(New("test")()),
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

provider "collibra-data-access" {
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
