package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/rancher/dartboard/internal/summarize/countresources"
)

func main() {
	var kubeconfigPath string
	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file")
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cfg := countresources.Config{
		Kubeconfig: kubeconfigPath,
	}

	if err := countresources.Run(ctx, cfg); err != nil {
		log.Fatalf("count-resources: %v", err)
	}
}