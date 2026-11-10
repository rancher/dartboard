# updatecli automation

The dartboard project uses [updatecli] to automate version updates for
dependencies that are not covered by Dependabot.

## Tool

We use updatecli for this automation because of its extensibility and multiple
plugins that allow greater flexibility when automating sequences of conditional
update steps.

For detailed information on how to use updatecli, please consult its
[documentation].

## Scheduled workflow

The automation runs as a GitHub Actions scheduled workflow once per week.
Manual execution of the pipelines can be [triggered] when needed.

## Covered dependencies

The following dependencies are managed by updatecli:

- **Vendored binaries** (`download-vendored-bin.sh`):
  - OpenTofu
  - kubectl
  - Helm
  - k3d

## Project organization

```
updatecli/
├── README.md
├── updatecli.d                            # Update workflows
│   └── update-vendored-binaries           # Vendored binaries update workflow
└── values.d                               # Configuration files
    └── values.yaml                        # Configuration values
```

## Local testing

Local testing of manifests requires:

1. The updatecli binary that can be downloaded from
   [updatecli/updatecli#releases]. Test only with the latest stable version.
2. A GitHub personal fine-grained token.

```shell
export UPDATECLI_GITHUB_TOKEN="your GH token"
updatecli diff --clean --values updatecli/values.d/values.yaml --config updatecli/updatecli.d/update-vendored-binaries/
```

## Contributing

Before contributing, please follow the guidelines provided in this README and
make sure to test locally your changes before opening a PR.

<!-- Links -->
[updatecli]: https://github.com/updatecli/updatecli
[documentation]: https://www.updatecli.io/docs/prologue/introduction/
[triggered]: ../../actions/workflows/updatecli.yml
[updatecli/updatecli#releases]: https://github.com/updatecli/updatecli/releases
