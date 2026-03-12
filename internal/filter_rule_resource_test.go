package internal

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccFilterRuleResource(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
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
					ResourceName: "collibra-data-access_filter_rule.test_rule",
					Config: providerConfig + `
				data "collibra-data-access_datasource" "ds" {
				   name = "Snowflake"
				}

				resource "collibra-data-access_filter_rule" "test_rule" {
					name = "tfTestFilterRule"
					description = "Filter rule for testing purposes"
					filter_policy = "{Category} = 'Reseller'"
				}
				
				resource "collibra-data-access_filter" "test" {
					name        = "tfTestFilter"
				    description = "filter description"
					table = {
						type: "table"
						path: ["MASTER_DATA","SALES","SPECIALOFFER"]
						data_source: data.collibra-data-access_datasource.ds.id
					}
					filter_rules = [ collibra-data-access_filter_rule.test_rule.id ]
				}
				`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "name", "tfTestFilterRule"),
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "description", "Filter rule for testing purposes"),
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "filter_policy", "{Category} = 'Reseller'"),

						resource.TestCheckNoResourceAttr("collibra-data-access_filter_rule.test_rule", "who"),
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "who_locked", "false"),
					),
				},
				{
					ResourceName: "collibra-data-access_filter_rule.test_rule",
					Config: providerConfig + `
				data "collibra-data-access_datasource" "ds" {
				   name = "Snowflake"
				}

				resource "collibra-data-access_filter_rule" "test_rule" {
					name = "tfTestFilterRule"
					description = "Filter rule for testing purposes"
					who = [
						{
							"user": "terraform-acc-test-1@collibra.com"
						}
					]
					filter_policy = "{Category} = 'Reseller'"
				}
				
				resource "collibra-data-access_filter" "test" {
					name        = "tfTestFilter"
				    description = "filter description"
					table = {
						type: "table"
						path: ["MASTER_DATA","SALES","SPECIALOFFER"]
						data_source: data.collibra-data-access_datasource.ds.id
					}
					filter_rules = [ collibra-data-access_filter_rule.test_rule.id ]
				}
				`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "name", "tfTestFilterRule"),
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "description", "Filter rule for testing purposes"),
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "filter_policy", "{Category} = 'Reseller'"),

						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "who.#", "1"),
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "who.0.user", "terraform-acc-test-1@collibra.com"),
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "who_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-data-access_filter_rule.test_rule",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who"},
				},
				{
					ResourceName: "collibra-data-access_filter_rule.test_rule",
					Config: providerConfig + `
				data "collibra-data-access_datasource" "ds" {
				   name = "Snowflake"
				}

				resource "collibra-data-access_filter_rule" "test_rule" {
					name = "tfTestFilterRule"
					description = "Filter rule for testing purposes"
					filter_policy = "{Category} = 'Reseller'"
					who_locked = true
					inheritance_locked = true
				}
				
				resource "collibra-data-access_filter" "test" {
					name        = "tfTestFilter"
				    description = "filter description"
					table = {
						type: "table"
						path: ["MASTER_DATA","SALES","SPECIALOFFER"]
						data_source: data.collibra-data-access_datasource.ds.id
					}
					filter_rules = [ collibra-data-access_filter_rule.test_rule.id ]
				}
				`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "name", "tfTestFilterRule"),
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "description", "Filter rule for testing purposes"),
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "filter_policy", "{Category} = 'Reseller'"),

						resource.TestCheckNoResourceAttr("collibra-data-access_filter_rule.test_rule", "who"),

						resource.TestCheckNoResourceAttr("collibra-data-access_filter_rule.test_rule", "who"),
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-data-access_filter_rule.test_rule", "inheritance_locked", "true"),
					),
				},
			},
		})
	})
}
