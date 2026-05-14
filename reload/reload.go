package reload

import (
	"encoding/json"
	"fmt"
	"lb4a/parser"
	"lb4a/types"
	"log"
	"os"
	"strconv"
	"syscall"
)

func ReloadConfig() error {
	types.Log.Info("Attempting to hot-reload configuration...")

	// Try to read the file
	newConfig, err := parser.ReadJsonFile("lb4a.json")
	if err != nil {
		// DO NOT FATAL! Just log the error. The server keeps running on the old config!
		types.Log.Error("Failed to parse config file. Reload aborted.", "error", err)
		return err
	}

	// If we made it here, the JSON is perfect. Swap it into the live gateway!
	types.SetConfig(newConfig)

	configBytes, _ := json.MarshalIndent(types.GetConfig(), "", "  ")
	types.Log.Info("Hot-reload successful! New routing rules applied.", "config", string(configBytes))

	return nil
}

func TriggerReload() {
	//  Read the PID file
	pidBytes, err := os.ReadFile("gateway.pid")
	if err != nil {
		log.Fatalf(" Gateway is not running (could not find gateway.pid)")
	}

	// Convert the string back to a number
	pid, _ := strconv.Atoi(string(pidBytes))

	//  Find the running Gateway process
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Fatalf("Could not find Gateway process %d", pid)
	}

	// Send the SIGHUP signal from Go!
	err = process.Signal(syscall.SIGHUP)
	if err != nil {
		log.Fatalf("Failed to send reload signal: %v", err)
	}

	fmt.Println("Reload signal sent successfully to Gateway (PID:", pid, ")")
}
