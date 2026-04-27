package main

import (
	"bytes"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

func gitShortHash(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func buildPipeline(repoPath, portainerWebhookURL string) func() pipelineResult {
	return func() pipelineResult {
		fromHash := gitShortHash(repoPath)

		var out bytes.Buffer

		pull := exec.Command("git", "-C", repoPath, "pull")
		pull.Stdout = &out
		pull.Stderr = &out
		if err := pull.Run(); err != nil {
			return pipelineResult{Tail: tailLines(out.String(), 20)}
		}

		if strings.Contains(out.String(), "Already up to date.") {
			return pipelineResult{OK: true, AlreadyUpToDate: true, FromHash: fromHash, ToHash: fromHash}
		}

		toHash := gitShortHash(repoPath)
		out.Reset()

		build := exec.Command("docker", "build", "-t", "ledgersyncserver:latest", repoPath)
		build.Stdout = &out
		build.Stderr = &out
		if err := build.Run(); err != nil {
			return pipelineResult{FromHash: fromHash, ToHash: toHash, Tail: tailLines(out.String(), 20)}
		}

		if portainerWebhookURL != "" {
			if resp, err := http.Post(portainerWebhookURL, "", nil); err != nil {
				log.Printf("deploy-agent: portainer webhook: %v", err)
			} else {
				resp.Body.Close()
			}
		}
		return pipelineResult{OK: true, FromHash: fromHash, ToHash: toHash}
	}
}
