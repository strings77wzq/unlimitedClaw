## 1. Root Cause Reproduction and Contract Alignment

- [x] 1.1 Reproduce the CI `go vet` failure context locally (or by replaying failing SHA metadata) and document exact unresolved symbols/paths.
- [x] 1.2 Ensure `core/providers/types.go` contains canonical health contract definitions (`HealthStatus`, `HealthChecker`) required by gateway and provider adapters.
- [x] 1.3 Update gateway/provider references to consume canonical provider contract types without duplicate local contracts.

## 2. Regression Hardening

- [x] 2.1 Add or adjust compile-time assertions/tests in provider adapters to enforce `providers.HealthChecker` conformance.
- [x] 2.2 Add/adjust gateway-side regression tests that exercise health checker typing/wiring paths.
- [x] 2.3 Validate static contract integrity with `go vet ./...` and ensure no unresolved provider symbols remain.

## 3. Verification and CI Closure

- [x] 3.1 Run full local verification set (`go test ./...`, `CGO_ENABLED=0 go build ...`) after contract fixes.
- [ ] 3.2 Commit and push the fix branch to trigger GitHub Actions.
- [ ] 3.3 Confirm the previously failing CI path passes and capture the run URL/result in change notes.
