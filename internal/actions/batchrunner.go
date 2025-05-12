package actions

import (
	"fmt"
	"sync"
	"time"

	"github.com/rancher/dartboard/internal/dart"
	"github.com/rancher/dartboard/internal/tofu"
	"github.com/rancher/shepherd/clients/rancher"
	shepherddefaults "github.com/rancher/shepherd/extensions/defaults"
	"github.com/sirupsen/logrus"
)

// Mutex to sync map[string]*ClusterStatus mutations and file writes
var stateMutex sync.Mutex

// stateUpdate is a simple "signaling" struct for the file writer goroutine to persist state
type stateUpdate struct {
	Name      string
	Stage     Stage
	Completed time.Time
}

type jobResult struct {
	skipped bool
	err     error
}

// SequencedBatchRunner contains all the channels and WaitGroups needed
// for processing a batch of Clusters
type SequencedBatchRunner struct {
	// Channel to sequence updates
	seqCh chan struct{}
	// Channel for all write requests
	Updates chan stateUpdate
	// Channel for batch jobs
	Jobs chan tofu.Cluster
	// Channel for each individual job's results/error output
	Results chan jobResult

	// WaitGroups for Job workers and the Updates channel which sequences writes to the ClustarStatus state file
	wgWorkers sync.WaitGroup
	wgWriter  sync.WaitGroup
}

// NewBatchRunner constructs a new runner for one batch.
func NewSequencedBatchRunner(batchSize int) *SequencedBatchRunner {
	br := &SequencedBatchRunner{
		Updates: make(chan stateUpdate, batchSize*3),
		seqCh:   make(chan struct{}, 1),
		Jobs:    make(chan tofu.Cluster, batchSize),
		Results: make(chan jobResult, batchSize),
	}
	// seed the sequencer
	br.seqCh <- struct{}{}
	return br
}

// Run executes the batch: starts the file writer, workers, enqueues jobs, collects results
func (br *SequencedBatchRunner) Run(batch []tofu.Cluster, r *dart.Dart, statuses map[string]*ClusterStatus,
	statePath string, client *rancher.Client, config *rancher.Config) error {
	// Start writer
	br.wgWriter.Add(1)
	go br.writer(statuses, statePath)

	// Spawn workers
	for range maxWorkers {
		br.wgWorkers.Add(1)
		go br.worker(statuses, client, config)
	}

	// Enqueue and close jobs
	for _, c := range batch {
		br.Jobs <- c
	}
	close(br.Jobs)

	// Reset skip count for this batch
	numSkipped := 0
	sleepAfter := false
	// Collect results
	for range batch {
		res := <-br.Results
		if res.err != nil {
			// Clean up in case of error
			br.wgWorkers.Wait()
			close(br.Updates)
			br.wgWriter.Wait()
			close(br.Results)
			return fmt.Errorf("error during batch run: %w", res.err)
		}
		if res.skipped {
			numSkipped++
		}
		// Decide whether to sleep before propagating error
		sleepAfter = numSkipped < len(batch)/2
	}

	// After finishing this batch:
	if sleepAfter {
		// If fewer than half were skipped, sleep briefly
		fmt.Printf("Batch done: %d/%d skipped; sleeping before next batch.\n", numSkipped, len(batch))
		time.Sleep(shepherddefaults.TwoMinuteTimeout)
	} else {
		// Otherwise, go straight into the next batch
		fmt.Printf("Batch done: %d/%d skipped; continuing without sleep.\n", numSkipped, len(batch))
	}

	// Clean up
	br.wgWorkers.Wait()
	close(br.Updates)
	br.wgWriter.Wait()
	close(br.Results)
	return nil
}

// TODO: Make this more generic so we can utilize it across all Cluster Registration scenarios
// writer serializes all state updates and persists immediately.
func (br *SequencedBatchRunner) writer(statuses map[string]*ClusterStatus, statePath string) {
	defer br.wgWriter.Done()
	for u := range br.Updates {
		stateMutex.Lock()
		cs := statuses[u.Name]
		switch u.Stage {
		case StageCreated:
			cs.Created = true
		case StageImported:
			cs.Imported = true
		}
		if err := SaveClusterState(statePath, statuses); err != nil {
			logrus.Errorf("failed to save state for %s:%s: %v", u.Name, u.Stage, err)
		}
		stateMutex.Unlock()
	}
}

// TODO: Make this more generic so we can utilize it across all Cluster Registration scenarios
// worker consumes Jobs, calls importClusterWithRunner, signals Updates and Results.
func (br *SequencedBatchRunner) worker(statuses map[string]*ClusterStatus, client *rancher.Client, config *rancher.Config) {
	defer br.wgWorkers.Done()
	for c := range br.Jobs {
		skipped, err := importClusterWithRunner(br, c, statuses, client, config)
		br.Results <- jobResult{skipped: skipped, err: err}
		if err != nil {
			return
		}
	}
}
