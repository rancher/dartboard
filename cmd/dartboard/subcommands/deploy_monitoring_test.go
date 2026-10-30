package subcommands

import (
	"reflect"
	"strings"
	"testing"
)

func TestMonitoringNodeExporterAndReservedNodeScheduling(t *testing.T) {
	for _, reserved := range []bool{false, true} {
		legacy := getRancherMonitoringValsJSON(reserved, "")
		stack := getKubePrometheusStackVals(reserved, false, false, "", "")
		for _, values := range []map[string]any{legacy, stack} {
			exporter := values["prometheus-node-exporter"].(map[string]any)
			if !reflect.DeepEqual(exporter["nodeSelector"], map[string]any{"kubernetes.io/os": "linux"}) {
				t.Fatalf("reserved=%t: node exporter must cover all Linux nodes: %#v", reserved, exporter)
			}
			if !reflect.DeepEqual(exporter["tolerations"], []any{map[string]any{"operator": "Exists"}}) {
				t.Fatalf("reserved=%t: node exporter must tolerate control-plane and monitoring taints", reserved)
			}
		}
		wantSelector := map[string]any{}
		wantTolerations := []any{}
		if reserved {
			wantSelector["monitoring"] = "true"
			wantTolerations = []any{map[string]any{"key": "monitoring", "operator": "Exists", "effect": "NoSchedule"}}
		}
		operator := stack["prometheusOperator"].(map[string]any)
		webhooks := operator["admissionWebhooks"].(map[string]any)
		for _, component := range []map[string]any{
			stack["grafana"].(map[string]any),
			stack["prometheus"].(map[string]any)["prometheusSpec"].(map[string]any),
			stack["kube-state-metrics"].(map[string]any), operator,
			webhooks["patch"].(map[string]any), webhooks["deployment"].(map[string]any),
		} {
			if !reflect.DeepEqual(component["nodeSelector"], wantSelector) || !reflect.DeepEqual(component["tolerations"], wantTolerations) {
				t.Fatalf("reserved=%t: incorrect monitoring component scheduling: %#v", reserved, component)
			}
		}
	}
}

func TestMonitoringDashboardsChartPathUsesReleaseRepo(t *testing.T) {
	got := monitoringDashboardsChartPath("https://github.com/rancher/charts/raw/refs/heads/release-v2.15", "110.0.1+up0.1.4")

	want := "https://github.com/rancher/charts/raw/refs/heads/release-v2.15/assets/rancher-monitoring-dashboards/rancher-monitoring-dashboards-110.0.1+up0.1.4.tgz"
	if got != want {
		t.Fatalf("monitoringDashboardsChartPath() = %q, want %q", got, want)
	}
}

func TestKubePrometheusStackValuesKeepMonitoringIntegration(t *testing.T) {
	values := getKubePrometheusStackVals(true, false, false, "", "http://mimir.example/mimir/api/v1/push")

	prometheus := values["prometheus"].(map[string]any)["prometheusSpec"].(map[string]any)
	if prometheus["serviceMonitorSelectorNilUsesHelmValues"] != false || prometheus["podMonitorSelectorNilUsesHelmValues"] != false {
		t.Fatal("Prometheus must discover ServiceMonitors and PodMonitors without Helm release labels")
	}

	if len(prometheus["remoteWrite"].([]any)) != 1 || len(prometheus["additionalScrapeConfigs"].([]any)) != 1 {
		t.Fatal("Prometheus must keep Mimir remote write and cgroups exporter scrape")
	}

	if values["kubeEtcd"].(map[string]any)["enabled"] != false || values["kubeProxy"].(map[string]any)["enabled"] != false {
		t.Fatal("unsupported control-plane scrapes must remain disabled")
	}

	if prometheus["nodeSelector"].(map[string]any)["monitoring"] != "true" {
		t.Fatal("Prometheus must use reserved monitoring node")
	}
}

func TestDashboardValuesDisableMissingAlertmanagerProxy(t *testing.T) {
	values := getMonitoringDashboardsValues(true, false)
	if values["alertmanagerProxy"].(map[string]any)["enabled"] != false {
		t.Fatal("dashboard chart must not proxy to disabled kube-prometheus-stack Alertmanager")
	}

	global := values["global"].(map[string]any)
	if global["disableProxyIPv6"] != true {
		t.Fatal("dashboard proxy IPv6 must be disabled by default")
	}

	if _, nested := global["cattle"].(map[string]any)["disableProxyIPv6"]; nested {
		t.Fatal("dashboard proxy IPv6 setting must not be nested under global.cattle")
	}
}

func TestMonitoringIPv6DefaultsOffAndCanBeEnabled(t *testing.T) {
	values := getKubePrometheusStackVals(false, false, false, "", "")

	service := values["prometheus"].(map[string]any)["service"].(map[string]any)["ipDualStack"].(map[string]any)
	if service["enabled"] != false || service["ipFamilyPolicy"] != "SingleStack" {
		t.Fatalf("IPv6-disabled service values = %#v", service)
	}

	if values["prometheus-node-exporter"].(map[string]any)["service"].(map[string]any)["ipDualStack"].(map[string]any)["enabled"] != false {
		t.Fatal("node exporter dual-stack must be disabled by default")
	}

	values = getKubePrometheusStackVals(false, false, true, "", "")

	service = values["prometheus"].(map[string]any)["service"].(map[string]any)["ipDualStack"].(map[string]any)
	if service["enabled"] != true || service["ipFamilyPolicy"] != "PreferDualStack" {
		t.Fatalf("IPv6-enabled service values = %#v", service)
	}

	dashboard := getMonitoringDashboardsValues(false, true)

	global := dashboard["global"].(map[string]any)
	if global["disableProxyIPv6"] != false {
		t.Fatalf("IPv6-enabled dashboard proxy setting = %#v", global)
	}
}

func TestGrafanaAdminPasswordIsOptional(t *testing.T) {
	values := getKubePrometheusStackVals(false, false, false, "", "")
	if _, exists := values["grafana"].(map[string]any)["adminPassword"]; exists {
		t.Fatal("empty Grafana password must preserve kube-prometheus-stack chart default")
	}

	values = getKubePrometheusStackVals(false, false, false, "custom-secret-value", "")
	if got := values["grafana"].(map[string]any)["adminPassword"]; got != "custom-secret-value" {
		t.Fatalf("grafana.adminPassword = %v, want configured value", got)
	}
}

func TestGrafanaPasswordCommandReadsAndDecodesSecret(t *testing.T) {
	command := kubePrometheusStackGrafanaPasswordCommand()
	for _, part := range []string{"kubectl", "kube-prometheus-stack-grafana", "admin-password", "openssl base64 -d"} {
		if !strings.Contains(command, part) {
			t.Errorf("Grafana password command %q missing %q", command, part)
		}
	}
}

func TestCertManagerOCISupportMatchesDocumentedVersions(t *testing.T) {
	for _, tc := range []struct {
		version      string
		oci, crdsVal bool
	}{
		{"1.8.0", false, false},
		{"1.12.17", true, false},
		{"1.15.0", true, true},
		{"1.21.0", true, true},
		{"invalid", false, false},
	} {
		if got := certManagerSupportsOCI(tc.version); got != tc.oci {
			t.Errorf("certManagerSupportsOCI(%q) = %t, want %t", tc.version, got, tc.oci)
		}

		if got := certManagerSupportsCRDsValue(tc.version); got != tc.crdsVal {
			t.Errorf("certManagerSupportsCRDsValue(%q) = %t, want %t", tc.version, got, tc.crdsVal)
		}
	}
}
