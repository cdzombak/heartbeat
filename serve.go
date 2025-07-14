package heartbeat

import (
	"errors"
	"fmt"
	"net/http"
)

func (h *heartbeat) startHttpServerLocked() {
	if h.serverPort == 0 {
		return
	}

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}

			if h.okUnlocked() {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"ok":false}`))
			}
		})

		if err := http.ListenAndServe(fmt.Sprintf(":%d", h.serverPort), mux); err != nil && !errors.Is(err, http.ErrServerClosed) && h.onError != nil {
			go h.onError(err)
			return
		}
	}()
}
