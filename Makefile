BINARY    = terraform-provider-openstatus
HOSTNAME  = registry.terraform.io
NAMESPACE = openstatusHQ
TYPE      = openstatus
VERSION   = 0.0.0-dev
OS_ARCH   = $(shell go env GOOS)_$(shell go env GOARCH)
INSTALL_DIR = ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(TYPE)/$(VERSION)/$(OS_ARCH)

.PHONY: build test testacc install clean

build:
	go build -o $(BINARY)

test:
	go test ./...

testacc:
	@echo "TODO: acceptance tests not implemented yet"

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/

clean:
	rm -f $(BINARY)
