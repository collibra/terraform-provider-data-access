resource "raito_datasource" "ds" {
  name = "exampleDS"
}

resource "raito_grant_category" "example_category" {
  name                   = "exampleCategory"
  description            = "A simple category"
  icon                   = "testIcon"
  can_create             = true
  allow_duplicated_names = true
  multi_data_source      = true
  default_type_per_data_source = [
    {
      data_source : raito_datasource.ds.id
      type : "table"
    }
  ]
  allowed_who_items = {
    user        = true
    group       = true
    inheritance = true
    self        = true
    categories  = ["otherCategoryId"]
  }
  allowed_what_items = {
    data_object = true
  }
}