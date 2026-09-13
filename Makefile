.PHONY: build test tidy tf-fmt tf-validate clean

GOFLAGS_LAMBDA = CGO_ENABLED=0 GOOS=linux GOARCH=arm64
BUILD_DIR = build

build: clean
	$(GOFLAGS_LAMBDA) go build -tags lambda.norpc -ldflags="-s -w" -o $(BUILD_DIR)/auth-cpf/bootstrap ./cmd/auth-cpf
	$(GOFLAGS_LAMBDA) go build -tags lambda.norpc -ldflags="-s -w" -o $(BUILD_DIR)/authorizer/bootstrap ./cmd/authorizer
	cd $(BUILD_DIR)/auth-cpf && zip -q -j ../auth-cpf.zip bootstrap
	cd $(BUILD_DIR)/authorizer && zip -q -j ../authorizer.zip bootstrap

test:
	go vet ./...
	go test ./... -cover

tidy:
	go mod tidy

tf-fmt:
	terraform -chdir=terraform fmt -recursive

tf-validate:
	terraform -chdir=terraform init -backend=false -input=false
	terraform -chdir=terraform validate

clean:
	rm -rf $(BUILD_DIR)
