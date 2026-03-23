package internal

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccUserDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: false,
		PreCheck: func() {
			AccProviderPreCheck(t)
		},
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "collibra-data-access_user" "terraform-acc-test-1" {
	email = "terraform-acc-test-1@collibra.com"
}

data "collibra-data-access_user" "terraform-acc-test-2" {
	email = "terraform-acc-test-2@collibra.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.collibra-data-access_user.terraform-acc-test-1", "name", "Terraform Provider IT User 1"),
					resource.TestCheckResourceAttr("data.collibra-data-access_user.terraform-acc-test-1", "type", "Human"),
					resource.TestCheckResourceAttrWith("data.collibra-data-access_user.terraform-acc-test-1", "id", func(value string) error {
						if value == "" {
							return errors.New("id is empty")
						}

						return nil
					}),

					resource.TestCheckResourceAttr("data.collibra-data-access_user.terraform-acc-test-2", "name", "Terraform Provider IT User 2"),
					resource.TestCheckResourceAttr("data.collibra-data-access_user.terraform-acc-test-2", "type", "Human"),
					resource.TestCheckResourceAttrWith("data.collibra-data-access_user.terraform-acc-test-2", "id", func(value string) error {
						if value == "" {
							return errors.New("id is empty")
						}

						return nil
					}),
				),
			},
		},
	})
}
