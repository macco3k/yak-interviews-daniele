package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/urfave/cli/v3"
)

const apiBaseUrl = "https://api.eu2.coralogix.com/mgmt/openapi"

// webhookResponse represents the response from webhook creation API
type webhookResponse struct {
	ID string `json:"id"`
}

// webhookDetailsResponse represents the response from webhook details API
type webhookDetailsResponse struct {
	Webhook struct {
		ExternalID int `json:"externalId"`
	} `json:"webhook"`
}

// createWebhook creates a new webhook and returns its ID
func createWebhook(apiKey, webhookFile string) (string, error) {
	bodyContent, err := os.ReadFile(webhookFile)
	if err != nil {
		return "", fmt.Errorf("failed to read webhook file '%s': %w", webhookFile, err)
	}

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/v1/outgoing-webhooks", apiBaseUrl),
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
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read webhook creation response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("webhook creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response webhookResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse webhook creation response: %w", err)
	}

	fmt.Printf("Webhook created successfully with ID: %s\n", response.ID)
	return response.ID, nil
}

// getWebhookExternalId retrieves the external ID of a webhook
func getWebhookExternalId(apiKey, webhookId string) (int, error) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/v1/outgoing-webhooks/%s", apiBaseUrl, webhookId),
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
			log.Printf("failed to close webhook details response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read webhook details response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("webhook details retrieval failed with status %d: %s", resp.StatusCode, string(body))
	}

	var webhookDetails webhookDetailsResponse
	if err := json.Unmarshal(body, &webhookDetails); err != nil {
		return 0, fmt.Errorf("failed to parse webhook details response: %w", err)
	}

	externalID := webhookDetails.Webhook.ExternalID
	fmt.Printf("Webhook external (integration) ID: %d. The alert will be linked to this webhook via this value.\n", externalID)

	return externalID, nil
}

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
		fmt.Sprintf("%s/v3/alert-defs", apiBaseUrl),
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
	defer resp.Body.Close()

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

func main() {
	var webhookFile string
	var alertFile string
	var apiKey string

	cmd := &cli.Command{
		Name:  "webhook",
		Usage: "Manage Coralogix webhooks and (optional) alert definitions.",
		Commands: []*cli.Command{
			{
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "webhook-file",
						Usage:       "Path to file containing JSON body of the webhook.",
						Destination: &webhookFile,
						Required:    true,
					},
					&cli.StringFlag{
						Name:        "alert-file",
						Usage:       "Path to file containing JSON body of the alert definition.",
						Destination: &alertFile,
					},
					&cli.StringFlag{
						Name:        "api-key",
						Usage:       "The Coralogix API key to use for authentication.",
						Destination: &apiKey,
					},
				},
				Name:  "create",
				Usage: "Creates a webhook. If both the webhook and alert files are provided, the webhook will be associated with the alert definition.",
				Action: func(ctx context.Context, command *cli.Command) error {
					// Use flag value or fallback to environment variable
					if apiKey == "" {
						apiKey = os.Getenv("CORALOGIX_API_KEY")
					}
					if apiKey == "" {
						return fmt.Errorf("API key is required. Set CORALOGIX_API_KEY environment variable or use --api-key flag")
					}

					// Create webhook
					webhookId, err := createWebhook(apiKey, webhookFile)
					if err != nil {
						return fmt.Errorf("webhook creation failed: %w", err)
					}

					// If an alert file is provided, create the alert and associate the webhook
					if alertFile != "" {
						// Get webhook external ID
						externalID, err := getWebhookExternalId(apiKey, webhookId)
						if err != nil {
							return fmt.Errorf("failed to get webhook external ID: %w", err)
						}

						// Create alert
						_, err = createAlert(apiKey, alertFile, externalID)
						if err != nil {
							return fmt.Errorf("alert creation failed: %w", err)
						}
					}

					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
