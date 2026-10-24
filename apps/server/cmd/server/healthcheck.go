package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// healthcheckCommand makes the program check a running server instead of
// starting one, for container health checks in images without a shell or curl.
const healthcheckCommand = "healthcheck"

// healthcheckTimeout bounds the whole check.
const healthcheckTimeout = 5 * time.Second

// checkHealth asks the server listening on listenAddr, the HTTP_ADDR it was
// started with, for its health over the loopback interface, and fails unless
// it answers 200.
func checkHealth(ctx context.Context, listenAddr string) error {
	_, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return fmt.Errorf("reading the port from %q: %w", listenAddr, err)
	}

	ctx, cancel := context.WithTimeout(ctx, healthcheckTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+net.JoinHostPort("127.0.0.1", port)+"/api/health", nil)
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("checking health: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("checking health: the server answered %s", response.Status)
	}

	return nil
}
