# Initial architecture scaffold

Status: documentation only. No implementation completed.

## Goal
Establish stable product intent, module boundaries, terminology, security constraints, and V1 acceptance tests before agentic implementation.

## Follow-up plan
Start with milestone 1 from `docs/PLANS.md`: host-local Git state inspection with fixture-based tests and a host-native CLI. Do not start with GUI or privileged executor code.

## Open decisions

This initial list is superseded by the [maintainer decisions and remaining gates](0001-architecture-readiness.md#maintainer-decisions-and-remaining-approval-gates): Go/system Git/JSON/optional systemd, narrow privileged helpers, Git-backed topology and explicit locally authorized offline deployment are approved. Remaining technical proposals and tests are in plans [0001](0001-architecture-readiness.md) and [0002](0002-agent-git-core.md); controller/UI framework, caller enrollment and exact firewall scope/recovery still need their implementation reviews.
