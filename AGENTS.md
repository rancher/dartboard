# Agent Instructions

This file contains instructions for AI agents working on this repository.

## Verification Commands

Before submitting any changes, run:

```bash
make verify-all  # Run all verification steps
```

## Workflow

1. Make changes
2. Run `make verify-all`
3. Fix any errors
4. Repeat until all checks pass
5. Submit PR

## Key Files

- `Makefile`: All build and verification targets
- `.golangci.yml`: Go linter configuration
- `tofu/main/`: OpenTofu main configurations (k3d, aws, azure, harvester)
- `tofu/modules/`: Reusable OpenTofu modules

## Common Issues

### Go Module Errors

Run `go mod tidy` then `make go-mod-verify`.

### OpenTofu Format Errors

Run `make tofu-fmt` to auto-format, then `make tofu-fmt-check`.

### Lint Errors

Check `.golangci.yml` for enabled linters. Fix issues or add justification for exclusions.
