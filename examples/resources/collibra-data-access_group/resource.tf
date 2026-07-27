resource "collibra-data-access_datasource" "ds" {
  name = "exampleDS"
}

resource "collibra-data-access_group" "example" {
  name        = "A Group"
  description = "A simple group"
  state       = "Active"
  who = [
    {
      user : "user1@company.com"
    },
  ]
  data_sources = [data.collibra-data-access_datasource.ds.id]
}
