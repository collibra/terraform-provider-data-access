resource "collibra-data-access_datasource" "ds" {
  name = "exampleDS"
}

resource "collibra-data-access_purpose" "purpose1" {
  name        = "Purpose1"
  description = "Purpose"
  state       = "Active"
  who = [
    {
      user : "user1@company.com"
    }
  ]
}

resource "collibra-data-access_filter" "filter1" {
  name        = "First filter"
  description = "A simple filter"
  state       = "Active"
  who = [
    {
      user : "user1@company.com"
    },
    {
      user : "user2@company.com"
      promise_duration : 604800
    },
    {
      access_control : collibra-data-access_purpose.purpose1.id
    }
  ]
  table = {
    type        = "table"
    path        = ["DB1", "SCHEMA1", "TABLE1"]
    data_source = data.collibra-data-access_datasource.ds.id
  }
  filter_policy = "{state} = 'NJ'"
}