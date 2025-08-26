resource "raito_datasource" "ds" {
  name = "exampleDS"
}

resource "raito_grant" "grant1" {
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
  data_source = [
    {
      data_source : raito_datasource.ds.id
      type : "role"
    }
  ]
  what_data_objects = {
    data_object : [
      {
        name : "data_object1"
        permissions : ["SELECT", "INSERT"]
        global_permissions : []
      },
      {
        name : "data_object2"
        global_permissions : ["READ"]
      }
    ]
  }
}

resource "raito_grant" "grant_purpose1" {
  name        = "Grant2"
  description = "Grant with inherited who"
  category    = "purpose"
  state       = "Active"
  who = [
    {
      access_control = raito_grant.grant1.id
    }
  ]
  data_source = [
    {
      data_source : raito_datasource.ds.id
      type : "role"
    }
  ]
}