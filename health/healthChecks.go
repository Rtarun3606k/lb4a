package health

import (
	"fmt"
	"lb4a/types"
	"log/slog"
	"net/http"
	"time"
)

func StartCheckHealth(timeTicker int) {

	config := types.GetConfig()

	//define timmer
	timer := time.NewTicker(time.Duration(timeTicker) * time.Second)

	// init logger
	types.Log.Info("Health ceck has started  : ", slog.Int("timer", timeTicker))

	go func() {
		for range timer.C {
			for _, backends := range config.Routes {
				for _, backend := range backends {
					// Spin up a new goroutine for every single ping.
					// This guarantees one slow server doesn't block the loop.
					go pingbackend(backend, 2000)
				}
			}
		}
	}()

}

func pingbackend(b *types.Backend, timeout int) {
	isNowDead := false
	var errMsg string

	client := http.Client{
		Timeout: time.Duration(timeout) * time.Microsecond,
	}

	healthUrl := b.URL + "/health"
	// 1. Explicitly ping the /health endpoint
	resp, err := client.Get(healthUrl)

	// 2. Safely separate the error handling!
	if err != nil {
		// The network connection completely failed (timeout, refused, etc.)
		isNowDead = true
		errMsg = err.Error()
	} else if resp.StatusCode != http.StatusOK {
		// The connection worked, but Flask returned a bad status code (e.g., 500)
		isNowDead = true
		errMsg = fmt.Sprintf("Bad Status Code: %d", resp.StatusCode)
	}

	// 3. Prevent memory leaks
	if resp != nil {
		resp.Body.Close()
	}

	// 4. Thread-safe state check
	wasDead := b.GetDead() // Use whatever you named your thread-safe getter!

	if isNowDead && !wasDead {
		types.Log.Warn("SERVER OFFLINE - Evicting from pool", "url", healthUrl, "reason", errMsg)
		b.SetDead(true)
	} else if !isNowDead && wasDead {
		types.Log.Info("SERVER Online - Adding to pool", "url", healthUrl)
		b.SetDead(false)
	}
}
