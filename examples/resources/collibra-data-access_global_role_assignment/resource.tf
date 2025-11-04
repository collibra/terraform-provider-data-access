resource "collibra-data-access_user" "u1" {
  name          = "user name"
  email         = "test-user@collibra.com"
  collibra_user = true
  type          = "Machine"
  password      = "!23vV678"
}

resource "collibra-data-access_global_role_assignment" "u1_admin" {
  user = collibra-data-access_user.u1.id
  role = "Admin"
}

resource "collibra-data-access_global_role_assignment" "u1_creator" {
  user = collibra-data-access_user.u1.id
  role = "Creator"
}