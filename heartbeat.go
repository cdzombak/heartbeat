package heartbeat

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Config is used to create a Heartbeat client
type Config struct {
	// HeartbeatInterval is the interval at which heartbeats are sent. Required.
	HeartbeatInterval time.Duration
	// LivenessThreshold is the maximum time between Alive() calls before heartbeats will be stopped. Required.
	LivenessThreshold time.Duration
	// HeartbeatURL is the URL to GET to send a heartbeat. Required.
	// Redirects will be followed, but the final request must receive an HTTP 2xx response.
	HeartbeatURL string
	// HTTPTimeout is an optional timeout for the heartbeat HTTP requests.
	// If not set, a default timeout of max(HeartbeatInterval - 1 second, 1 second) applies.
	// If set, it must be less than HeartbeatInterval.
	HTTPTimeout time.Duration
	// OnError, if not nil, will be called when an error is encountered while sending a heartbeat. Optional.
	OnError func(error)
}

// NewHeartbeat creates a new Heartbeat client.
// Errors are returned only if the given Config is invalid.
func NewHeartbeat(cfg *Config) (Heartbeat, error) {
	if cfg.LivenessThreshold <= 0.0 {
		return nil, errors.New("liveness threshold must be positive")
	}
	if cfg.HeartbeatInterval <= 0.0 {
		return nil, errors.New("heartbeat interval must be positive")
	}
	if cfg.HTTPTimeout != 0 && cfg.HTTPTimeout >= cfg.HeartbeatInterval {
		return nil, errors.New("timeout must be less than heartbeat interval")
	}
	if cfg.HeartbeatURL == "" {
		return nil, errors.New("heartbeat URL must be set")
	}
	}

	timeout := cfg.HTTPTimeout
	if timeout == 0 {
		timeout = cfg.HeartbeatInterval - time.Second
		if timeout < time.Second {
			timeout = time.Second
		}
	}

	return &heartbeat{
		livenessThreshold: cfg.LivenessThreshold,
		heartbeatInterval: cfg.HeartbeatInterval,
		heartbeatURL:      cfg.HeartbeatURL,
		onError:           cfg.OnError,
		client:            &http.Client{Timeout: timeout},
	}, nil
}

// Heartbeat sends heartbeats to a remote server every HeartbeatInterval,
// as long as Alive has been called in the last LivenessThreshold.
type Heartbeat interface {
	Start()
	Alive(at time.Time)
}

type heartbeat struct {
	heartbeatInterval time.Duration
	livenessThreshold time.Duration
	heartbeatURL      string
	lastAlive         time.Time
	client            *http.Client
	onError           func(error)
	started           bool
	mu                sync.Mutex
}

// Start starts sending heartbeats.
func (h *heartbeat) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		return
	}

	h.started = true
	h.runLocked()
}

// Alive indicates that whatever this heartbeat monitors was alive and functioning
// at the given time.
func (h *heartbeat) Alive(at time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.lastAlive.Before(at) {
		h.lastAlive = at
	}
}

func (h *heartbeat) okUnlocked() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	return time.Since(h.lastAlive) < h.livenessThreshold
}

func (h *heartbeat) runLocked() {
	ticker := time.NewTicker(h.heartbeatInterval)
	go func() {
		for range ticker.C {
			if !h.okUnlocked() {
				continue
			}
			resp, err := h.client.Get(h.heartbeatURL)
			if err != nil {
				err = fmt.Errorf("heartbeat to '%s' failed: %v", h.heartbeatURL, err)
			} else if resp.StatusCode < 200 || resp.StatusCode > 299 {
				err = fmt.Errorf("heartbeat to '%s' failed: %s", h.heartbeatURL, resp.Status)
			}
			if err != nil {
				if h.onError != nil {
					go h.onError(err)
				}
				continue
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}

			var ukRespBody uptimeKumaPushResp
			if err = json.Unmarshal(bodyBytes, &ukRespBody); err == nil && !ukRespBody.OK {
				err = fmt.Errorf("heartbeat to '%s' failed: %s", h.heartbeatURL, ukRespBody.Msg)
			} else {
				err = nil
			}

			if err != nil && h.onError != nil {
				go h.onError(err)
			}
		}
	}()
}

type uptimeKumaPushResp struct {
	OK  bool   `json:"ok"`
	Msg string `json:"msg"`
}
