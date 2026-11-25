package dart

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeDart(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "dart.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("failed to write dart: %v", err)
	}

	return path
}

const minimalDart = `tofu_main_directory: ./tofu/main/harvester
chart_variables:
  rancher_version: 2.9.1
`

func TestParseDefaultsPodStatusTimeout(t *testing.T) {
	d, err := Parse(writeDart(t, minimalDart))
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}

	if d.PodStatusTimeout != DefaultPodStatusTimeout {
		t.Fatalf("expected the default pod status timeout, got %s", d.PodStatusTimeout)
	}
}

func TestParseHonoursPodStatusTimeout(t *testing.T) {
	d, err := Parse(writeDart(t, minimalDart+"pod_status_timeout: 15m\n"))
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}

	if d.PodStatusTimeout != 15*time.Minute {
		t.Fatalf("expected 15m, got %s", d.PodStatusTimeout)
	}
}

func TestParseRejectsNonPositivePodStatusTimeout(t *testing.T) {
	// Values must be duration strings; a bare integer is rejected by the YAML
	// decoder rather than silently reinterpreted.
	for _, value := range []string{"0s", "-30s"} {
		d, err := Parse(writeDart(t, minimalDart+"pod_status_timeout: "+value+"\n"))
		if err != nil {
			t.Fatalf("Parse(%s) returned an error: %v", value, err)
		}

		if d.PodStatusTimeout != DefaultPodStatusTimeout {
			t.Fatalf("expected %s to fall back to the default, got %s", value, d.PodStatusTimeout)
		}
	}
}
