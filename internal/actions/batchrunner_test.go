package actions

import (
	"errors"
	"testing"
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
