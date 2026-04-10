package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/rancher/dartboard/internal/summarize/collectprofiles"
)

func main() {
	var cfg collectprofile.Config
	flag.StringVar(&cfg.App, "a", "rancher", "Application: rancher, cattle-cluster-agent, fleet-controller, or fleet-agent")
	flag.StringVar(&cfg.Profiles, "p", "goroutine,heap,profile", "Profiles to be collected (comma separated)")
	flag.IntVar(&cfg.Duration, "t", 30, "Time of CPU profile collections (seconds)")
	flag.StringVar(&cfg.LogLevel, "l", "debug", "Log level of the Rancher pods: debug or trace")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := collectprofile.Run(ctx, cfg); err != nil {
		log.Fatalf("collect-profile: %v", err)
	}
}