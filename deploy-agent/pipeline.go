package main

import (
	"bytes"
	"log"
	"net/http"
	"os/exec"
)

func buildPipeline(repoPath, portainerWebhookURL string) func() (bool, string) {
	return func() (bool, string) {
		var out bytes.Buffer

		pull := exec.Command("git", "-C", repoPath, "pull")
		pull.Stdout = &out
		pull.Stderr = &out
		if err := pull.Run(); err != nil {
			return false, tailLines(out.String(), 20)
		}

		out.Reset()

		build := exec.Command("docker", "build", "-t", "ledgersyncserver:latest", repoPath)
		build.Stdout = &out
		build.Stderr = &out
		if err := build.Run(); err != nil {
			return false, tailLines(out.String(), 20)
		}

		if portainerWebhookURL != "" {
			if resp, err := http.Post(portainerWebhookURL, "", nil); err != nil {
				log.Printf("deploy-agent: portainer webhook: %v", err)
			} else {
				resp.Body.Close()
			}
		}
		return true, ""
	}
}
