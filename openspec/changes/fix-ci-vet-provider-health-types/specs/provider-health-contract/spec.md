## ADDED Requirements

### Requirement: Provider Health Contract MUST be Defined in Core Providers
The system SHALL define provider health-check contract types in `core/providers` so all adapters and gateway-facing health integrations use a single canonical definition.

#### Scenario: Health contract symbols exist for shared imports
- **WHEN** gateway and provider packages import `core/providers`
- **THEN** they can resolve `HealthStatus` and `HealthChecker` symbols without local duplicate type definitions

### Requirement: Provider Adapters MUST Implement Health Contract Consistently
Provider adapters that expose health checks SHALL implement `providers.HealthChecker` and return `providers.HealthStatus` for successful and failed health probes.

#### Scenario: Adapter compile-time contract assertion passes
- **WHEN** a provider adapter declares health-check support
- **THEN** a compile-time assertion against `providers.HealthChecker` succeeds

#### Scenario: Adapter returns unhealthy status on transport error
- **WHEN** the adapter health probe cannot reach the provider endpoint
- **THEN** it returns a `HealthStatus` with status `unhealthy`, latency, checked timestamp, and error message

### Requirement: CI MUST Detect Provider Contract Drift Before Merge
The project SHALL fail CI before merge when provider health contract symbols are missing or inconsistent across core, gateway, and adapter packages.

#### Scenario: Contract drift triggers vet/compile failure
- **WHEN** a change removes or renames shared provider health symbols used by gateway or adapters
- **THEN** CI `go vet` or compile validation fails and blocks the workflow
