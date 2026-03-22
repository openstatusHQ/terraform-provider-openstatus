default: build

build:
    go build -o terraform-provider-openstatus

install: build
    #!/usr/bin/env sh
    dir=~/.terraform.d/plugins/registry.terraform.io/openstatusHQ/openstatus/0.0.1/$(go env GOOS)_$(go env GOARCH)
    mkdir -p "$dir"
    cp terraform-provider-openstatus "$dir"/

test:
    go test ./... -v

testacc:
    TF_ACC=1 go test ./... -v -timeout 120m

lint:
    golangci-lint run ./...
