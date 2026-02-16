package engine

import (
	"net/http"
	"time"
)

// DefaultHTTPClient is the shared HTTP client with a 30-second timeout.
var DefaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}
