package actions

import (
	"errors"
	"testing"
	"time"

	"github.com/rancher/dartboard/internal/dart"
	"github.com/rancher/dartboard/internal/tofu"
)

func TestCollectBatchResultsAggregatesFailures(t *testing.T) {
	results := make(chan jobResult, 3)
	results <- jobResult{err: errors.New("first")}

	results <- jobResult{skipped: true}

	results <- jobResult{err: errors.New("second")}

	skipped, err := collectBatchResults(results, 3)
	if skipped != 1 {
		t.Fatalf("expected one skipped job, got %d", skipped)
	}

	if err == nil || !errors.Is(err, errBatchFailed) {
		t.Fatalf("expected aggregated batch failure, got %v", err)
	}
}

func TestNewSequencedBatchRunnerUsesConfiguredPodStatusTimeout(t *testing.T) {
	br := NewSequencedBatchRunner[tofu.Cluster](2, 90*time.Second)
	if br.PodStatusTimeout != 90*time.Second {
		t.Fatalf("expected the configured timeout to be used, got %s", br.PodStatusTimeout)
	}
}

func TestNewSequencedBatchRunnerRejectsNonPositivePodStatusTimeout(t *testing.T) {
	for _, timeout := range []time.Duration{0, -1 * time.Second} {
		br := NewSequencedBatchRunner[tofu.Cluster](1, timeout)
		if br.PodStatusTimeout != dart.DefaultPodStatusTimeout {
			t.Fatalf("expected %s to fall back to the default, got %s", timeout, br.PodStatusTimeout)
		}
	}
}
