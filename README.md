# Coralogix Webhook & Alert Manager

A Go CLI tool for managing Coralogix webhooks and alert definitions via the Coralogix API.

## Overview

This CLI application allows you to:
- Create webhooks in Coralogix
- Create alert definitions
- Automatically link webhooks to alert definitions
- Manage webhook configurations through JSON files

## Prerequisites

- Go 1.24.5 or higher
- A valid Coralogix API key
- Access to the Coralogix EU2 region API

## Usage

To run the CLI application, execute the following command:

```shell
go run webhook.go -- create --api-key "your api key" --webhook-file webhook.json --alert-file alert.json
```

- A valid Coralogix API key is required to run the CLI application. This must be provided via the `CORALOGIX_API_KEY` environment variable, or via the `--api-key` flag. If the flag is not provided, the environment variable will be used.
- The `--webhook-file` and `--alert-file` flags are the paths of the files contain the json definitions of the webhook and alert.
- The `--alert-file` flag is optional. If given, the webhook will be automatically linked to the alert.

## Notes

### Coralogix Documentation and API

While working on the assignment, I used the documentation provided by Coralogix at https://docs.coralogix.com/introduction-latest. That proved to be a little bit confusing at times:

- The documentation is quite terse, and it is sometimes difficult to understand what a specific field refers to or does.
- The relationship between the webhook and alert definitions is not clear from just the documentation. In particular, that the webhook's `externalId` field must be used as the alert's `integrationId` in the `notificationGroup` section.
- The `payload` webhook field is defined as a string, but it is actually a JSON object. This makes it difficult to easily include them in the JSON definitions. A full-fledged object would have been easier to work with.

### Design decisions

I went for a cli application to experiment with Go and its packages. I chose to use the [cli](https://github.com/urfave/cli) library for the command line interface as that seems to be one of the most popular one.
This allowed be to focus on the logic of the assignment, and not on the command line interface implementation details.

I've decided to get the webhook's and alert's definitions from JSON files, as that allows some flexibility and allows the user to easily edit the definitions.
The api key can also be provided either directly via a flag, or an environment variable, whichever is more convenient.

Not being familiar with Go, I relied on packages to manage tricky bits such as JSON parsing from bytes without having to define structs for the JSON schema.
The general structure of the application is pretty simple, and boils down to a simple split between the different functions representing the steps of the command:

- `createWebhook`
- `getWebhookExternalId`
- `createAlert`

I've tried to print the most important information to the console, and to provide useful error messages, especially when the problem is related to the requests to the Coralogix API.