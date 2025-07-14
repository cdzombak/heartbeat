package heartbeat

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

func (h *heartbeat) startHeartbeatLocked() {
	if h.heartbeatURL == "" {
		return
	}

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
