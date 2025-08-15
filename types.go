package main

// API base URL for Coralogix EU2 region
const ApiBaseUrl = "https://api.eu2.coralogix.com/mgmt/openapi"

// WebhookResponse represents the response from webhook creation API
type WebhookResponse struct {
	ID string `json:"id"`
}

// WebhookDetailsResponse represents the response from webhook details API
type WebhookDetailsResponse struct {
	Webhook struct {
		ExternalID int `json:"externalId"`
	} `json:"webhook"`
}
