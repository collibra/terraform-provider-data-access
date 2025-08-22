package internal

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func TestAccUserResource(t *testing.T) {
	testId := gonanoid.MustGenerate("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 8)

	t.Run("start with non-raito user", func(t *testing.T) {
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
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "u1" {
	name = "tfTestUser-%[1]s"
	email = "test-user-%[1]s@raito.io"
	raito_user = false
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_user.u1", "name", fmt.Sprintf("tfTestUser-%s", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "email", fmt.Sprintf("test-user-%s@raito.io", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "raito_user", "false"),
						resource.TestCheckResourceAttr("raito_user.u1", "type", "Human"),
						resource.TestCheckNoResourceAttr("raito_user.u1", "password"),
						resource.TestCheckResourceAttr("raito_user.u1", "roles.#", "0"),
						resource.TestCheckResourceAttrWith("raito_user.u1", "id", func(value string) error {
							if len(value) == 0 {
								return fmt.Errorf("ID should not be empty")
							}

							return nil
						}),
					),
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "u1" {
	name = "tfTestUser-%[1]s"
	email = "test-user-%[1]s@raito.io"
	raito_user = false
	type = "Machine"
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_user.u1", "name", fmt.Sprintf("tfTestUser-%s", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "email", fmt.Sprintf("test-user-%s@raito.io", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "raito_user", "false"),
						resource.TestCheckResourceAttr("raito_user.u1", "type", "Machine"),
						resource.TestCheckNoResourceAttr("raito_user.u1", "password"),
						resource.TestCheckResourceAttr("raito_user.u1", "roles.#", "0"),
						resource.TestCheckResourceAttrWith("raito_user.u1", "id", func(value string) error {
							if len(value) == 0 {
								return fmt.Errorf("ID should not be empty")
							}

							return nil
						}),
					),
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "u1" {
	name = "tfTestUser-%[1]s"
	email = "test-user-%[1]s@raito.io"
	raito_user = true
	type = "Machine"
	password = "!23vV678"
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_user.u1", "name", fmt.Sprintf("tfTestUser-%s", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "email", fmt.Sprintf("test-user-%s@raito.io", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "raito_user", "true"),
						resource.TestCheckResourceAttr("raito_user.u1", "type", "Machine"),
						resource.TestCheckResourceAttr("raito_user.u1", "password", "!23vV678"),
						resource.TestCheckResourceAttrWith("raito_user.u1", "id", func(value string) error {
							if len(value) == 0 {
								return fmt.Errorf("ID should not be empty")
							}

							return nil
						}),
					),
				},
			},
		})
	})

	t.Run("user with password write only", func(t *testing.T) {
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
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "u1" {
	name = "tfTestUser-%[1]s"
	email = "test-user-%[1]s@raito.io"
	raito_user = true
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_user.u1", "name", fmt.Sprintf("tfTestUser-%s", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "email", fmt.Sprintf("test-user-%s@raito.io", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "raito_user", "true"),
						resource.TestCheckResourceAttr("raito_user.u1", "type", "Human"),
						resource.TestCheckNoResourceAttr("raito_user.u1", "password"),
						resource.TestCheckNoResourceAttr("raito_user.u1", "password_wo"),
						resource.TestCheckNoResourceAttr("raito_user.u1", "password_wo_version"),
						resource.TestCheckResourceAttrWith("raito_user.u1", "id", func(value string) error {
							if len(value) == 0 {
								return fmt.Errorf("ID should not be empty")
							}

							return nil
						}),
					),
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "u1" {
	name = "tfTestUser-%[1]s"
	email = "test-user-%[1]s@raito.io"
	raito_user = true
	type = "Machine"
	password_wo = "!23vV678"
	password_wo_version = 1
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_user.u1", "name", fmt.Sprintf("tfTestUser-%s", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "email", fmt.Sprintf("test-user-%s@raito.io", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "raito_user", "true"),
						resource.TestCheckResourceAttr("raito_user.u1", "type", "Machine"),
						resource.TestCheckNoResourceAttr("raito_user.u1", "password"),
						resource.TestCheckNoResourceAttr("raito_user.u1", "password_wo"),
						resource.TestCheckResourceAttr("raito_user.u1", "password_wo_version", "1"),
						resource.TestCheckResourceAttrWith("raito_user.u1", "id", func(value string) error {
							if len(value) == 0 {
								return fmt.Errorf("ID should not be empty")
							}

							return nil
						}),
					),
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "u1" {
	name = "tfTestUser-%[1]s"
	email = "test-user-%[1]s@raito.io"
	raito_user = true
	type = "Machine"
	password_wo = "!23vV679"
	password_wo_version = 2
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_user.u1", "name", fmt.Sprintf("tfTestUser-%s", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "email", fmt.Sprintf("test-user-%s@raito.io", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "raito_user", "true"),
						resource.TestCheckResourceAttr("raito_user.u1", "type", "Machine"),
						resource.TestCheckNoResourceAttr("raito_user.u1", "password"),
						resource.TestCheckNoResourceAttr("raito_user.u1", "password_wo"),
						resource.TestCheckResourceAttr("raito_user.u1", "password_wo_version", "2"),
						resource.TestCheckResourceAttrWith("raito_user.u1", "id", func(value string) error {
							if len(value) == 0 {
								return fmt.Errorf("ID should not be empty")
							}

							return nil
						}),
					),
				},
			},
		})
	})

	t.Run("start with raito user", func(t *testing.T) {
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
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "u1" {
	name = "tfTestUser-%[1]s"
	email = "test-user-%[1]s@raito.io"
	raito_user = true
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_user.u1", "name", fmt.Sprintf("tfTestUser-%s", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "email", fmt.Sprintf("test-user-%s@raito.io", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "raito_user", "true"),
						resource.TestCheckResourceAttr("raito_user.u1", "type", "Human"),
						resource.TestCheckNoResourceAttr("raito_user.u1", "password"),
						resource.TestCheckResourceAttrWith("raito_user.u1", "id", func(value string) error {
							if len(value) == 0 {
								return fmt.Errorf("ID should not be empty")
							}

							return nil
						}),
					),
				},
				{
					Config: providerConfig + fmt.Sprintf(`
resource "raito_user" "u1" {
	name = "tfTestUser-%[1]s"
	email = "test-user-%[1]s@raito.io"
	raito_user = true
	type = "Machine"
	password = "!23vV678"
}
`, testId),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_user.u1", "name", fmt.Sprintf("tfTestUser-%s", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "email", fmt.Sprintf("test-user-%s@raito.io", testId)),
						resource.TestCheckResourceAttr("raito_user.u1", "raito_user", "true"),
						resource.TestCheckResourceAttr("raito_user.u1", "type", "Machine"),
						resource.TestCheckResourceAttr("raito_user.u1", "password", "!23vV678"),
						resource.TestCheckResourceAttrWith("raito_user.u1", "id", func(value string) error {
							if len(value) == 0 {
								return fmt.Errorf("ID should not be empty")
							}

							return nil
						}),
					),
				},
			},
		})
	})
}
