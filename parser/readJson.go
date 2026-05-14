package parser

import (
	"encoding/json"
	"lb4a/types"
	"os"
)

func ReadJsonFile(filename string) (types.Config, error) {
	var parsedConfig types.Config

	file, err := os.ReadFile(filename)
	if err != nil {
		return parsedConfig, err // Pass the error back up
	}

	err = json.Unmarshal(file, &parsedConfig)
	if err != nil {
		return parsedConfig, err // Pass the error back up
	}

	return parsedConfig, nil // Success! Return the data.
}
