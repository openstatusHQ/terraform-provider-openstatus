default: build

build:
    go build -o terraform-provider-openstatus

# Local iteration: build and use the binary directly via dev_overrides.
# Bypasses Terraform's lockfile / plugin cache so rebuilds are picked up
# immediately. Run once to set up ~/.terraformrc, then just `just build`
# between iterations.
dev: build
    #!/usr/bin/env sh
    set -e
    rc="${HOME}/.terraformrc"
    repo="$(pwd)"
    snippet="provider_installation {\n  dev_overrides {\n    \"openstatusHQ/openstatus\" = \"${repo}\"\n  }\n  direct {}\n}"
    if [ ! -f "$rc" ]; then
        printf "%b\n" "$snippet" > "$rc"
        echo "Wrote $rc with dev override -> $repo"
    elif grep -q "openstatusHQ/openstatus" "$rc"; then
        echo "$rc already references openstatusHQ/openstatus; leaving it alone."
        echo "Verify the override path points at: $repo"
    else
        echo "$rc exists and has no openstatus override. Add this manually:"
        echo ""
        printf "%b\n" "$snippet"
    fi

# Install the built binary into Terraform's user plugin directory. Only
# useful for non-dev-override workflows; the published lockfile hash will
# reject this binary unless you also `terraform init -upgrade`. Prefer
# `just dev` for local iteration.
install: build
    #!/usr/bin/env sh
    dir=~/.terraform.d/plugins/registry.terraform.io/openstatusHQ/openstatus/0.1.1/$(go env GOOS)_$(go env GOARCH)
    mkdir -p "$dir"
    cp terraform-provider-openstatus "$dir"/

test:
    go test ./... -v

testacc:
    TF_ACC=1 go test ./... -v -timeout 120m

lint:
    golangci-lint run ./...
