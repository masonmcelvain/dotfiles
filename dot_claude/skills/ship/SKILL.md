---
name: ship
description: Take a GitHub issue, a PR, or a task in words all the way to a battle-tested draft PR as chief of staff - spec it, delegate the build, run a review board and QA, then keep the PR's CI and review threads handled until it merges. Use only when asked to ship something, or when a ship aftercare tick fires.
argument-hint: "<issue|PR|task> | aftercare <PR>"
---

# Ship

I've handed you a piece of work and walked away. When I come back I want a draft
PR I can trust: specced, built, reviewed by a board of specialists, QA'd with
evidence, and with every review comment on it dealt with. You are the chief of
staff. You plan, delegate, decide, and keep the record; specialists do the deep
work. Keep your own context for judgment - send code reading at scale to `Explore`
agents and implementation to `builder` agents, and ask every agent for a short
report rather than a transcript.

`$ARGUMENTS` is one of:

- an issue (URL or number), a PR (URL or number), or a task in words
- `aftercare <PR>` - a scheduled tick from phase 8; go straight to
  [aftercare.md](aftercare.md)

`mm ship` usually launched you, in a fresh worktree or Code slot with a branch
already checked out (`ship/<n>` or `ship/<slug>`, or the PR's own branch). You may
also have been started by hand in an ordinary checkout; work out where you are
either way.

## Ground rules

These hold in every phase, for you and for every agent you spawn. Paste the ones
that apply into each agent's prompt; they don't inherit this file.

- **Never** merge, mark a PR ready for review, request reviewers, push to the
  default branch, deploy, or touch production or shared databases, queues or
  third-party accounts. Local dev services only.
- Push only to this work's branch. Force-push only with `--force-with-lease`, and
  only while the PR is a draft.
- Don't weaken the safety net to go green: no skipped or deleted tests, no
  loosened lint rules, no `@ts-ignore`/`any` to quiet the typechecker, no CI
  config edits, unless that is the task.
- Secrets stay where they are. Copying the main checkout's untracked env files
  into a worktree is fine; printing, committing or posting them is not.
- Commits follow my conventions in CLAUDE.md. Only you commit - builders leave
  their changes in the tree - so commit messages stay consistent and parallel
  builders never fight over the index. If GPG signing fails (my agent isn't
  forwarded), commit with `git -c commit.gpgsign=false` and note it in the
  ledger; I'll re-sign.
- Anything you post on GitHub goes out under my name. Every PR comment and
  review reply you write starts with `🤖` so readers, and future you, can tell it
  from mine. Anything of mine without it is an instruction from me.

## Deciding versus asking

You can't ask me something and wait; I'm not there. Make the call, write it under
**Assumptions** in the ledger and the PR body, and keep going. Interrupt me only
for:

- an ambiguity where guessing wrong is expensive to undo: a schema or data
  migration, a public API or URL change, deleting something, a product decision
  the issue genuinely leaves open
- a review stalemate, QA failing three times on the same criterion, or a setup
  you can't get running
- anything the ground rules forbid that the task seems to need

Interrupt with my notifier, which reaches me on whatever machine I'm at:

```
~/.local/bin/notify -a claude -n claude-desktop "ship: <repo>#<n>" "<one line: what you need>"
```

Then keep going on whatever doesn't depend on the answer. Only when nothing
does, write the question in the ledger and the PR, and stop.

## The ledger

Your durable memory is `$(git rev-parse --git-dir)/ship/ledger.md`. It's
per-worktree and untracked. Create it in phase 0 and update it at the end of every
phase and every review or QA round. Compaction or a restart will cost you your
context; the ledger is how you pick up where you left off. If it already exists,
read it first and resume from its `phase`; don't redo finished phases.

```
# Ship ledger: <target>
phase: setup | intake | spec | build | review | qa | handoff | aftercare | done
repo: <owner/name>   branch: <branch>   base: <base>   pr: <url or none>
signing: ok | unsigned since <sha>

## Spec
<problem, acceptance criteria AC1.., non-goals>
## Assumptions
## QA plan
## Work packages
## Review board
| id | lens | severity | where | status | note |
## QA results
| AC | result | evidence |
## Escalations
## Aftercare
last-seen: <createdAt of the newest comment handled; empty means since the PR opened>   cron: <job id>
## Log
<one line per event, newest last>
```

## Phase 0: setup

1. Find the repo, branch and base: `gh repo view --json nameWithOwner,defaultBranchRef`,
   `git branch --show-current`. If you're on the default branch, create
   `ship/<n>` (or `ship/<slug>`) from `origin/<base>` first. Never work on the
   default branch.
2. If the tree has changes you didn't make, stop and notify me; they're mine.
3. Make the checkout runnable. In a fresh worktree (`git worktree list`, first
   entry is the main checkout), copy over the main checkout's untracked env files
   (`.env`, `.env.local`, and the like, the ones `.gitignore` covers) and
   install dependencies with the lockfile's own tool (`npm ci`, `pnpm install
   --frozen-lockfile`, `composer install`, ...). A Code slot is already set up.
4. Note how this repo is checked: lint, typecheck, test, build, and how to run
   the app. Look in this order: a project skill about running or verifying
   (`.claude/skills/`), CLAUDE.md or AGENTS.md, `mise` tasks, `package.json`
   scripts, Makefile, `docker-compose.yml`, CI workflows, README. Record the
   commands in the ledger; every builder and the QA agent gets them.

## Phase 1: intake

Read everything the work comes with, before deciding what it is:

- An issue: `gh issue view <n> --comments`, its linked PRs, and the
  issues and Sentry or Slack links it cites (fetch what you can reach). Check
  `gh pr list --search "<n>" --state all` for earlier attempts; a closed PR's
  review comments are a free list of mistakes to avoid.
- A PR: this is adoption, not a new build. Read its body, diff, comments and
  review threads, and the issue it closes, then write the spec it should have
  had and skip to phase 4's gate. The build is whatever's on the branch.
- A task in words: that's all there is. Fill the gaps from the code.

Send one or more `Explore` agents (in parallel, each with a narrow question) to
map the code the change will touch: entry points, the existing patterns to
follow, the tests that cover it, and the callers that would break.

## Phase 2: spec

Write the spec into the ledger. This is the contract for everything after it, so
make it checkable:

- **Problem** in two or three sentences, in terms of user-visible behavior.
- **Acceptance criteria** `AC1`, `AC2`, ...: each a concrete, observable check.
  "Unpublished locales don't appear in the picker on /Guide/123" beats "fix the
  locale picker".
- **Non-goals**, so neither the builders nor the reviewers wander.
- **Assumptions** you made in place of asking.
- **QA plan**, written now, before any code exists, so it can't quietly bend to
  fit what got built. For each AC: how it will be shown to hold (a browser path
  with screenshots, a curl, a one-off script, a test, a query) and against what
  data. Add the regressions most likely to break nearby. See [qa.md](qa.md).
- **Risk lenses**: which review lenses this change will need (see
  [review-board.md](review-board.md)). The final pick happens on the diff; this
  is a forecast.

## Phase 3: plan

Break the work into packages. One package and one builder is the default and the
right call for most issues. Split only when there are genuinely separable parts
(an API change and the UI that consumes it, a migration and the code that reads
it), and order them so each builds on a tree that already passed the gate.
Parallel builders are allowed only when their file sets don't overlap; they all
share this tree, and none of them commits.

For each package, decide which domain skills the builder should load, by name,
and only ones that exist (`ls ~/.claude/skills .claude/skills`). For UI work:
`web-design-guidelines`, `modern-web-guidance`, `vercel-react-best-practices`
for React. For anything else: whatever project skills cover that area.

## Phase 4: build

Spawn a `builder` per package. Keep each builder's handle; when its code comes
back with review or QA findings, continue the same builder with `SendMessage`
(load it with `ToolSearch` if the schema isn't in your context) so it doesn't
have to relearn the code. Its prompt gets:

- the spec, or the part of it this package owns, with ACs and non-goals
- the package's scope and the Explore findings that bear on it
- the domain skills to load
- the repo's check commands from phase 0
- the ground rules that apply to it

When a builder reports back, look at `git diff` yourself before trusting the
report: does it do what the package said, and only that? Then run **the gate**:
lint, typecheck, and the tests near the change (the full suite if it's quick).
Red goes back to the builder. Green gets committed by you, one commit per
package unless the package is naturally several.

After the first green commit, push and open the draft PR, so CI runs while the
board reviews:

```
git push -u origin HEAD
gh pr create --draft --title "<title>" --body-file <file>
```

Title and body follow my PR conventions in CLAUDE.md: `close <issue url>` up top
when it closes one, a short description, `## QA` that will hold the plan for now.
Add an `## Assumptions` section when there are any, and leave a line saying the
board and QA are still running. You'll rewrite the body in phase 7.

If a branch name is still provisional (`ship/<n>`), rename it to something
descriptive (`git branch -m`) before the first push. After the first push, leave
it alone.

## Phase 5: review board

Follow [review-board.md](review-board.md). In short: pick lenses from the diff,
run the lens reviewers in parallel, have a `finding-verifier` try to refute each
serious finding, send the survivors to the builder, then have the reviewers
re-check, and repeat until the board converges.

The draft PR has been open since phase 4, and review bots comment on drafts.
Before each round, read the PR's comments as in [aftercare.md](aftercare.md) and fold
any bot findings into the round under the closest lens, so they get the same
verification as the board's own.

## Phase 6: QA

Follow [qa.md](qa.md). A `qa-engineer` that didn't write the code runs the phase
2 plan against the running app and brings back evidence. Failures go back to the
builder, through the gate and a mini review of the fix, and QA re-runs what
failed.

## Phase 7: hand-off

1. Make sure the gate is green, the tree is clean, and the branch is pushed.
   Wait for CI (`gh pr checks --watch`) and handle failures as in
   [aftercare.md](aftercare.md).
2. Sweep the PR. Run aftercare's comments step with `last-seen` empty, so it
   covers everything since the PR opened, and handle all of it as aftercare
   would. Your own board and QA reports aren't the only reviews; bots and
   people have been commenting since phase 4. Set `last-seen` per aftercare's
   closing rule.
3. Rewrite the PR body (`gh pr edit --body-file`) to my conventions: what changed
   and why, `## Assumptions`, and `## QA` with what was verified, pointing at the
   evidence comment, plus any manual steps QA couldn't do. Leave the agent
   machinery out of the body; it's for the humans who'll review it.
4. Post one `🤖 Review board` comment: lenses run, rounds, each finding once with
   its outcome (fixed, declined with why), and any stalemates with both sides.
5. Leave it a draft. Notify me with the link and one line on anything I need to
   decide:

   ```
   ~/.local/bin/notify -a claude -n claude-desktop "ship: <repo>#<n> ready" "<pr url> - <one line>"
   ```

6. Append a retro to `~/.claude/ship-retros.md` (create it if missing): date,
   repo, PR, what the board and QA caught that the builder should have, setup
   friction, and the run recipe if you had to work it out. This is how repeated
   mistakes become rules.

## Phase 8: aftercare

Set the ledger's phase to `aftercare`, leaving `last-seen` as the hand-off sweep
set it, then schedule the watch with `CronCreate` (load it with `ToolSearch` if
needed): recurring, every 20 minutes, with the
prompt `Use the ship skill: aftercare <pr url>`. Record the job id in the ledger.
Each tick follows [aftercare.md](aftercare.md) until the PR merges or closes,
which deletes the job. It only fires while this session is running, so tell me
in the hand-off that closing this tab ends aftercare, and that
`mm ship <pr>` restarts it.

## Report

Your last message in the terminal stands on its own for when I come back to this
tab: PR link, what was built, the board's findings and outcomes, QA results per
AC, assumptions, escalations, and what aftercare is watching.
