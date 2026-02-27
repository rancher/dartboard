package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/rancher/dartboard/internal/summarize/exportmetrics"
)

func main() {
	var kubeconfigPath string
	var selector string
	var startStr string
	var endStr string
	var offset int64

	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file")
	flag.StringVar(&selector, "selector", "", "Prometheus selector query")
	flag.StringVar(&startStr, "start", "", "Start time (RFC3339: 2006-01-02T15:04:05Z)")
	flag.StringVar(&endStr, "end", "", "End time (RFC3339: 2006-01-02T15:04:05Z)")
	flag.Int64Var(&offset, "offset", 0, "Offset in seconds")
	flag.Parse()

	if kubeconfigPath == "" {
		if flag.NArg() > 0 {
			kubeconfigPath = flag.Arg(0)
		} else {
			kubeconfigPath = os.Getenv("KUBECONFIG")
		}
	}

	if kubeconfigPath == "" {
		log.Fatal("Error: Kubeconfig not found. Please set KUBECONFIG env var or pass as argument.")
	}

	var fromSeconds, toSeconds int64
	if startStr != "" {
		t, err := time.Parse(exportmetrics.PromTimeFormat, startStr)
		if err != nil {
			log.Fatalf("Invalid start time format: %v", err)
		}
		fromSeconds = t.Unix()
	}
	if endStr != "" {
		t, err := time.Parse(exportmetrics.PromTimeFormat, endStr)
		if err != nil {
			log.Fatalf("Invalid end time format: %v", err)
		}
		toSeconds = t.Unix()
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := exportmetrics.Config{
		Kubeconfig:    kubeconfigPath,
		Selector:      selector,
		FromSeconds:   fromSeconds,
		ToSeconds:     toSeconds,
		OffsetSeconds: offset,
	}

	if err := exportmetrics.Run(ctx, cfg); err != nil {
		log.Fatalf("export-metrics: %v", err)
	}
}
