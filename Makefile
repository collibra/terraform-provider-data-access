make gotestsum := go run gotest.tools/gotestsum@latest

generate:
	go generate ./...

lint:
	golangci-lint run ./...
	go fmt ./...

testacc:
	TF_ACC=1 ${gotestsum} ./... -v $(TESTARGS) -timeout 120m