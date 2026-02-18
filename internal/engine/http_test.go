package engine

import (
	"testing"
	"time"
)

func TestDefaultHTTPClient(t *testing.T) {
	if DefaultHTTPClient == nil {
		t.Fatal("DefaultHTTPClient should not be nil")
	}
	if DefaultHTTPClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", DefaultHTTPClient.Timeout)
	}
}
