GO := go
go_lint := golangci-lint

make gotestsum := go run gotest.tools/gotestsum@latest

generate:
	$(GO) generate ./...

lint:
	$(go_lint) run ./...
	$(GO) fmt ./...

fix-lint:
	$(GO) fmt ./...
	$(go_lint) run --fix ./...

testacc:
	TF_ACC=1 ${gotestsum} ./... -v $(TESTARGS) -timeout 120m

clean-gitignore:
	@echo "Removing ignored Go files..."
	git clean -fX -- "**/*.go"
