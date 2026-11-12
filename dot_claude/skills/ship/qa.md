# QA

QA proves the acceptance criteria hold in the running software, and leaves
evidence I can check in a minute without re-running anything. It all stays
local: the evidence feeds the PR description's QA section and its attachments,
and is never posted as a comment. A `qa-engineer` that didn't write the code
runs the plan from phase 2. The point is independence: the builder's tests show
the code does what the builder thought; QA shows it does what the issue asked.

## What QA looks like

Match the method to the change; most web changes use more than one.

- **Browser**: drive the app through the AC's path with `agent-browser`, which
  runs headless and doesn't take my focus, in a session named after the branch
  (`--session US-1234`). If a bot check like Cloudflare's blocks it, switch to
  my own Chrome with Claude in Chrome (the `claude-in-chrome` skill), in a new
  tab of your own that you close when you're done. No permission needed. Being
  blocked never makes an AC BLOCKED while Chrome is left to try. Screenshot the states that matter, check the
  console for new errors and the network panel for failed requests, try a
  narrow viewport and keyboard navigation when the change is UI. Capture a
  "before" from the base branch or the live site when that's cheap; otherwise
  describe it.
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

Show each AC in the real app, the way a user would reach it: the local dev
server first, pointed at a dev store, then the live site for read-only
comparison. A test harness or script that stands in for the page is weaker
evidence, so use one only after you've actually tried to run the app and it
failed. Say what you ran and the error it gave.

**Production is read-only.** My Chrome is signed in as me, including to
production admin tools, so a wrong click there changes real data. On production
and staging, only look: browse pages, read admin screens, check the console and
network. Never save, publish, delete or import anything, place an order, submit
a form that saves anything, change an account or setting, or send email or
SMS. Mutations are only for local dev and the dev stores. If you can't tell
whether a store or admin is a dev one, treat it as production. If an AC can
only be shown by changing production, it's a manual step for me.

## Running it

Spawn a `qa-engineer` with:

- the spec's ACs and the QA plan
- how to run the app and the check commands from phase 0, and anything the
  setup taught you
- the evidence directory: `$(git rev-parse --git-dir)/ship/qa/round-<n>/`
- the ground rules: it doesn't edit the repo, doesn't commit, doesn't push or
  post anything to Bitbucket or Jira, and treats production as read-only

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
  exist locally): check that the report names the command it ran and the error.
  If it doesn't, the agent didn't really try, so send it back. Then try once
  yourself to fix the setup. If that doesn't work, the
  AC becomes a manual QA step in the PR description, with exact instructions,
  and the retro
  records what was missing. Building a project "verify" skill for this repo is
  the long-term fix; suggest it to me.

## Writing it up

Once every AC is `PASS` or explicitly manual, write
`$(git rev-parse --git-dir)/ship/qa/summary.md` for me and for the PR: a table
of ACs with result and method, the screenshot that proves each one, and the
transcripts that matter. Phase 7 turns it into the description's `## QA`
section, which says what was verified and how in plain steps a reviewer could
repeat, and copies the decisive screenshots into the attachments folder, named
after the AC they prove (`ac1-card-alt-text.png`). Keep it scannable: I want to
verify it at a glance, not read a log. Never keep anything for the PR that shows
secrets, tokens or personal data; crop or redact first.
