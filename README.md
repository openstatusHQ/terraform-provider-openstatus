# OpenStatus Terraform

Our Terraform Providers to manage your OpenStatus monitors.


## Development

### Generate OpenAPI Spec

```
tfplugingen-openapi generate \
  --config generator_config.yml \
  --output provider-code-spec.json \
  openapi.json
```

### Generate Terraform Provider

```
 tfplugingen-framework generate all \
    --input provider-code-spec.json \
    --output internal
```