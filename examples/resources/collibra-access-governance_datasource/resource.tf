resource "collibra-access-governance_datasource" "example" {
  name        = "DataSourceName"
  description = "A description for the data source"
  sync_method = "ON_PREM"
  parent      = "ParentId"
}