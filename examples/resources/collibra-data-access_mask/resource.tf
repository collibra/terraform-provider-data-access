resource "collibra-data-access_datasource" "ds" {
  name = "exampleDS"
}

resource "collibra-data-access_mask" "example" {
  name        = "A Mask"
  description = "A simple mask"
  state       = "Active"
  who = [
    {
      user : "user1@company.com"
    },
  ]
  data_sources = [
    {
      data_source = data.collibra-data-access_datasource.ds.id
      type        = "SHA256"
    }
  ]
  columns = [
    {
      type        = "column"
      path        = ["SOME_DB", "SOME_SCHEMA", "SOME_TABLE", "SOME_COLUMN"]
      data_source = data.collibra-data-access_datasource.ds.id
    }
  ]
}
