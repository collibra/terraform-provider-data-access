package internal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func TestAccGlobalRoleAssignmentResource(t *testing.T) {
	testId := gonanoid.Must(8)

	t.Run("basic", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			IsUnitTest: false,
			PreCheck: func() {
				AccProviderPreCheck(t)
			},
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow(tfversion.Version1_0_0),
			},
			Steps: []resource.TestStep{
				{
					ResourceName: "raito_global_role_assignment.gra1",
					Config: fmt.Sprintf(`
%[2]s					
					
resource "raito_user" "u1" {
	name = "gra-tfTestUser-%[1]s"
	email = "gra-test-user-%[1]s@raito.io"
	raito_user = true
}					

resource "raito_global_role_assignment" "gra1" {
	role = "Admin"
	user = raito_user.u1.id
}
					`, testId, providerConfig),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_global_role_assignment.gra1", "role", "Admin"),
						resource.TestCheckResourceAttrPair("raito_global_role_assignment.gra1", "user", "raito_user.u1", "id"),
						resource.TestCheckResourceAttrWith("raito_global_role_assignment.gra1", "id", func(value string) error {
							if !strings.HasPrefix(value, "Admin#") {
								return fmt.Errorf("expected id to start with Admin# but is %q", value)
							}

							return nil
						}),
					),
				},
				{
					ResourceName:      "raito_global_role_assignment.gra1",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					ResourceName: "raito_global_role_assignment.gra1",
					Config: fmt.Sprintf(`
%[2]s					
					
resource "raito_user" "u1" {
	name = "gra-tfTestUser-%[1]s"
	email = "gra-test-user-%[1]s@raito.io"
	raito_user = true
}					

resource "raito_global_role_assignment" "gra1" {
	role = "Creator"
	user = raito_user.u1.id
}
					`, testId, providerConfig),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_global_role_assignment.gra1", "role", "Creator"),
						resource.TestCheckResourceAttrPair("raito_global_role_assignment.gra1", "user", "raito_user.u1", "id"),
						resource.TestCheckResourceAttrWith("raito_global_role_assignment.gra1", "id", func(value string) error {
							if !strings.HasPrefix(value, "Creator#") {
								return fmt.Errorf("expected id to start with Creator#")
							}

							return nil
						}),
					),
				},
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		})
	})

	t.Run("multiple assignments", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			IsUnitTest: false,
			PreCheck: func() {
				AccProviderPreCheck(t)
			},
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow(tfversion.Version1_0_0),
			},
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
%[2]s					
					
resource "raito_user" "u1" {
	name = "gra-tfTestUser-%[1]s"
	email = "gra-test-user-%[1]s@raito.io"
	raito_user = true
}					

resource "raito_global_role_assignment" "gra1" {
	role = "Admin"
	user = raito_user.u1.id
}
					
resource "raito_global_role_assignment" "gra2" {
	role = "AccessCreator"
	user = raito_user.u1.id
}
					
					
					`, testId, providerConfig),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_global_role_assignment.gra1", "role", "Admin"),
						resource.TestCheckResourceAttrPair("raito_global_role_assignment.gra1", "user", "raito_user.u1", "id"),
						resource.TestCheckResourceAttrWith("raito_global_role_assignment.gra1", "id", func(value string) error {
							if !strings.HasPrefix(value, "Admin#") {
								return fmt.Errorf("expected id to start with Admin# but is %q", value)
							}

							return nil
						}),

						resource.TestCheckResourceAttr("raito_global_role_assignment.gra2", "role", "AccessCreator"),
						resource.TestCheckResourceAttrPair("raito_global_role_assignment.gra2", "user", "raito_user.u1", "id"),
						resource.TestCheckResourceAttrWith("raito_global_role_assignment.gra2", "id", func(value string) error {
							if !strings.HasPrefix(value, "AccessCreator#") {
								return fmt.Errorf("expected id to start with Creator# but is %q", value)
							}

							return nil
						}),
					),
				},
				{
					ResourceName:      "raito_global_role_assignment.gra1",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		})
	})
}
