resource "raito_datasource" "example" {
  name        = "DataSourceName"
  description = "A description for the data source"
  sync_method = "ON_PREM"
  parent      = "ParentId"
  identity_stores = [
    "linked_identity_store_id"
  ]
}