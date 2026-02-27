+++
title = "Merge Request Standards"
author = ["readb5@cardiff.ac.uk"]
date = "2026-02-26"
+++

## Summary

This writeup defines the standards for merge request approvals in our development workflow. It outlines the requirements
such as number of approvals needed, when to squash a merge request, who can approve merge requests, and the process of
merging an approved merge request.

1 Approval per MR

Only squash small, or fixup commits

Only approve MRs you have not contributed to

Only MR Authors/Significant Contributors can merge approved MRs

ping `@all ready for review` when an MR is ready for review

## Recommendations

### Number of Approvals

Originally, 2 approvals were required for a merge request to be merged. However, due to significant delays in the
approval process, we have decided to reduce the number of required approvals to 1. This change is aimed at speeding up
the development process while still maintaining a level of code review, we have operated with this number of approvals
before and have not encountered any issues with code quality. Especially since we are only a team of 4/5, 2 approvals is
a significant bottleneck.

### When to Squash Commits (and when not to)

Squashing commits is a practice that can help maintain a cleaner commit history. However, it should be used judiciously.
We recommend squashing commits when:

- The merge request contains multiple small commits that are related to the same feature or bug fix.
- The commits include a lot of "fixup" or "WIP" commits that do not add significant value to the commit history.

On the other hand, we recommend not squashing commits when:

- The commits represent distinct logical changes that are important to preserve in the commit history.
- The commits include important context or information that would be lost if they were squashed.

### Who can approve merge requests (If multiple people worked on it)

GitLab allows anyone but the merge request author to approve a merge request. However, in cases where multiple people
have contributed to a merge request, it is important that the approver is someone who did not contribute significantly
to the merge request, and has fresh eyes.

For example, if Ed and Mohammed have both worked on a front-end page together, it would be best if neither of them
approves the merge request.

### Merging an Approved Merge Request

Once a merge request has received the required number of approvals, the author of the merge request is responsible for
merging it into the main branch.

Unfortunately, GitLab does not allow for this to be enforced, so we will rely on the honor system for this. We trust
that our team members will merge their approved merge requests in a timely manner to keep the development process moving
smoothly.

### Flagging Down MRs for Approval

To ensure that merge requests are reviewed and approved in a timely manner, we recommend that team members flag down
merge requests for approval when they are ready for review. This can be done by tagging the relevant team members in the
merge request description or comments, or by using GitLab's built-in notification system/our discord webhook alert
system to notify the team that a merge request is ready for review. Nothing complicated is necessary just

`@all ready for review` or something similar is sufficient. This will help ensure that merge requests are not overlooked
and that the approval process is as efficient as possible.
