module github.com/rancher/dartboard

go 1.24.0

toolchain go1.24.2

replace go.qase.io/client => github.com/rancher/qase-go/client v0.0.0-20250627195016-142ff3dfec16

require (
	al.essio.dev/pkg/shellescape v1.5.0
	github.com/rancher/tests/actions v0.0.0-20251002210344-0f42b2030fa8
	github.com/sirupsen/logrus v1.9.3
	github.com/urfave/cli/v2 v2.27.1
	go.qase.io/client v0.0.4
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/antihax/optional v1.0.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
