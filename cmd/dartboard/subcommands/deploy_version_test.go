package subcommands

import (
	"testing"

	"github.com/rancher/dartboard/internal/dart"
)

func TestUseLegacyRancherMonitoring(t *testing.T) {
	for _, tc := range []struct {
		version string
		force   bool
		want    bool
	}{
		{"2.9.1", false, true},
		{"v2.9.1", false, true},
		{"2.14.3", false, true},
		{"2.15.0", false, false},
		{"v2.15.0-alpha1", false, false},
		{"2.16", false, false},
		{"2", false, true},
		{"v2", false, true},
		{"", false, true},
		{"3", false, false},
		{"2.9.1", true, false},
	} {
		r := &dart.Dart{ChartVariables: dart.ChartVariables{RancherVersion: tc.version, ForceKubePrometheusStack: tc.force}}
		if got := useLegacyRancherMonitoring(r); got != tc.want {
			t.Errorf("useLegacyRancherMonitoring(%q, force=%v) = %v, want %v", tc.version, tc.force, got, tc.want)
		}
	}
}
