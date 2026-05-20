package testhelper

import (
	"os"
)

func init() {
	// Fix for testcontainers-go v0.36.0 on Windows:
	// Testcontainers sometimes fails to auto-detect the Docker host on Windows
	// and panics with "rootless Docker is not supported on Windows".
	// Setting DOCKER_HOST explicitly to the default Docker Desktop named pipe
	// prevents this issue if it's not already set.
	if os.Getenv("DOCKER_HOST") == "" {
		os.Setenv("DOCKER_HOST", "npipe:////./pipe/docker_engine")
	}
}
