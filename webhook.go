package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

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
