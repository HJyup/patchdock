package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/utils"
	"github.com/google/uuid"
)

func main() {
	ctx := context.Background()
	cli, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalf("Failed to establish a docker client: %v", err)
	}
	defer cli.Close()

	jobDefs := []string{
		"./example-projects/errored-time",
		"./example-projects/runtime",
		"./example-projects/python-test",
	}

	var jobs []*docker.Job
	for _, path := range jobDefs {
		id := fmt.Sprintf("%s-%s", filepath.Base(path), uuid.New().String())
		job, err := docker.NewJob(id, path)
		if err != nil {
			log.Fatalf("failed to schedule job: %v", err)
		}
		jobs = append(jobs, job)
	}

	var logChans []<-chan docker.LogLine
	var resChans []<-chan docker.Result

	for _, job := range jobs {
		jobLogs, jobRes := job.Run(ctx, cli)
		logChans = append(logChans, jobLogs)
		resChans = append(resChans, jobRes)
	}

	logs := utils.FanIn(logChans...)
	results := utils.FanIn(resChans...)

	// ANSI colors
	const (
		reset  = "\033[0m"
		red    = "\033[31m"
		green  = "\033[32m"
		yellow = "\033[33m"
		cyan   = "\033[36m"
	)

	for logs != nil || results != nil {
		select {
		case msg, ok := <-logs:
			if !ok {
				logs = nil
				continue
			}
			color := cyan
			if msg.Stream == "stderr" {
				color = yellow
			}
			fmt.Printf("%s[%s: %s]%s %s\n", color, msg.ID, msg.Phase, reset, msg.Text)

		case result, ok := <-results:
			if !ok {
				results = nil
				continue
			}
			if result.Err != nil {
				fmt.Printf("%s[%s] failed: %v%s\n", red, result.ID, result.Err, reset)
			} else if result.ExitCode != 0 {
				fmt.Printf("%s[%s] exited with code %d%s\n", red, result.ID, result.ExitCode, reset)
			} else {
				fmt.Printf("%s[%s] exited with code %d%s\n", green, result.ID, result.ExitCode, reset)
			}
		}
	}
}
