package main

import (
	"encoding/json"
	"fmt"
	"lb4a/connection"
	"lb4a/health"
	"lb4a/logger"
	"lb4a/parser"
	ratelimmiter "lb4a/rateLimmiter"
	"lb4a/reload"
	"lb4a/types"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func initialRead() {
	initialConfig, err := parser.ReadJsonFile("lb4a.json")
	if err != nil {
		// It's safe to Fatal here because the server hasn't started yet
		log.Fatalf("CRITICAL: Cannot boot Gateway: %v\n", err)
	}

	// Set the thread-safe config
	types.SetConfig(initialConfig)

	// Log the beautiful JSON dump using the Getter!
	configBytes, _ := json.MarshalIndent(types.GetConfig(), "", "  ")
	fmt.Println("================ INITIAL CONFIGURATION ================\n" + string(configBytes))

}

func main() {

	// backends := []string{"localhost:3000", "localhost:3001"}
	// Inside main()
	if len(os.Args) > 1 && os.Args[1] == "reload" {
		reload.TriggerReload()
		return
	}

	//save the process ID to an file for reload trigger config
	pid := os.Getpid()
	err := os.WriteFile("gateway.pid", []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		log.Fatalf("CRITICAL: Could not write gateway.pid: %v\n", err)
	}
	defer os.Remove("gateway.pid")

	//read initial setup
	initialRead()

	go func() {
		sigChan := make(chan os.Signal, 1)
		// Listen specifically for the SIGHUP (Hangup) signal
		signal.Notify(sigChan, syscall.SIGHUP)

		for {
			<-sigChan // The goroutine pauses here and waits for the signal
			types.Log.Info("SIGHUP received! Triggering hot-reload...")

			// Call your safe reload function!
			err := reload.ReloadConfig()
			if err != nil {
				types.Log.Error("Hot-reload failed, keeping old config.", "error", err)
			} else {

				configBytes, _ := json.MarshalIndent(types.GetConfig(), "", "  ")
				fmt.Println("================ Reload CONFIGURATION ================\n" + string(configBytes))
			}
		}
	}()

	// boot the health workers
	health.StartCheckHealth(2)

	// http.HandleFunc("/", connection.MannualProxy)
	http.HandleFunc("/", ratelimmiter.RateLimitMiddleware(logger.LoggingMiddleware(connection.MannualProxy)))

	types.Log.Info("Gateway started", slog.String("port", "8080"))

	fmt.Println("Manual Layer 7 API Gateway running on :8080...")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Gateway crashed:", err)
	}

}
