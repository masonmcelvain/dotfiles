---
name: ship
description: Take a Jira issue, a Bitbucket PR, or a task in words to a battle-tested branch as chief of staff - spec it, delegate the build, run a review board and QA, all locally - then wait for my go-ahead before pushing and opening the Bitbucket PR. Use only when asked to ship something, when I give a waiting ship run the go-ahead, or when a ship aftercare tick fires.
argument-hint: "<issue key|PR|task> | aftercare <PR>"
---

# Ship

I've handed you a piece of work and walked away. When I come back I want a
branch I can trust: specced, built, reviewed by a board of specialists, and QA'd
with evidence, all on this machine. Nothing leaves it until I say so. Then you
push and open the PR. You are the chief of staff. You plan, delegate, decide,
and keep the record; specialists do the deep work. Keep your own context for
judgment - send code reading at scale to `Explore` agents and implementation to
`builder` agents, and ask every agent for a short report rather than a
transcript.

`$ARGUMENTS` is one of:

- a Jira issue (key like `US-1234`, or a link), a Bitbucket PR (number or
  link), or a task in words
- `aftercare <PR>` - a scheduled tick from phase 9; go straight to
  [aftercare.md](aftercare.md)

Either of the first two may come with **my note on it** (`mm ship US-1234
"verify it's a problem first"` puts it at the end of your prompt after "My note
on it:"). A note is an instruction from me, and it outranks the issue's own
framing: if it says to verify before building, the spec waits on that answer.
Record it verbatim under **Notes** in the ledger in phase 0, so it outlives
compaction, and follow it in every phase it bears on. If the note needs my input
partway through ("check with me before building"), that's an interruption I've
asked for: notify me there and wait, as at the go-ahead. A resumed run with a
new note adds it to the ledger beside the old ones; it doesn't replace them.

`mm ship` usually launched you, in a fresh worktree with a branch already
checked out: the issue key (`US-1234`), `ship/<slug>` for a task, or the PR's
own branch. You may also have been started by hand in an ordinary checkout;
work out where you are either way.

In a Nordic Naturals repo (origin in the `nordicnaturals` Bitbucket workspace),
load the `nordic` skill in phase 0. Its branch, commit and PR conventions win
over CLAUDE.md's, and every builder and the PR follow them.

## Ground rules

These hold in every phase, for you and for every agent you spawn. Paste the ones
that apply into each agent's prompt; they don't inherit this file.

- **Local until I say go.** Never `git push`, create or edit a PR, or comment
  anywhere on Bitbucket or Jira without my go-ahead for that specific push (see
  [The go-ahead](#the-go-ahead)). Reading Bitbucket and Jira is always fine.
- **Never** merge, decline or approve a PR, mark a draft ready, push to the
  default or a release branch, deploy, change Jira, or change production or
  shared databases, queues or third-party accounts. Reading production, such
  as browsing the live site or an admin in my Chrome, is fine. Mutations only
  on local dev services and dev stores.
- Push only to this work's branch. Rewrite history freely while it's unpushed;
  after the first push, force-push only with `--force-with-lease`, and only as
  part of a go-ahead I gave knowing it rewrites.
- Don't weaken the safety net to go green: no skipped or deleted tests, no
  loosened lint rules, no `@ts-ignore`/`any` to quiet the typechecker, no CI
  config edits, unless that is the task.
- Secrets stay where they are. Copying the main checkout's untracked env files
  into a worktree is fine; printing, committing or attaching them is not.
- Commits follow my conventions (the `nordic` skill's in a Nordic repo,
  CLAUDE.md's otherwise). Only you commit - builders leave their changes in the
  tree - so commit messages stay consistent and parallel builders never fight
  over the index. If GPG signing fails (my agent isn't reachable), commit with
  `git -c commit.gpgsign=false` and note it in the ledger; I'll re-sign before
  I give the go-ahead.

## Deciding versus asking

Until the go-ahead pause, I'm not there. Make the call, write it under
**Assumptions** in the ledger, and keep going. Interrupt me only for:

- an ambiguity where guessing wrong is expensive to undo: a schema or data
  migration, a public API or URL change, deleting something, a product decision
  the issue genuinely leaves open
- a review stalemate, QA failing three times on the same criterion, or a setup
  you can't get running
- anything the ground rules forbid that the task seems to need

Interrupt with my notifier:

```
~/.local/bin/notify -a claude -n claude-desktop "ship: <key or slug>" "<one line: what you need>"
```

Then keep going on whatever doesn't depend on the answer. Only when nothing
does, write the question in the ledger and stop.

## The ledger

Your durable memory is `$(git rev-parse --git-dir)/ship/ledger.md`. It's
per-worktree and untracked. Create it in phase 0 and update it at the end of
every phase and every review or QA round. Compaction or a restart will cost you
your context; the ledger is how you pick up where you left off. If it already
exists, read it first and resume from its `phase`; don't redo finished phases.

```
# Ship ledger: <target>
phase: setup | intake | spec | build | review | qa | awaiting-go-ahead | publish | aftercare | settled | done
repo: <workspace/slug>   branch: <branch>   base: <base>   target: <PR target branch>
issue: <key or none>   pr: <url or none>   pushed: <sha or never>
signing: ok | unsigned since <sha>

## Notes
<my notes from mm ship, verbatim, oldest first>
## Spec
<problem, acceptance criteria AC1.., non-goals>
## Assumptions
## QA plan
## Work packages
## Review board
| id | lens | severity | where | status | note |
## QA results
| AC or finding | result | evidence |
## Escalations
## Go-ahead
<what's waiting: commits to push, PR title, target, open questions>
## Aftercare
last-seen: <created_on of the newest comment handled; empty means since the PR opened>   cron: <job id>
## Log
<one line per event, newest last>
```

## Phase 0: setup

1. Find the repo, branch and base: `git remote get-url origin`,
   `git branch --show-current`, `git symbolic-ref --short refs/remotes/origin/HEAD`.
   If you're on the default branch, create the work's branch (the issue key, or
   `ship/<slug>`) from `origin/<base>` first. Never work on the default branch.
2. If origin is in `nordicnaturals`, load the `nordic` skill now.
3. If the tree has changes you didn't make, stop and notify me; they're mine.
4. Make the checkout runnable. In a fresh worktree (`git worktree list`, first
   entry is the main checkout), copy over the main checkout's untracked local
   config, the files `.gitignore` covers that the app needs to run: env files
   (`.env`, `.env.local`, and the like) and `.shopify/project.json`, at the
   same paths, without overwriting any the worktree already has. Then
   install dependencies with the lockfile's own tool (`yarn install
   --frozen-lockfile`, `npm ci`, `pnpm install --frozen-lockfile`, ...).
5. Note how this repo is checked: lint, typecheck, test, build, and how to run
   the app. Look in this order: a project skill about running or verifying
   (`.claude/skills/`), CLAUDE.md or AGENTS.md, `mise` tasks, `package.json`
   scripts, Makefile, `docker-compose.yml`, `bitbucket-pipelines.yml` (the CI
   steps are the gate CI will hold the PR to), README. Record the commands in
   the ledger; every builder and the QA agent gets them.
6. Run the app now, before any code is written, and load a page in a browser.
   Reading the scripts isn't enough. If it fails, read the error and fix what
   you can: a missing env var the main checkout has, a dependency, a port in
   use, a CLI login that's expired. If it needs something only I can give, such
   as credentials or a login, notify me and keep going with the spec while you
   wait. Record in the ledger whether it runs, and if not, the command you ran
   and the error. Never say the app can't run locally without having tried.

## Phase 1: intake

Read everything the work comes with, before deciding what it is:

- An issue: `acli jira workitem view <key> --fields '*all' --json` and its
  comments (the `jira` skill has the commands), its parent epic and linked
  issues, and the links it cites (fetch what you can reach). Check Bitbucket for
  earlier attempts: `bkt pr list --state all --json` filtered to the key, and
  whether origin already has a branch named after it. A declined PR's comments
  (`bkt pr comments <n>`) are a free list of mistakes to avoid.
- A PR: this is adoption, not a new build. Read its description, diff
  (`bkt pr diff <n>`), comments, and the Jira issue in its title, then write the
  spec it should have had and skip to phase 4's gate. The build is whatever's on
  the branch.
- A task in words: that's all there is. Fill the gaps from the code. In a Nordic
  repo it still needs an issue key before anything is pushed; put that question
  in the go-ahead request rather than interrupting me for it now.

Send one or more `Explore` agents (in parallel, each with a narrow question) to
map the code the change will touch: entry points, the existing patterns to
follow, the tests that cover it, and the callers that would break.

## Phase 2: spec

Write the spec into the ledger. This is the contract for everything after it, so
make it checkable:

- **Problem** in two or three sentences, in terms of user-visible behavior.
- **Acceptance criteria** `AC1`, `AC2`, ...: each a concrete, observable check.
  "The cart stays open and shows the error when an update fails" beats "fix
  the cart". When the Jira issue has its own acceptance criteria, start from
  those.
- **Non-goals**, so neither the builders nor the reviewers wander.
- **Assumptions** you made in place of asking.
- **QA plan**, written now, before any code exists, so it can't quietly bend to
  fit what got built. For each AC: how it will be shown to hold (a browser path
  with screenshots, a curl, a one-off script, a test, a query) and against what
  data. Add the regressions most likely to break nearby. See [qa.md](qa.md).
- **Deploy notes**: anything beyond a plain deploy - environment variables,
  CMS or platform settings, scripts, ordering. Usually none; say so.
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
lint, typecheck, format check, and the tests near the change (the full suite if
it's quick). Red goes back to the builder. Green gets committed by you, one
commit per package unless the package is naturally several.

Commit only. Don't push; there's no PR yet.

## Phase 5: review board

Follow [review-board.md](review-board.md). In short: pick lenses from the diff,
run the lens reviewers in parallel, have a `finding-verifier` try to refute each
serious finding, send the survivors to the builder, then have the reviewers
re-check, and repeat until the board converges. The board's record stays in the
ledger and your report; it's never posted.

## Phase 6: QA

Follow [qa.md](qa.md). A `qa-engineer` that didn't write the code runs the phase
2 plan against the running app and brings back evidence. Failures go back to the
builder, through the gate and a mini review of the fix, and QA re-runs what
failed.

## Phase 7: the go-ahead

Everything is local and done. Get it ready to publish, then ask me.

1. Make sure the gate is green, the tree is clean, and every commit follows the
   conventions (in a Nordic repo, every subject starts with the key).
2. Draft the PR into `$(git rev-parse --git-dir)/ship/pr.md`: the title on the
   first line, a blank line, then the description, following the `nordic`
   skill's layout in a Nordic repo and CLAUDE.md's PR guidelines otherwise. It
   carries everything a reviewer needs, since nothing else gets posted: what
   changed and why, `## Changes` when it does several things, `## QA` with
   what was verified and how (and the screenshots by file name, plus any
   manual steps QA couldn't do), and `## Deploy`. Add `## Assumptions` only for
   ones a reviewer should weigh. Leave the agent machinery out; it's for the
   humans who'll review it. Then run the description through the `humanizer`
   skill in embedded mode, with CLAUDE.md's writing voice examples as the
   sample.
3. Gather the screenshots worth attaching into
   `~/Pictures/screenshots/<key>/` (the branch's slug for a task with no key
   yet), named as `pr.md` names them. Nothing that shows secrets, tokens or
   personal data.
4. Fill the ledger's **Go-ahead** section and set the phase to
   `awaiting-go-ahead`.
5. Notify me and end your turn with the report (below), which closes with
   exactly what the go-ahead will do:

   ```
   ~/.local/bin/notify -a claude -n claude-desktop "ship: <key or slug> ready for go-ahead" "<one line>"
   ```

   > Ready to push `US-1234` (3 commits) to origin and open a PR into `master`
   > titled "US-1234 fix: ...". Say go, or tell me what to change.

   List what needs my answer first: the issue key for a task that has none, the
   target branch if a release branch is plausible, unsigned commits, open
   escalations.

## The go-ahead

Only a reply from me in this session approves a push. Starting the skill again
(`mm ship` on the same target lands here with the phase at
`awaiting-go-ahead`) is not one: re-read the ledger, check the branch still
matches it, repeat the go-ahead request, and wait.

- **Go** ("go", "ship it", "push it", "lgtm, open it", with or without
  details like a target branch or "as a draft"): proceed to phase 8 with what I
  said.
- **Changes**: anything else is feedback. Treat it like a review finding from
  me: fix it through a builder, gate, commit, give it a mini review (see
  [review-board.md](review-board.md)) and re-run the QA it touches. Then update
  `pr.md` and the attachments and ask again. My answers to your questions
  without a go are just answers; ask again.

A go-ahead covers the push and PR it describes, not later ones. After the PR is
open, each further push needs its own go-ahead (see
[aftercare.md](aftercare.md)).

## Phase 8: publish

1. If I gave an issue key for a task, rename the branch to it
   (`git branch -m <key>`), move its screenshots folder to
   `~/Pictures/screenshots/<key>/`, and reword every commit subject that
   doesn't already lead with it; they're unpushed, so rewriting is fine.
   Re-run the gate.

   ```
   git rebase origin/<base> -x 'git commit --amend -m "<key> $(git log -1 --format=%B)"'
   ```
2. `git fetch origin <target>`. If the target moved and the branch no longer
   applies cleanly, rebase it, gate, and tell me; ask again if the rebase
   changed anything beyond a trivial conflict.
3. Push: `git push -u origin HEAD`. Record the sha in the ledger's `pushed`.
4. Open the PR with the `bkt` skill, from the first line and the rest of
   `pr.md`. Add `--draft` only if I asked for one:

   ```
   bkt pr create --title "<line 1 of pr.md>" --target <target> \
     --with-default-reviewers --description "$(tail -n +3 pr.md)"
   ```

   An adopted PR already exists: push, and update its description with
   `bkt pr edit <n> --description ...` only if I said to.
5. Record the PR URL and open it for me (`open "<pr url>"`). Bitbucket's API
   can't attach files, so I'll drag the screenshots in from
   `~/Pictures/screenshots/<key>/`.
6. Append a retro to `~/.claude/ship-retros.md` (create it if missing): date,
   repo, issue, PR, what the board and QA caught that the builder should have,
   what my go-ahead feedback caught that the board and QA didn't, setup
   friction, and the run recipe if you had to work it out. This is how repeated
   mistakes become rules.

## Phase 9: aftercare

Set the ledger's phase to `aftercare`, leaving `last-seen` empty, then schedule
the watch with `CronCreate` (load it with `ToolSearch` if needed): recurring,
every 20 minutes, with the prompt `Use the ship skill: aftercare <pr url>`.
Record the job id in the ledger. Each tick follows [aftercare.md](aftercare.md)
until the PR merges or is declined, or settles (pipelines green, nothing
unaddressed), either of which deletes the job. It only fires while this session
is running, so tell me that closing this tab ends aftercare, and that
`mm ship <pr>` restarts it. Resuming with the phase at `aftercare` or `settled`
means running one tick now and scheduling the job again.

## Report

Your messages at the go-ahead pause and after publishing stand on their own for
when I come back to this tab: what was built, the board's findings and outcomes,
QA results per AC with the evidence paths, assumptions, escalations, and then
either the go-ahead request or the PR link and what aftercare is watching.
