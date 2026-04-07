## Context

The failing workflow run (`CI`, push to `main`, run `23534198300`) failed at the `Vet` step for the macOS matrix job. Check-run annotations show compile-time failures tied to provider health types (`undefined: coreproviders.HealthStatus`) and a related unresolved `Message` symbol, indicating contract drift in provider type definitions versus gateway/provider usages at the commit snapshot. Current CI ordering (`go vet` before `go test`) is correct, but it surfaced a mismatch that was not protected by explicit contract-oriented regression coverage.

Observed CI annotation evidence:
- `internal/gateway/server.go:21` → `undefined: coreproviders.HealthStatus`
- workflow annotation (`.github`) reported unresolved `Message` symbol in the same vet failure group

## Goals / Non-Goals

**Goals:**
- Ensure `core/providers` is the single source of truth for provider health contract types (`HealthStatus`, `HealthChecker`) used by gateway and provider adapters.
- Remove ambiguity by making gateway health interfaces depend directly on stable provider contracts.
- Add regression tests/compile assertions that fail fast when provider contract symbols drift.
- Keep CI deterministic across Linux/macOS by validating vet/test/build on the corrected type graph.

**Non-Goals:**
- No redesign of overall provider architecture or LLMProvider/StreamingProvider signatures.
- No addition of new external dependencies.
- No broad CI workflow redesign beyond what is needed to prevent this contract regression.

## Decisions

1. **Centralize health contract in `core/providers/types.go`**  
   - Decision: keep `HealthStatus` and `HealthChecker` alongside existing provider core types.
   - Rationale: gateway and provider adapters already import `core/providers`; consolidating health types there preserves layering and avoids duplicate/competing definitions.
   - Alternative considered: defining gateway-local health DTOs and conversion logic. Rejected because it introduces duplication and drift risk.

2. **Use compile-time interface assertions in provider adapters**  
   - Decision: keep/add `var _ providers.HealthChecker = (*Provider)(nil)` in adapters implementing health checks.
   - Rationale: catches signature drift during compile/vet before runtime.
   - Alternative considered: runtime checks only. Rejected because CI should fail earlier and more deterministically.

3. **Add regression tests for health contract references**  
   - Decision: introduce/adjust tests around gateway health checker wiring and provider health structures.
   - Rationale: annotations indicate unresolved symbols at compile stage; regression tests ensure symbols remain available and consistently wired.
   - Alternative considered: relying on `go vet` only. Rejected because focused tests provide clearer failure locality and faster diagnosis.

4. **Preserve existing CI command chain, verify with local parity runs**  
   - Decision: keep `go vet ./...` then `go test -race ...` and confirm with local runs prior to push.
   - Rationale: current workflow correctly catches static issues early; the issue is contract consistency, not step ordering.

## Risks / Trade-offs

- **[Risk] Existing in-flight branch changes may mask root-cause commit state** → **Mitigation:** validate against the failing SHA context and ensure final fix compiles in current branch before push.
- **[Risk] Additional health contracts in `core/providers` can expand package surface** → **Mitigation:** keep types minimal and internal-contract-oriented, no public CLI behavior changes.
- **[Risk] CI may still fail on unrelated flaky tests after vet fix** → **Mitigation:** run vet/test/build locally and address only deterministic failures introduced by this change.

## Migration Plan

1. Align `core/providers/types.go` with required health contract symbols.
2. Verify all gateway/provider references compile against the unified contract.
3. Add/adjust targeted regression tests for gateway health checker and provider health interfaces.
4. Run `go vet ./...`, `go test ./...`, and `CGO_ENABLED=0 go build ...` locally.
5. Commit and push fix branch; monitor CI run for green outcome.

Rollback strategy: if regressions appear, revert the contract additions and interface assertions as one atomic change, then re-apply with narrower scope.

## Open Questions

- Should health-check capability be implemented for all providers immediately or remain optional per adapter (current approach: optional interface)?
- Should CI add a dedicated compile-only job for matrix targets to detect symbol drift even faster than vet/test?
