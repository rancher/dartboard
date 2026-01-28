package actions

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// BackoffWait is a shared helper for wait.ExponentialBackoff.
func BackoffWait(steps int, cond func() (bool, error)) error {
	return wait.ExponentialBackoff(wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   1.1,
		Jitter:   0.1,
		Steps:    steps,
	}, cond)
}
