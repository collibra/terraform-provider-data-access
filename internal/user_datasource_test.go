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
				Config: providerConfig + `data "raito_user" "carla" {
	email = "c_harris@raito.io"
}

data "raito_user" "angelica" {
	email = "a_abbotatkinson7576@raito.io"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.raito_user.carla", "name", "Carla Harris"),
					resource.TestCheckResourceAttr("data.raito_user.carla", "type", "Human"),
					resource.TestCheckResourceAttr("data.raito_user.carla", "raito_user", "true"),
					resource.TestCheckResourceAttrWith("data.raito_user.carla", "id", func(value string) error {
						if value == "" {
							return errors.New("id is empty")
						}

						return nil
					}),

					resource.TestCheckResourceAttr("data.raito_user.angelica", "name", "Angelica Abbot Atkinson"),
					resource.TestCheckResourceAttr("data.raito_user.angelica", "type", "Machine"),
					resource.TestCheckResourceAttr("data.raito_user.angelica", "raito_user", "false"),
					resource.TestCheckResourceAttrWith("data.raito_user.angelica", "id", func(value string) error {
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
