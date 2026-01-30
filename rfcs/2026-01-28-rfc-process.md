+++
title = "RFC Guidelines and Process"
author = ["cheongyx@cardiff.ac.uk"]
reviewers = ["ReadB5@cardiff.ac.uk", "ChristieE1@cardiff.ac.uk"]
date = "2026-01-28"
status = "complete"
+++

## Summary

An RFC (Request For Comments) process puts design decisions in writing, gathers feedback, and acts as documentation. The
process should be lax, concise, and decision focused to avoid bureaucracy for our small team. The purpose is align the
team on what we are doing and avoid implementations built on incorrect assumptions.

## Problem

We have 5 developers, working mostly asynchronously with only 1 meeting a week. For a startup-like project where it
isn't even clear _what_ to build, this allows for a lot of misunderstandings and confusion around design and
implementation.

## Options considered

### More in-person meetings

Nobody wants that.

### Increased discussion on issues via chat channels

We have a Discord group used for communication. We could more thoroughly discuss design before implementing while
remaining asynchronous. However, since the process is not formal, it is difficult to ensure discussion was done. With
only a single linear chat, history is easily lost and members not online during discussions get left in the dark.

### Just allow the tech debt

If this was a short term contracted client project, this may be viable. However, we are planning on maintaining this
long term. As such, excessive tech debt is not recommended.

### RFC Processes

There are many different styles of RFC processes. We specifically looked at those used by small but successful startups.
Some write-ups used as reference:

- <https://highimpactengineering.substack.com/p/the-illusion-of-shared-understanding>
- <https://spin.atomicobject.com/remote-team-decisions-rfc/>
- <https://engineering.squarespace.com/blog/2019/the-power-of-yes-if>

A few extracted points from the writings that I found relevant:

- **Auto-approval** is necessary. If a certain amount of time passes and nobody reviews the RFC, take it as passed. This
  helps prevent being blocked by bureaucracy. While this may create a negative incentive structure to produce more RFCs
  as to allow more to automatically pass, this would only occur in financially motivated teams evaluated by management
  on a specific metric. We have no managers nor metrics, and therefore the downsides are unlikely to apply.
- **No hard requirements on RFC content,** only suggestions. Reduce the effort to write trivial RFCs. If the concept and
  implementation is simple, a few lines documenting it is sufficient.
- **You can still write an implementation without an RFC.** If something is very obvious and doesn't require discussion,
  you can always write a brief summary after the fact.

## Recommendation

### Template

You are **not** required to follow this format. This is merely a recommendation.

```markdown
+++
title = "A few words"
authors = ["you@example.com"]
reviewers = ["me@example.com", "rando@example.com"]
tags = ["reproducible_builds", "behavioral_detection"] # Components of the system. Specific tags are to be added in a later RFC on architecture and components
creation = "2026-01-27"
last_update = "2026-01-29"
status = "draft" # draft -> review -> complete -> superseded | declined. Approved is not a valid status as that is implied by presence in the repository.
superseded_by = ["2026-01-29-other-rfc.md"] # Optional
depends_on = ["2026-01-01-discord.md"] # Optional
+++

## Summary

<!-- A couple lines that describe the RFC without going into detail -->

## Problem

<!-- Quick summary about what we're working on -->

## Recommendation

<!-- Main text about what you want to happen -->

## Open Questions

<!-- Anything that isn't resolved, and may need to be resolved in a future RFC. Include dependencies -->

## Change Log

<!-- So that someone who read a draft can quickly figure out what has changed since they last read -->
```

### Auto-approval

If 5 working days pass without review, everyone should be pinged and it must be brought up in the subsequent meeting.
Any member can call for in-person review if auto-approval is imminent. If not blocked after this process, the RFC is
considered automatically approved.

### What needs an RFC and what doesn't

#### Yes RFC

- Complex implementation that has significant trade-offs
- Architectural design (includes things like database schemas, bug fixes that changes interfaces)
- User interaction flow or user interface design. Your mock ups go here.
- APIs that affect other team members. For example, if your component exposes an HTTP API, and a route changes,
  requiring a change in a component maintained by another developer

#### No RFC

- Normal bug fixes
- Implementation details with no significance to anyone else
- Adding new libraries/dependencies
- Refactoring existing code to be more elegant without changing interfaces

### Notification

For the RFC process to be effective, members must be notified and aware of the process. A Discord webhook should post
new RFCs to a dedicated channel. Tagged reviewers should be pinged.

GitLab's webhook feature is limited. It only allows filtering by feature (Merge request, issue, comments), but not
differences between event types within. So every time you pushed to a branch with a open pull request, it treats that
the same as opening a new pull request. As such, we have decided to go with Cloudflare workers to do a pre-processing
step before pushing to Discord. The source code is tracked at `../infra/gitlab-webhooks-handler/`.

### Revision of an existing RFC

Modifying an existing RFC is often a bad idea. We need an accurate historical record of decision making while keeping
previous discussion threads valid for context. If version 1 is already implemented and deployed, this maintains
documentation for that version. The change log section should only show the series of changes _before_ approval.

The original should only be updated after approval for minor clarifications (not material changes), documenting
implementation details discovered during work, adding links to implementation PRs, and fixing typos or ambiguities. Mild
breaking changes are allowed before implementation as long as no other RFC has been made with the assumption of certain
behavior or interfaces.

Instead, RFCs are superseded by new documents. When an RFC is superseded, we can move them to a dedicated superseded
RFCs folder as to not pollute context. The old RFC should clearly reference which RFC it has been superseded by.

Example flow:

1. RFC-001: Use MongoDB → Approved → Partially implemented
2. Discover: MongoDB doesn't meet needs → Write RFC-002
3. RFC-002: Use PostgreSQL → supersedes = ["RFC-001"]
4. Keep MongoDB code running during migration
5. Both RFCs remain valid documentation for their respective implementations

### Approve, decline, and block

For a reviewer, you can decide to approve, decline, or block an RFC. A block must be associated with constructive
comments on specific parts that could be refined. We require 2 approvals or declines to instantly merge or close a PR.

Depending on the complexity and importance of an RFC, 1 approval is sufficient given a 2 working day buffer for other
interested parties to voice their opinions. Still, at least 2 people must be pinged and aware of the process. An RFC
cannot be merged with 1 approval without prior knowledge of 1 more reviewer.

If there are conflicts between reviewers on whether to approve or decline an RFC, a meeting shall be held (or waited
until) with the whole team.

Blocks and approvals can happen at any point in time.

### Competing RFCs

This should never happen with proper communication. Before writing an RFC, intent should be voiced and heard. However,
if two competing RFCs do show up for whatever reason, this should raise a meeting on organizational issues and finally a
vote held after live discussion.

### Dependencies

One RFC might depend on another. For example, the implementation of a worker that fetches from a queue depends on the
specification of the queue's interface of communication.

**These must be properly defined** as part of the markdown metadata if relevant. This allows us to know which RFCs
require updating when another is superseded. If no RFCs currently depend on an implemented RFC, we may allow breaking
changes if we realize an issue after the fact.

### Exceptions

Exceptions will be made during emergencies. We can work out better strategies as work out how well we collaborate.

## Open Questions

1. What happens in unlikely scenarios such as if a reviewer doesn't respond to being tagged?

## Change Log

N/A
