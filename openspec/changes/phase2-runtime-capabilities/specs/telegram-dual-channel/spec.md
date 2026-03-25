## ADDED Requirements

### Requirement: Telegram adapter SHALL support both polling and webhook modes
The system SHALL allow the user to select between polling and webhook modes for the Telegram channel via configuration.

#### Scenario: Polling mode starts successfully
- **WHEN** the user starts the Telegram adapter in polling mode
- **THEN** the adapter uses `getUpdates` long polling to receive messages

#### Scenario: Webhook mode starts successfully
- **WHEN** the user starts the Telegram adapter in webhook mode with a valid webhook URL
- **THEN** the adapter sets the webhook with Telegram and mounts an HTTP handler for incoming updates

### Requirement: Webhook handler SHALL be mountable on the gateway server
The system SHALL expose an HTTP handler from the Telegram adapter that can be mounted on the existing gateway HTTP server.

#### Scenario: Gateway mounts Telegram webhook handler
- **WHEN** the gateway server is started with Telegram webhook mode enabled
- **THEN** incoming Telegram webhook POST requests are received and processed by the Telegram adapter

#### Scenario: Webhook handler verifies secret token
- **WHEN** a webhook request arrives with the correct `X-Telegram-Bot-Api-Secret-Token` header
- **THEN** the request is accepted and processed

#### Scenario: Webhook handler rejects invalid secret token
- **WHEN** a webhook request arrives with an invalid or missing secret token
- **THEN** the request is rejected with HTTP 401

### Requirement: Telegram adapter SHALL be wired via CLI flag
The system SHALL provide a `--telegram` CLI flag that enables the Telegram channel adapter.

#### Scenario: Agent starts with Telegram enabled
- **WHEN** the user runs `golem agent --telegram`
- **THEN** the agent starts with the Telegram adapter active in the configured mode

#### Scenario: Agent starts without Telegram flag
- **WHEN** the user runs `golem agent` without the `--telegram` flag
- **THEN** the agent starts without any Telegram adapter
