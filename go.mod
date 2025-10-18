module github.com/rancher/dartboard

go 1.24.0

toolchain go1.24.2

replace github.com/qase-tms/qase-go/pkg/qase-go => github.com/git-ival/qase-go/pkg/qase-go v0.0.0-20251017020041-734cab39c3e6

require (
	al.essio.dev/pkg/shellescape v1.5.0
	github.com/qase-tms/qase-go/pkg/qase-go v1.0.3
	github.com/qase-tms/qase-go/qase-api-client v1.2.0
	github.com/qase-tms/qase-go/qase-api-v2-client v1.1.3
	github.com/sirupsen/logrus v1.9.3
	github.com/urfave/cli/v2 v2.27.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/sys v0.34.0 // indirect
	gopkg.in/validator.v2 v2.0.1 // indirect
)
