package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// createAlert creates an alert and links it to the webhook using the external ID
func createAlert(apiKey, alertFile string, externalID int) (string, error) {
	alertBody, err := os.ReadFile(alertFile)
	if err != nil {
		return "", fmt.Errorf("failed to read alert file '%s': %w", alertFile, err)
	}

	// Use sjson to directly set the integrationId in the first webhook
	updatedAlertBody, err := sjson.SetBytes(alertBody, "notificationGroup.webhooks.0.integration.integrationId", externalID)
	if err != nil {
		return "", fmt.Errorf("failed to update alert definition with integration ID: %w", err)
	}

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/v3/alert-defs", ApiBaseUrl),
		bytes.NewReader(updatedAlertBody))
	if err != nil {
		return "", fmt.Errorf("failed to create alert request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("x-http2-scheme", "https")

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send alert creation request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("failed to close alert creation response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read alert creation response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("alert creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	alertId := gjson.Get(string(body), "alertDef.id")
	fmt.Printf("Alert created successfully with ID: %s\n", alertId.String())
	return alertId.String(), nil
}
