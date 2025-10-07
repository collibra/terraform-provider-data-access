resource "collibra-access-governance_user" "u1" {
  name                            = "user name"
  email                           = "test-user@collibra.com"
  collibra_user                   = true
  type                            = "Machine"
  password                        = "!23vV678"
}

resource "collibra-access-governance_global_role_assignment" "u1_admin" {
  user = collibra-access-governance_user.u1.id
  role = "Admin"
}

resource "collibra-access-governance_global_role_assignment" "u1_creator" {
  user = collibra-access-governance_user.u1.id
  role = "Creator"
}