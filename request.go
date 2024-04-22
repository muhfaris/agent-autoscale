package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Container struct {
	ServiceID   string       `json:"service_id"`
	ServiceName string       `json:"service_name"`
	Current     CurrentStats `json:"current"`
	Based       BasedStats   `json:"based"`
}

type BasedStats struct {
	CPUPercentage    float64 `json:"cpu_percentage"`
	MemoryPercentage float64 `json:"memory_percentage"`
	Min              int64   `json:"min"`
	Max              int64   `json:"max"`
}

type CurrentStats struct {
	CPUPercentage    float64 `json:"cpu_percentage"`
	MemoryPercentage float64 `json:"memory_percentage"`
	Replicas         int     `json:"replicas"`
}

func sendStats(container Container) error {
	var basedurl = "http://0.0.0.0:2441/api/stats"

	// Convert the Container struct to JSON
	jsonData, err := json.Marshal(container)
	if err != nil {
		return err
	}

	// Create a new HTTP client
	client := &http.Client{}

	// Create a new POST request
	req, err := http.NewRequest(http.MethodPost, basedurl, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// Set the content type header to application/json
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultiStatus {
		return nil
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return fmt.Errorf("unable to send stats*(%s): %v", resp.Status, string(body))
}
