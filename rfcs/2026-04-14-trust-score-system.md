+++
title = "Trust Score System"
author = ["cheongyx@cardiff.ac.uk"]
reviewers = []
creation = "2026-04-14"
depends_on = ["2026-03-07-behavioral-analysis-pipeline.md", "2026-03-20-simplified-reproducible-build-integration.md"]
+++

## Summary

Define how the trust score displayed in the UI is calculated. The trust score is a 0-3 scale representing how
trustworthy a package version is, based on provenance checks and behavioral analysis. It is not a simple counter of how
many individual checks pass.

## Problem

We have multiple independent checks for package versions:

- **Provenance checks** (any of: upstream attestation, OSS rebuild, reproducible build)
- **Behavioral analysis** (sandbox-based detection of malicious behavior)
- **Manual review** (human override)

The UI currently shows a trust score but there is no documented specification for how to calculate it. As we add new
check types (e.g. the `reproducible` tag), the scoring logic has been implemented inconsistently across commits because
the semantics were never written down.

## Recommendation

### Score scale: 0 to 3

The trust score is an integer from 0 to 3. It is **not** the count of individual checks that pass. Instead, it
represents a holistic trust level:

| Score | Meaning                                                                                                    |
| ----- | ---------------------------------------------------------------------------------------------------------- |
| 0     | No checks pass                                                                                             |
| 1     | Has provenance but behavioral analysis failed (red flag, provenance alone does not redeem failed behavior) |
| 2     | Behavioral analysis passed but no provenance                                                               |
| 3     | Full trust                                                                                                 |

### Score calculation

The inputs are:

- `has_provenance`: true if **any** of the following are true: `has_attestation`, `has_oss_rebuild`, `has_reproducible`
- `behavior_passed`: true/false/null (null = not yet run)
- `manually_approved`: true/false/null

Rules, evaluated in order:

1. **Manual approval**: If `manually_approved` is true, score is **3**. Manual review is an override.
2. **Pending behavioral analysis**: If `behavior_passed` is null, score is **null** (displayed as "Pending"). We cannot
   assign a meaningful trust level without behavioral analysis results.
3. **Both pass**: If `behavior_passed` is true AND `has_provenance` is true, score is **3**.
4. **Only behavior passes**: If `behavior_passed` is true AND `has_provenance` is false, score is **2**. Missing
   provenance is common and expected for many packages, so passing behavioral analysis alone still indicates reasonable
   trust.
5. **Only provenance passes**: If `behavior_passed` is false AND `has_provenance` is true, score is **1**. Failed
   behavioral analysis is a significant red flag. Even with provenance, the package exhibited suspicious behavior in
   the sandbox. This is intentionally low to draw attention.
6. **Neither passes**: If `behavior_passed` is false AND `has_provenance` is false, score is **0**.

### Provenance as a group

Attestation, OSS rebuild, and reproducible build all serve the same purpose: establishing that the published package
matches its source code. They are different mechanisms for achieving the same guarantee. For scoring purposes, they are
treated as a single boolean group. Having multiple provenance signals is better than one, but for the trust score, any
one of them is sufficient.

This means adding new provenance check types in the future (e.g. a new rebuild service) does not change the score scale
or calculation. It only adds another way to satisfy the provenance requirement.

### UI display

- The trust score is displayed as `{score}/3` (not as a count of individual checks).
- Individual check results (attestation, OSS rebuild, reproducible, behavioral analysis) are shown separately in the
  detail view as PASS/Missing/Pending indicators so users can see exactly which checks passed.
- The "all checks pass" color class applies when `score === 3`.

### Policy enforcement (reg-proxy)

The provenance policy (`require_provenance`) is satisfied if **any** provenance check passes. The policy check should
use the same grouped logic:

```go
if require_provenance && !has_attestation && !has_oss_rebuild && !has_reproducible {
    violation
}
```

This is an OR condition, not an AND. Any single provenance mechanism is sufficient to satisfy the policy.

## Open Questions

1. Should we weight provenance methods differently in the detail view? For example, a reproducible build we performed
   ourselves is arguably stronger evidence than an upstream attestation we can't independently verify. This doesn't
   affect the trust score but could affect how we present information to users.
2. Should the trust score account for transitive dependencies? Currently it only reflects the individual package
   version.
