// Tiny static binary for container healthchecks in distroless images.
// Usage: healthcheck [url]
// Defaults to http://localhost:8080/healthz
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	url := "http://localhost:8080/healthz"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: status %d\n", resp.StatusCode)
		os.Exit(1)
	}
}
