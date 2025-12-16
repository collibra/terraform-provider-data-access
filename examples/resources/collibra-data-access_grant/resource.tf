resource "collibra-data-access_datasource" "ds" {
  name = "exampleDS"
}

resource "collibra-data-access_grant" "grant1" {
  name        = "First grant"
  description = "A simple grant"
  state       = "Active"
  who = [
    {
      user : "user1@company.com"
    },
    {
      user : "user2@company.com"
      promise_duration : 604800
    }
  ]
  type = "role"
  data_sources = [
    {
      data_source : data.collibra-data-access_datasource.ds.id
      type : "role"
    }
  ]
  what_data_objects = [
    {
      data_object = {
        type        = "table"
        path        = ["DB1", "SCHEMA1", "TABLE1"]
        data_source = data.collibra-data-access_datasource.ds.id
      }
      permissions : ["SELECT", "INSERT"]
      global_permissions : []
    },
    {
      data_object = {
        type        = "table"
        path        = ["DB1", "SCHEMA1", "TABLE2"]
        data_source = data.collibra-data-access_datasource.ds.id
      }
      permissions : []
      global_permissions : ["READ"]
    }
  ]
}

resource "collibra-data-access_grant" "grant_purpose1" {
  name        = "Grant2"
  description = "Grant with inherited who"
  category    = "purpose"
  state       = "Active"
  who = [
    {
      access_control = collibra-data-access_grant.grant1.id
    }
  ]
  data_sources = [
    {
      data_source : collibra-data-access_datasource.ds.id
      type : "role"
    }
  ]
}