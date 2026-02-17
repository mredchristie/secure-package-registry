+++
title = "Initial UI Wireframe"
author = "Bolajims@cardiff.ac.uk"
reviewer = [""]
date = "2026-02-02"
status = "draft"
+++

## Summary

This RFC defines the initial user interface structure and **data requirements** for the Secure Package Registry. The
accompanying wireframe is **low-fidelity** and intended to illustrate layout and information hierarchy rather than final
visual design.

## Problem

Users need a clear way to:

- Search for packages in the secure registry
- View package verification status
- Request verification for unverified packages

The UI must reflect the system’s role as an **alternative package registry**, rather than a vulnerability scanning or
threat-intelligence platform.

## Data Requirements

See `2026-02-04-core-data-schema.md`

## Input Methods (Initial Version)

The initial UI supports a search-based interaction model consistent with an alternative package registry. User search
for packages by name and version. Upload SBOMs (eg. CycloneDX), dependency files (package.json, go.mod) or manually
selecting dependencies are initially deferred to future interactions. This allows early development using mocked data
while keeping the ui aligned with core registry workflow.

## User Interaction Flow (v1)

1. User searches for a package
2. Registry returns matching packages and versions
3. Verification status is displayed for each result
4. If a package is unverified, the user may request verification
5. Status updates asynchronously once analysis completes

## Recommendation

A low-fidelity Excalidraw mockup accompanies this RFC and focuses on data requirements and interaction flow rather than
visual design.

Link: <https://excalidraw.com/#json=Xti4wAsanONJ5sw3bxgq9,fPjLFjtl79mi6sDINAQJvw>

## Open Questions

- Should verification status be shown as tags, icons, or text labels?
- Is a package detail view required in v1, or are search results sufficient?
- What metadata (if any) should be exposed beyond verification status?

## Future Wireframe

- This is an initial wireframe, that would be ideal for the final product - however this is open to change. This is way
  too much for an MVP so this is one for the future....
  ![FutureWF.svg](/uploads/30bc200178d92a7f9de816411285c165/FutureWF.svg)
  <https://excalidraw.com/#json=lt7atDB1OAOg2QYrU9gCJ,riQ8MUH56FL-IEPVNjzPWA>
