# CLAUDE.md

Terraform provider (terraform-plugin-framework, protocol v6) for Collibra Data Access, built on `github.com/collibra/data-access-go-sdk`.

## Commands

- `make generate` regenerates `docs/` and the `*_enumer.go` files. It needs the `terraform` CLI on PATH, and CI fails on any diff, so commit the result. Change `templates/`, `examples/` or the `//go:generate` line, never `docs/` or `*_enumer.go`.
- Acceptance tests (`make testacc`, or `TF_ACC=1 go test ./internal/ -run TestAccGrantResource`) hit a live Collibra environment and need `TF_VAR_collibra_user`, `TF_VAR_collibra_secret` and `TF_VAR_collibra_url`. There are no unit tests, only `TestAcc*` cases.

## Architecture

- Access-control resources embed the generic `AccessControlResource` (`internal/access_control_resource.go`; `mask_resource.go` is a compact example): implement `AccessControlModel` and pass read, validation and plan-modifier hooks in the constructor. Don't write standalone CRUD for an access control.
- Register new resources and data sources in `Resources()` / `DataSources()` in `internal/provider.go`.
