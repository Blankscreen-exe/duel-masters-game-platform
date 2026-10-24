package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveHealth starts a server answering every request with status, and
// returns the HTTP_ADDR a server listening on its port would have.
func serveHealth(t *testing.T, status int) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/health", r.URL.Path)
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)

	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	require.NoError(t, err)

	return net.JoinHostPort("0.0.0.0", port)
}

func TestCheckHealthPassesForAHealthyServer(t *testing.T) {
	assert.NoError(t, checkHealth(context.Background(), serveHealth(t, http.StatusOK)))
}

func TestCheckHealthFailsForAnUnhealthyServer(t *testing.T) {
	err := checkHealth(context.Background(), serveHealth(t, http.StatusServiceUnavailable))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestCheckHealthFailsWhenNothingListens(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	assert.Error(t, checkHealth(context.Background(), addr))
}

func TestCheckHealthRefusesAnAddressWithoutAPort(t *testing.T) {
	assert.Error(t, checkHealth(context.Background(), "localhost"))
}
