# Terraform Pre-Commit Hook

This directory contains scripts for local Terraform validation before committing code.

## Quick Start

To install the pre-commit hook:

```bash
./scripts/install-pre-commit-hook.sh
```

This will automatically run Terraform validation tests before each commit.

## What It Does

The pre-commit hook runs the following checks:
- **Terraform Format Check** (`terraform fmt -check`)
- **Terraform Validation** (`terraform init` and `terraform validate`)

## Requirements

- **Go** (for running terratest)
- **Terraform** (for validation)

## Manual Testing

To run the tests manually without committing:

```bash
cd tests
export TERRAFORM_DIR=$(pwd)/..
export TF_VAR_ibmcloud_api_key="dummy-key-for-validation"
go test -v -timeout 5m
```

## Skipping the Hook

If you need to skip the pre-commit hook (not recommended):

```bash
git commit --no-verify
```

## Uninstalling

To remove the pre-commit hook:

```bash
rm .git/hooks/pre-commit
```

## Files

- `pre-commit-hook.sh` - The actual hook script that runs tests
- `install-pre-commit-hook.sh` - Installation script that sets up the hook
- `README.md` - This file

## Troubleshooting

### Tests are slow
The first run downloads Terraform providers and may take 20-30 seconds. Subsequent runs are faster (2-3 seconds) as providers are cached.

### Go not found
Install Go from https://go.dev/dl/ or use your package manager:
```bash
# macOS
brew install go

# Linux
sudo apt-get install golang  # Debian/Ubuntu
sudo yum install golang       # RHEL/CentOS
```

### Terraform not found
Install Terraform from https://www.terraform.io/downloads or use your package manager:
```bash
# macOS
brew install terraform

# Linux
sudo apt-get install terraform  # Debian/Ubuntu