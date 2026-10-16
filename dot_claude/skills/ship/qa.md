# QA

QA proves the acceptance criteria hold in the running software, and leaves
evidence I can check in a minute without re-running anything. A `qa-engineer`
that didn't write the code runs the plan from phase 2. The point is
independence: the builder's tests show the code does what the builder thought;
QA shows it does what the issue asked.

## What QA looks like

Match the method to the change; most web changes use more than one.

- **Browser**: drive the dev app through the AC's path with the
  `agent-browser` skill (a Chrome or Playwright tool if that's what's
  available). Screenshot the states that matter, check the console for new
  errors and the network panel for failed requests, try a narrow viewport and
  keyboard navigation when the change is UI. Capture a "before" from the base
  branch only when the dev setup makes that cheap; otherwise describe it.
- **HTTP**: `curl -i` the endpoints with the cases from the spec, plus the error
  and auth-failure cases. Keep the transcripts.
- **Script**: a one-off script that sets up state and asserts on the outcome,
  for logic that's awkward to reach through the UI. It lives in the evidence
  directory, not the repo, unless it would make a good test, in which case it's
  a finding for the builder.
- **Data**: run migrations up (and down, if supported) against the local
  database, and query that the data came out right.
- **Regression**: the existing tests around the change, and the full suite if
  CI won't run it.

Local dev only: no production or staging URLs, no real customer accounts, no
real payment, email or SMS sends. If an AC can't be shown without them, it's a
manual step for me.

## Running it

Spawn a `qa-engineer` with:

- the spec's ACs and the QA plan
- how to run the app and the check commands from phase 0, and anything the
  setup taught you
- the evidence directory: `$(git rev-parse --git-dir)/ship/qa/round-<n>/`
- the ground rules: it doesn't edit the repo, doesn't commit, and doesn't
  post anything

It comes back with a report: per AC `PASS`, `FAIL` or `BLOCKED`, the evidence
files, for each failure the steps, the expected and actual results, and its
best guess at where the bug is, and any other findings: a failed regression
check, a new console error, a broken neighbor it noticed on the way.

Record each failure and finding in the ledger's QA results as `open`. A
finding is addressed once it's `fixed` (QA re-ran it and it passes) or
`declined` with a reason (out of scope, already broken on the base branch).

- **FAIL** or a finding worth fixing: send it to the builder, run the gate,
  commit, give the fix a mini review (see [review-board.md](review-board.md)),
  then continue the same QA agent with `SendMessage` (or a fresh one, with this
  round's report, if it's gone) to re-run what failed and spot-check its
  neighbors. Three failures on the same AC is an escalation.
- **BLOCKED** (it can't get the app running, or the data the AC needs doesn't
  exist locally): try once yourself to fix the setup. If that doesn't work, the
  AC becomes a manual QA step in the PR, with exact instructions, and the retro
  records what was missing. Building a project "verify" skill for this repo is
  the long-term fix; suggest it to me.

## Posting the evidence

Once every AC is `PASS` or explicitly manual, post the evidence to the PR as one
comment. `gh pr comment --attach` uploads images and rewrites references to them
in the body. Run it from the evidence directory so relative references
resolve:

```
cd "$(git rev-parse --git-dir)/ship/qa/round-<n>"
gh pr comment <pr> --body-file qa-comment.md \
  --attach './ac1-picker.png#Picker without unpublished locales' \
  --attach './ac2-mobile.png#Narrow viewport'
```

`qa-comment.md` starts with `🤖 QA`, then a table of ACs with result and method,
the screenshots placed under the AC they prove (`![alt](./ac1-picker.png)`),
and the transcripts that matter in collapsed `<details>` blocks. Keep it
scannable: I want to verify it at a glance, not read a log. Never attach
anything that shows secrets, tokens or personal data; crop or redact first.
