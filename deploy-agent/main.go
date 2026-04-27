package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	secret := os.Getenv("DEPLOY_AGENT_SECRET")
	port := os.Getenv("DEPLOY_AGENT_PORT")
	repoPath := os.Getenv("DEPLOY_REPO_PATH")
	containerName := os.Getenv("DEPLOY_CONTAINER_NAME")
	webhookURL := os.Getenv("PORTAINER_WEBHOOK_URL")

	if port == "" {
		port = "7777"
	}

	h := &deployHandler{
		secret:      secret,
		runPipeline: buildPipeline(repoPath, containerName, webhookURL),
	}

	mux := http.NewServeMux()
	mux.Handle("POST /deploy", h)

	log.Printf("deploy-agent listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("deploy-agent: %v", err)
	}
}
