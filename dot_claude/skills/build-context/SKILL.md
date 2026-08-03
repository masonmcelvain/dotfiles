---
name: build-context
description: Use when the user wants context on a PR or code change before reviewing it — what changes for callers, what's irreversible, where the risk sits, and what to ask the author. Does not review the code.
---

# Build Context

Read this PR and the surrounding code. Do not review it. Produce:

1. What behavior changes for a user or caller, in plain language.
2. What in this diff is permanent or hard to reverse — migrations, schema, persisted data, published contracts, external side effects.
3. The five riskiest files and one sentence each on why.
4. What existing behavior this changes that the PR description doesn't mention.
5. The five questions you'd ask the author.
