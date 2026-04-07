## Why

The CI run failed at the `go vet ./...` step on `main` because provider health-related types referenced by gateway code were not consistently available in the committed source snapshot, causing compile-time symbol resolution errors. We need a focused change that restores a stable type contract between `core/providers` and gateway-facing health interfaces and makes CI failures easier to diagnose and prevent.

## What Changes

- Define and enforce a stable provider health contract in `core/providers` (`HealthStatus` and `HealthChecker`) that can be referenced safely by gateway and provider adapters.
- Ensure gateway health abstractions consume the provider health contract without depending on ad-hoc local type definitions.
- Add/adjust CI verification coverage so contract drift is detected deterministically before merge (compile + vet/test path consistency).
- Add regression tests for the affected provider/gateway health contract paths.

## Capabilities

### New Capabilities
- `provider-health-contract`: Standardizes provider health check type/interface contracts consumed across provider adapters and gateway health reporting.

### Modified Capabilities
- `agent`: Tighten CI-facing quality requirements so compile-time contract drift in core provider abstractions is caught before release.

## Impact

- Affected code: `core/providers/*`, `internal/gateway/*`, provider adapter packages (`core/providers/openai`, `core/providers/anthropic`), and CI workflow validation commands.
- Affected APIs: internal type/interface contracts for health checks; no intentional user-facing CLI API breaking changes.
- Dependencies/systems: GitHub Actions CI (`go vet`, tests), internal provider/gateway integration boundaries.
