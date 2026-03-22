package monitor

import (
	"terraform-provider-openstatus/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
)

type providerConfig = client.ProviderConfig

var idPath = path.Root("id")
