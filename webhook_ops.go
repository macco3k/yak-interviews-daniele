package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// createWebhook creates a new webhook and returns its ID
func createWebhook(apiKey, webhookFile string) (string, error) {
	bodyContent, err := os.ReadFile(webhookFile)
	if err != nil {
		return "", fmt.Errorf("failed to read webhook file '%s': %w", webhookFile, err)
	}

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/v1/outgoing-webhooks", ApiBaseUrl),
		bytes.NewReader(bodyContent),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")
	// Required for HTTP/2 requests. Failing to set this header will result in a 401 error.
	req.Header.Set("x-http2-scheme", "https")

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send webhook creation request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("failed to close webhook creation response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read webhook creation response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("webhook creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response WebhookResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse webhook creation response: %w", err)
	}

	fmt.Printf("Webhook created successfully with ID: %s\n", response.ID)
	return response.ID, nil
}

// getWebhookExternalId retrieves the external ID of a webhook
func getWebhookExternalId(apiKey, webhookId string) (int, error) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/v1/outgoing-webhooks/%s", ApiBaseUrl, webhookId),
		nil,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create webhook details request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-http2-scheme", "https")

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send webhook details request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("failed to close webhook details response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read webhook details response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("webhook details retrieval failed with status %d: %s", resp.StatusCode, string(body))
	}

	var webhookDetails WebhookDetailsResponse
	if err := json.Unmarshal(body, &webhookDetails); err != nil {
		return 0, fmt.Errorf("failed to parse webhook details response: %w", err)
	}

	externalID := webhookDetails.Webhook.ExternalID
	fmt.Printf("Webhook external (integration) ID: %d. The alert will be linked to this webhook via this value.\n", externalID)
	return externalID, nil
}
