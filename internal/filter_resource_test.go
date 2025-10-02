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
					ResourceName: "collibra-access-governance_filter.test",
					Config: providerConfig + `
				data "collibra-access-governance_datasource" "ds" {
				   name = "Snowflake"
				}
				
				resource "collibra-access-governance_filter" "test" {
					name        = "tfTestFilter"
				   description = "filter description"
					data_source = data.collibra-access-governance_datasource.ds.id
					table = "MASTER_DATA.SALES.SPECIALOFFER"
					who = [
						{
							"user": "terraform@collibra.com"
						}
					]
					filter_policy = "{Category} = 'Reseller'"
				}
				`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "name", "tfTestFilter"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "description", "filter description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_filter.test", "data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "table", "MASTER_DATA.SALES.SPECIALOFFER"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "filter_policy", "{Category} = 'Reseller'"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "what_locked", "true"),
					),
				},
				{
					ResourceName:            "collibra-access-governance_filter.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "table"},
				},
				{
					ResourceName: "collibra-access-governance_filter.test",
					Config: providerConfig + `
				data "collibra-access-governance_datasource" "ds" {
				   name = "Snowflake"
				}
				
				resource "collibra-access-governance_filter" "test" {
					name        = "tfTestFilter"
				   description = "filter description"
					data_source = data.collibra-access-governance_datasource.ds.id
					filter_policy = "{Category} = 'Reseller'"
				}
				`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "name", "tfTestFilter"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "description", "filter description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_filter.test", "data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_filter.test", "table"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "filter_policy", "{Category} = 'Reseller'"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "who_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "what_locked", "false"),
					),
				},
				{
					ResourceName: "collibra-access-governance_filter.test",
					Config: providerConfig + `
data "collibra-access-governance_datasource" "ds" {
    name = "Snowflake"
}

resource "collibra-access-governance_filter" "test" {
	name        = "tfTestFilter"
    description = "filter description"
	data_source = data.collibra-access-governance_datasource.ds.id
	filter_policy = "{Category} = 'Reseller'"
	what_locked = false
	who = [
		{
			"user": "terraform@collibra.com"
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "name", "tfTestFilter"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "description", "filter description"),
						resource.TestCheckResourceAttrPair("collibra-access-governance_filter.test", "data_source", "data.collibra-access-governance_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("collibra-access-governance_filter.test", "table"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "filter_policy", "{Category} = 'Reseller'"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "who_locked", "true"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "inheritance_locked", "false"),
						resource.TestCheckResourceAttr("collibra-access-governance_filter.test", "what_locked", "false"),
					),
				},
			},
		})
	})
}
