package cmd

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// setIdleTimeout shortens upgradeIdleTimeout for one test.
func setIdleTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	old := upgradeIdleTimeout
	upgradeIdleTimeout = d
	t.Cleanup(func() { upgradeIdleTimeout = old })
}

// waitOrDone sleeps for d, or returns early when the client goes away, so a
// handler never outlives its test.
func waitOrDone(r *http.Request, d time.Duration) bool {
	select {
	case <-time.After(d):
		return true
	case <-r.Context().Done():
		return false
	}
}

func TestDownloadAsset_SlowButSteadySucceeds(t *testing.T) {
	setIdleTimeout(t, 300*time.Millisecond)
	chunk := bytes.Repeat([]byte("x"), 64)
	const chunks = 10
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("response writer cannot flush")
			return
		}
		// Ten chunks 100ms apart: about a second in total, several times the
		// idle timeout, but never idle for that long.
		for i := 0; i < chunks; i++ {
			if _, err := w.Write(chunk); err != nil {
				return
			}
			flusher.Flush()
			if i < chunks-1 && !waitOrDone(r, 100*time.Millisecond) {
				return
			}
		}
	}))
	defer srv.Close()

	start := time.Now()
	data, err := downloadAsset(context.Background(), srv.Client(), srv.URL+"/a.tar.gz", 1<<20)
	require.NoError(t, err)
	require.Equal(t, bytes.Repeat(chunk, chunks), data)
	require.Greater(t, time.Since(start), upgradeIdleTimeout,
		"the download should outlast the idle timeout, or this test proves nothing")
}

func TestDownloadAsset_StallAborts(t *testing.T) {
	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
	}{
		{
			name: "body stops arriving",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte("partial")); err != nil {
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				waitOrDone(r, 10*time.Second)
			},
		},
		{
			name: "headers never arrive",
			handler: func(w http.ResponseWriter, r *http.Request) {
				waitOrDone(r, 10*time.Second)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setIdleTimeout(t, 200*time.Millisecond)
			srv := httptest.NewTLSServer(http.HandlerFunc(tt.handler))
			defer srv.Close()

			start := time.Now()
			_, err := downloadAsset(context.Background(), srv.Client(), srv.URL+"/a.tar.gz", 1<<20)
			require.Error(t, err)
			require.True(t, errors.Is(err, errDownloadStalled), "got %v", err)
			require.Contains(t, err.Error(), "no data received for 200ms")
			require.Less(t, time.Since(start), 5*time.Second, "a stall must abort promptly")
		})
	}
}

func TestDownloadAsset_CallerCancelIsNotAStall(t *testing.T) {
	setIdleTimeout(t, time.Minute)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		waitOrDone(r, 10*time.Second)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := downloadAsset(ctx, srv.Client(), srv.URL+"/a.tar.gz", 1<<20)
	require.Error(t, err)
	require.False(t, errors.Is(err, errDownloadStalled), "a cancelled upgrade is not a stall: %v", err)
}

func TestUpgradeHTTPClient_NoOverallTimeout(t *testing.T) {
	require.Zero(t, upgradeHTTPClient.Timeout,
		"an overall timeout would fail a valid download on a slow link")
	tr, ok := upgradeHTTPClient.Transport.(*http.Transport)
	require.True(t, ok)
	require.Positive(t, tr.ResponseHeaderTimeout)
	require.Positive(t, tr.TLSHandshakeTimeout)
	require.Positive(t, upgradeIdleTimeout)
}
