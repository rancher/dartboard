module github.com/rancher/dartboard

go 1.25.0

replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.6.27 // for compatibilty with docker 20.10.x
	github.com/crewjam/saml => github.com/rancher/saml v0.4.14-rancher3
	github.com/docker/distribution => github.com/docker/distribution v2.8.2+incompatible // rancher-machine requires a replace is set
	github.com/docker/docker => github.com/docker/docker v20.10.27+incompatible // rancher-machine requires a replace is set

	github.com/openshift/api => github.com/openshift/api v0.0.0-20191219222812-2987a591a72c
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c
	github.com/rancher/rancher/pkg/apis => github.com/rancher/rancher/pkg/apis v0.0.0-20251216090958-18e340c45365
	github.com/rancher/rancher/pkg/client => github.com/rancher/rancher/pkg/client v0.0.0-20250212213103-5c3550f55322
	// github.com/rancher/shepherd => github.com/rancher/shepherd v0.0.0-20250514182450-235145b2fd2a
	github.com/rancher/shepherd => ../shepherd
	github.com/rancher/tests => github.com/git-ival/rancher-tests v0.0.0-20251223194020-57770a8f364f
	github.com/rancher/tests/actions => github.com/git-ival/rancher-tests/actions v0.0.0-20251223194020-57770a8f364f
	github.com/rancher/tests/interoperability => github.com/git-ival/rancher-tests/interoperability v0.0.0-20251223194020-57770a8f364f

	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.53.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/metric => go.opentelemetry.io/otel/metric v1.28.0
	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.28.0
	go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v1.28.0
	go.opentelemetry.io/proto/otlp => go.opentelemetry.io/proto/otlp v1.3.1

	helm.sh/helm/v3 => github.com/rancher/helm/v3 v3.16.1-rancher1
	k8s.io/api => k8s.io/api v0.34.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.34.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.34.1
	k8s.io/apiserver => k8s.io/apiserver v0.34.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.34.1
	k8s.io/client-go => k8s.io/client-go v0.34.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.34.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.34.1
	k8s.io/code-generator => k8s.io/code-generator v0.34.1
	k8s.io/component-base => k8s.io/component-base v0.34.1
	k8s.io/component-helpers => k8s.io/component-helpers v0.34.1
	k8s.io/controller-manager => k8s.io/controller-manager v0.34.1
	k8s.io/cri-api => k8s.io/cri-api v0.34.1
	k8s.io/cri-client => k8s.io/cri-client v0.34.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.34.1
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.34.1
	k8s.io/endpointslice => k8s.io/endpointslice v0.34.1
	k8s.io/kms => k8s.io/kms v0.34.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.34.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.34.1
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20250710124328-f3f2b991d03b
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.34.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.34.1
	k8s.io/kubectl => k8s.io/kubectl v0.34.1
	k8s.io/kubelet => k8s.io/kubelet v0.34.1
	k8s.io/kubernetes => k8s.io/kubernetes v1.32.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.34.1
	k8s.io/metrics => k8s.io/metrics v0.34.1
	k8s.io/mount-utils => k8s.io/mount-utils v0.34.1
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.34.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.34.1
	oras.land/oras-go => oras.land/oras-go v1.2.2 // for docker 20.10.x compatibility
	sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v1.10.6
)

require (
	al.essio.dev/pkg/shellescape v1.6.0
	github.com/harvester/harvester v1.5.0
	github.com/matoous/go-nanoid/v2 v2.1.0
	github.com/minio/pkg v1.7.5
	github.com/qase-tms/qase-go/pkg/qase-go v1.0.7
	github.com/qase-tms/qase-go/qase-api-client v1.2.1
	github.com/qase-tms/qase-go/qase-api-v2-client v1.1.4
	github.com/rancher/rancher/pkg/apis v0.0.0
	github.com/rancher/shepherd v0.0.0-20251203195144-c9f6483abe67
	github.com/rancher/tests/actions v0.0.0-20251222130938-f37830bc438e
	github.com/rancher/tests/interoperability v0.0.0-20251222130938-f37830bc438e
	github.com/sirupsen/logrus v1.9.3
	github.com/urfave/cli/v2 v2.27.6
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.34.3
	k8s.io/apimachinery v0.34.3
	k8s.io/client-go v12.0.0+incompatible
	kubevirt.io/api v1.6.0
)

require (
	dario.cat/mergo v1.0.1 // indirect
	emperror.dev/errors v0.8.1 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Microsoft/hcsshim v0.12.0-rc.3 // indirect
	github.com/apparentlymart/go-cidr v1.1.0 // indirect
	github.com/aws/aws-sdk-go v1.55.8 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/cisco-open/operator-tools v0.37.0 // indirect
	github.com/containerd/cgroups/v3 v3.0.2 // indirect
	github.com/containerd/containerd v1.7.28 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/creasty/defaults v1.5.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v27.1.1+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evanphx/json-patch v5.9.11+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/harvester/harvester-network-controller v0.3.1 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/iancoleman/orderedmap v0.3.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.7.6 // indirect
	github.com/k8snetworkplumbingwg/whereabouts v0.9.2 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kube-logging/logging-operator v0.0.0-20250424202944-7e1f9aad6e21 // indirect
	github.com/kube-logging/logging-operator/pkg/sdk v0.12.0 // indirect
	github.com/kubereboot/kured v1.13.1 // indirect
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.2.0 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/longhorn/longhorn-manager v1.8.1 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mcuadros/go-version v0.0.0-20190830083331-035f6764e8d2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.5.0 // indirect
	github.com/moby/sys/mount v0.3.3 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/user v0.3.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/opencontainers/runc v1.2.1 // indirect
	github.com/openshift/custom-resource-status v1.1.2 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/sftp v1.13.5 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.82.0 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/rancher/aks-operator v1.13.0-rc.4 // indirect
	github.com/rancher/ali-operator v1.13.0-rc.2 // indirect
	github.com/rancher/apiserver v0.8.0 // indirect
	github.com/rancher/eks-operator v1.13.0-rc.4 // indirect
	github.com/rancher/fleet/pkg/apis v0.15.0-alpha.4 // indirect
	github.com/rancher/gke-operator v1.13.0-rc.3 // indirect
	github.com/rancher/lasso v0.2.5 // indirect
	github.com/rancher/norman v0.8.1 // indirect
	github.com/rancher/rancher v0.0.0-20251203234820-b95b2fb0d738 // indirect
	github.com/rancher/rke v1.8.5 // indirect
	github.com/rancher/system-upgrade-controller/pkg/apis v0.0.0-20250930163923-f2c9e60b1078 // indirect
	github.com/rancher/tfp-automation v0.0.0-20251219210947-f4a1a9882c29 // indirect
	github.com/rancher/wrangler v1.1.2 // indirect
	github.com/rancher/wrangler/v3 v3.3.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.8.0 // indirect
	github.com/spf13/cobra v1.10.1 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	go.etcd.io/etcd/api/v3 v3.6.5 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.5 // indirect
	go.etcd.io/etcd/client/v2 v2.305.21 // indirect
	go.etcd.io/etcd/client/v3 v3.6.5 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/mod v0.28.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/oauth2 v0.32.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/term v0.36.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	golang.org/x/time v0.13.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250825161204-c5933d9347a5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251002232023-7c0ddcbb5797 // indirect
	google.golang.org/grpc v1.75.1 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/validator.v2 v2.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.34.1 // indirect
	k8s.io/apiserver v0.34.1 // indirect
	k8s.io/cli-runtime v0.34.1 // indirect
	k8s.io/component-base v0.34.1 // indirect
	k8s.io/component-helpers v0.34.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-aggregator v0.34.1 // indirect
	k8s.io/kube-openapi v0.31.9 // indirect
	k8s.io/kubectl v0.34.1 // indirect
	k8s.io/kubernetes v1.34.1 // indirect
	k8s.io/pod-security-admission v0.34.1 // indirect
	k8s.io/utils v0.0.0-20251002143259-bc988d571ff4 // indirect
	kubevirt.io/containerized-data-importer-api v1.62.0 // indirect
	kubevirt.io/controller-lifecycle-operator-sdk/api v0.0.0-20220329064328-f3cc58c6ed90 // indirect
	sigs.k8s.io/cli-utils v0.37.2 // indirect
	sigs.k8s.io/cluster-api v1.10.6 // indirect
	sigs.k8s.io/controller-runtime v0.22.3 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/kustomize/api v0.20.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.20.1 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)
