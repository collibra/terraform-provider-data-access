package internal

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccFilterResource(t *testing.T) {
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
					ResourceName: "collibra-data-access_filter.test",
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
					data_source = data.collibra-data-access_datasource.ds.id
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
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "name", "tfTestFilter"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "description", "filter description"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "table.type", "table"),
						resource.TestCheckResourceAttrPair("collibra-data-access_filter.test", "table.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "filter_rules.#", "1"),
						resource.TestCheckResourceAttrPair("collibra-data-access_filter.test", "filter_rules.0", "collibra-data-access_filter_rule.test_rule", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-data-access_filter.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"table", "filter_rules"},
				},
				{
					ResourceName: "collibra-data-access_filter.test",
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
									data_source = data.collibra-data-access_datasource.ds.id
									filter_policy = "{Category} = 'Reseller'"
								}

								resource "collibra-data-access_filter_rule" "test_rule_2" {
									name = "tfTestFilterRule2"
									description = "Filter rule for testing purposes"
									data_source = data.collibra-data-access_datasource.ds.id
									filter_policy = "{Category} = 'Sales'"
								}
				
								resource "collibra-data-access_filter" "test" {
									name        = "tfTestFilter"
								    description = "filter description"
									table = {
										type: "table"
										path: ["MASTER_DATA","SALES","SPECIALOFFER"]
										data_source: data.collibra-data-access_datasource.ds.id
									}
									filter_rules = [ collibra-data-access_filter_rule.test_rule.id, collibra-data-access_filter_rule.test_rule_2.id ]
								}
								`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "name", "tfTestFilter"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "description", "filter description"),
						resource.TestCheckResourceAttrPair("collibra-data-access_filter.test", "table.data_source", "data.collibra-data-access_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "filter_rules.#", "2"),
						resource.TestCheckResourceAttr("collibra-data-access_filter.test", "what_locked", "true"),
					),
				},
			},
		})
	})
}
