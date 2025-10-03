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
				Config: providerConfig + `data "collibra-access-governance_user" "carla" {
	email = "c_harris@collibra.com"
}

data "collibra-access-governance_user" "angelica" {
	email = "a_abbotatkinson7576@collibra.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.collibra-access-governance_user.carla", "name", "Carla Harris"),
					resource.TestCheckResourceAttr("data.collibra-access-governance_user.carla", "type", "Human"),
					resource.TestCheckResourceAttr("data.collibra-access-governance_user.carla", "collibra_user", "true"),
					resource.TestCheckResourceAttrWith("data.collibra-access-governance_user.carla", "id", func(value string) error {
						if value == "" {
							return errors.New("id is empty")
						}

						return nil
					}),

					resource.TestCheckResourceAttr("data.collibra-access-governance_user.angelica", "name", "Angelica Abbot Atkinson"),
					resource.TestCheckResourceAttr("data.collibra-access-governance_user.angelica", "type", "Machine"),
					resource.TestCheckResourceAttr("data.collibra-access-governance_user.angelica", "collibra_user", "false"),
					resource.TestCheckResourceAttrWith("data.collibra-access-governance_user.angelica", "id", func(value string) error {
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
