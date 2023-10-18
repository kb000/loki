# Deprecated Config Checker

This script can check your configuration files for deprecated and deleted options.

## Usage

Run the script with `-help` for a list of options.

### Example

```bash
go run tools/deprecated-config-checker/main.go \
  -config.file tools/deprecated-config-checker/test-fixtures/config.yaml \
  -runtime-config.file tools/deprecated-config-checker/test-fixtures/runtime-config.yaml
```