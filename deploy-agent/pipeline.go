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

func buildPipeline(repoPath, containerName, portainerWebhookURL string) func() pipelineResult {
	return func() pipelineResult {
		fromHash := gitShortHash(repoPath)
		log.Printf("deploy-agent: pipeline start fromHash=%q repo=%s", fromHash, repoPath)

		var out bytes.Buffer

		pull := exec.Command("git", "-C", repoPath, "pull")
		pull.Stdout = &out
		pull.Stderr = &out
		if err := pull.Run(); err != nil {
			log.Printf("deploy-agent: git pull failed: %v", err)
			return pipelineResult{Tail: tailLines(out.String(), 20)}
		}

		if strings.Contains(out.String(), "Already up to date.") {
			log.Printf("deploy-agent: already up to date at %s", fromHash)
			return pipelineResult{OK: true, AlreadyUpToDate: true, FromHash: fromHash, ToHash: fromHash}
		}

		toHash := gitShortHash(repoPath)
		log.Printf("deploy-agent: pulled %s -> %s, building image", fromHash, toHash)
		out.Reset()

		build := exec.Command("docker", "build", "-t", "ledgersyncserver:latest", repoPath)
		build.Stdout = &out
		build.Stderr = &out
		if err := build.Run(); err != nil {
			log.Printf("deploy-agent: docker build failed: %v", err)
			return pipelineResult{FromHash: fromHash, ToHash: toHash, Tail: tailLines(out.String(), 20)}
		}
		log.Printf("deploy-agent: docker build ok")

		if containerName != "" {
			// Remove the old container so the webhook recreates it fresh with the new image.
			// docker stop/rm errors are non-fatal (container may already be stopped/absent).
			if stopOut, err := exec.Command("docker", "stop", containerName).CombinedOutput(); err != nil {
				log.Printf("deploy-agent: docker stop %s: %v: %s", containerName, err, stopOut)
			} else {
				log.Printf("deploy-agent: docker stop %s ok", containerName)
			}
			if rmOut, err := exec.Command("docker", "rm", containerName).CombinedOutput(); err != nil {
				log.Printf("deploy-agent: docker rm %s: %v: %s", containerName, err, rmOut)
			} else {
				log.Printf("deploy-agent: docker rm %s ok", containerName)
			}
		}

		if portainerWebhookURL != "" {
			log.Printf("deploy-agent: calling portainer webhook")
			resp, err := http.Post(portainerWebhookURL, "", nil)
			if err != nil {
				log.Printf("deploy-agent: portainer webhook error: %v", err)
				return pipelineResult{FromHash: fromHash, ToHash: toHash, Tail: "portainer webhook: " + err.Error()}
			}
			resp.Body.Close()
			log.Printf("deploy-agent: portainer webhook returned %s", resp.Status)
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return pipelineResult{FromHash: fromHash, ToHash: toHash, Tail: "portainer webhook returned " + resp.Status}
			}
		}
		log.Printf("deploy-agent: pipeline done ok %s -> %s", fromHash, toHash)
		return pipelineResult{OK: true, FromHash: fromHash, ToHash: toHash}
	}
}
