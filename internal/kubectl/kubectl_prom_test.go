package kubectl

import "testing"

func TestPickPrometheusService(t *testing.T) {
	for _, tc := range []struct {
		out  string
		want string
		ok   bool
	}{
		{"service/rancher-monitoring-prometheus\n", "rancher-monitoring-prometheus", true},
		{"service/kube-prometheus-stack-prometheus\n", "kube-prometheus-stack-prometheus", true},
		{"service/kube-prometheus-stack-prometheus\nservice/rancher-monitoring-prometheus\n", "rancher-monitoring-prometheus", true},
		{"", "", false},
	} {
		got, ok := pickPrometheusService(tc.out)
		if got != tc.want || ok != tc.ok {
			t.Errorf("pickPrometheusService(%q) = %q, %v; want %q, %v", tc.out, got, ok, tc.want, tc.ok)
		}
	}
}
