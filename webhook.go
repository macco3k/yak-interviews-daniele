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

func main() {
	const webhookFileFlagName = "webhook-file"
	const alertFileFlagName = "alert-file"

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
						Name:        webhookFileFlagName,
						Usage:       "Path to file containing JSON body of the webhook.",
						Destination: &webhookFile,
					},
					&cli.StringFlag{
						Name:        alertFileFlagName,
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
					if apiKey == "" {
						apiKey = os.Getenv("CORALOGIX_API_KEY")
					}
					if apiKey == "" {
						err := fmt.Errorf("CORALOGIX_API_KEY environment variable not set.")
						fmt.Println(err)
						return nil
					}

					if webhookFile == "" {
						panic("webhook-file flag not set.")
						return nil
					}

					// Read the request body from file
					bodyContent, err := os.ReadFile(webhookFile)
					if err != nil {
						panic(fmt.Errorf("failed to read file: %w", err))
					}

					req, err := http.NewRequest("POST",
						fmt.Sprintf("%s/v1/outgoing-webhooks", apiBaseUrl),
						bytes.NewReader(bodyContent),
					)

					if err != nil {
						panic(err)
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
						panic(err)
					}

					if resp.StatusCode >= 400 {
						panic(fmt.Errorf("failed to create webhook: %s", string(bodyContent)))
					}

					defer func(Body io.ReadCloser) {
						err := Body.Close()
						if err != nil {
							panic(err)
						}
					}(resp.Body)

					body, err := io.ReadAll(resp.Body)

					if err != nil {
						panic(err)
					}

					type webhookResponse struct {
						ID string `json:"id"`
					}

					var response webhookResponse
					if err := json.Unmarshal(body, &response); err != nil {
						panic(fmt.Errorf("failed to parse response: %w", err))
					}

					webhookId := response.ID
					fmt.Printf("Webhook created successfully with ID: %s\n", webhookId)

					// If an alert file is provided, create the alert and associate the webhook
					if alertFile != "" {
						// Retrieve the external id for the webhook previously saved
						req, err := http.NewRequest("GET",
							fmt.Sprintf("%s/v1/outgoing-webhooks/%s", apiBaseUrl, webhookId),
							nil,
						)

						if err != nil {
							panic(err)
						}

						req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
						req.Header.Set("Content-Type", "application/json")
						req.Header.Set("x-http2-scheme", "https")

						resp, err := client.Do(req)

						if err != nil {
							panic(err)
						}

						defer func(Body io.ReadCloser) {
							err := Body.Close()
							if err != nil {
								panic(err)
							}
						}(resp.Body)

						body, err := io.ReadAll(resp.Body)
						if err != nil {
							panic(err)
						}

						type webhookDetailsResponse struct {
							Webhook struct {
								ExternalID int `json:"externalId"`
							} `json:"webhook"`
						}

						var webhookDetails webhookDetailsResponse
						if err := json.Unmarshal(body, &webhookDetails); err != nil {
							panic(fmt.Errorf("failed to parse webhook details response: %w", err))
						}

						fmt.Printf("Webhook external (integration) ID: %d. The given alert will be linked to this webhook via this value.\n", webhookDetails.Webhook.ExternalID)
						externalID := webhookDetails.Webhook.ExternalID

						alertBody, err := os.ReadFile(alertFile)
						if err != nil {
							panic(fmt.Errorf("failed to read file: %w", err))
						}

						// Use sjson to directly set the integrationId in the first webhook
						updatedAlertBody, err := sjson.SetBytes(alertBody, "notificationGroup.webhooks.0.integration.integrationId", externalID)
						if err != nil {
							panic(fmt.Errorf("failed to update alert definition: %w", err))
						}

						req, err = http.NewRequest("POST",
							fmt.Sprintf("%s/v3/alert-defs", apiBaseUrl),
							bytes.NewReader(updatedAlertBody))

						if err != nil {
							panic(err)
						}

						req.Header.Set("Content-Type", "application/json")
						req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
						req.Header.Set("x-http2-scheme", "https")

						resp, err = client.Do(req)
						if err != nil {
							panic(err)
						}

						defer func(Body io.ReadCloser) {
							err := Body.Close()
							if err != nil {
								panic(err)
							}
						}(resp.Body)

						body, err = io.ReadAll(resp.Body)
						if err != nil {
							panic(err)
						}

						if resp.StatusCode >= 400 {
							panic(fmt.Errorf("failed to create alert: %s", string(body)))
						}

						alertId := gjson.Get(string(body), "alertDef.id")
						fmt.Printf("Alert created successfully with ID: %s\n", alertId)
					}

					return err
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
