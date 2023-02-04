package config

import (
	"encoding/json"
	"log"
	"os"
)

type Properties map[string]string

func LoadProperties(filename string) (Properties, error) {

	properties := Properties{}

	if len(filename) == 0 {
		return properties, nil
	}

	file, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("couldn't open file with filename %s with error: %v", filename, err)
		return nil, err
	}

	err = json.Unmarshal(file, &properties)
	if err != nil {
		log.Fatalf("error unmarshalling the properties: %v", err)
	}

	return properties, nil
}
