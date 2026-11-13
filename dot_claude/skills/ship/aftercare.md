# Aftercare

The PR is up, and now it lives in the world: pipelines run, colleagues comment,
I leave notes, and the target branch moves. Each aftercare tick finds what's new
since the last one and gets it handled locally. The go-ahead rule still holds:
you fix and commit here, then ask me before anything is pushed. You never reply
to, resolve or react to comments; the fix speaks, and I do the talking. Most
ticks find nothing; when that's the case, end the turn with a single line and
don't notify anyone. Start the app only when a tick needs it, and
[stop it](SKILL.md#stopping-the-app) before the tick ends.

## Each tick

Read the ledger first. Its `last-seen` is where the previous tick stopped, and
its **Go-ahead** section says whether fixes are already waiting on me.

1. **State.** `bkt pr view <n> --json`: state, draft, source commit, target
   branch. If the PR is merged or declined: delete the cron job (`CronDelete`
   with the ledger's id), set the phase to `done`, write a last log line, and
   stop.
2. **Sync.** `git fetch origin`. If the remote branch has commits that aren't
   local, someone (probably me) pushed: `git pull --rebase`, and if that
   conflicts, abort and notify me. Never push over commits you haven't pulled.
3. **Pipelines.** `bkt pr checks <n>`. For each failing build, find its run
   (`bkt pipeline list`) and read the failed step's log (`bkt pipeline logs
   <build number> --step <uuid>`), then decide:
   - caused by this branch: fix it (builder or yourself if it's trivial), gate,
     commit
   - flaky (passes locally, unrelated to the diff, a timeout or a network
     blip): a re-run is a write to Bitbucket, so ask for it in the go-ahead
     request rather than running it
   - broken on the target branch too: leave it, note it in the ledger and the
     report, don't fix it here
4. **Conflicts.** If `bkt pr mergeability <n>` says it conflicts, rebase onto
   `origin/<target>` locally, resolve, and gate. Pushing that is a force-push,
   so the go-ahead request says so.
5. **Comments.** `bkt pr comments <n> --details --json`. What needs handling:
   - **Unresolved inline threads** whose last comment isn't from me answering
     myself, however old, unless the ledger's Escalations already holds them.
     Thread state lives on Bitbucket, so this catches anything a bad
     `last-seen` would miss.
   - **Top-level comments** created after `last-seen`, or all of them when it's
     empty. They have no resolved state, so this is the only place the
     watermark matters.

## Handling a comment

Who wrote it decides how much latitude you have. `bkt auth status` names me.

- **Me**: an instruction. Do it: change requests get verified and then fixed;
  a question or anything you'd push back on goes in your report with
  `file:line`, since you don't reply on the PR.
- **Bots** (Bitbucket's code insights, security scanners and the like): a
  finding of unknown quality. Triage it: restate the mechanism (this state,
  this path, this wrong result), check that it's reachable here given the
  callers, guards and types, and that it matters, then fix or decline.
- **Other people**: fix clear, verified change requests. A question, a design
  objection, or anything you'd push back on: write your proposed reply in the
  ledger under Escalations for me to post, and include the comment's link.

Batch the tick's fixes: builder (continued via `SendMessage` if it's still alive,
otherwise a fresh `builder` with the spec and the comments), gate, a mini review
when the fix is more than a line or two (see [review-board.md](review-board.md)),
commit, following the commit conventions. If a fix changes what the PR does,
re-run the affected QA items as a new round of [qa.md](qa.md) and update
`pr.md`.

## Asking to push

When a tick leaves unpushed commits, or wants a pipeline re-run, fill the
ledger's **Go-ahead** section and notify me once:

```
~/.local/bin/notify -a claude -n claude-desktop "ship: <key> fixes ready" "<pr url> - <n> commits for <what>"
```

End the turn with the request, as in phase 7: each commit and the comment or
build it answers, whether it's a force-push, whether the PR description should
change (show the new `pr.md` diff), and any proposed replies for me to post.
Later ticks keep watching but don't notify again while the same request is
waiting; new fixes join it.

When I say go, push (`--force-with-lease` if it rewrote), run any re-run I
approved, update the description with `bkt pr edit` if I approved that, record
the sha in `pushed`, clear the Go-ahead section and log it. Anything else I say
is feedback, handled as in [The go-ahead](SKILL.md#the-go-ahead).

## Closing the tick

Set `last-seen` to the creation time of the newest comment you handled, never
the current time: a comment that landed while you worked would fall behind the
watermark unread. If you handled nothing, leave it as it was. Add a log line.

## Settling

Once there's nothing left to do, stop the watch. The PR is settled when all of
these hold at the end of a tick:

- **Pipelines are green**: `bkt pr checks <n>` on the current head has nothing
  failing or pending, apart from builds the ledger records as broken on the
  target branch. A tick right after a push is never settled, since its builds
  are still running.
- **Nothing is waiting**: no unpushed commits, no pending go-ahead, step 5
  finds nothing to handle, and Escalations has nothing still waiting on me.
- **No unaddressed QA findings**: every AC in the ledger's QA results is `PASS`
  or a manual step, and no failure or finding is still `open`.

Then delete the cron job (`CronDelete` with the ledger's id), clear `cron` in
the ledger, set the phase to `settled`, log it, and notify me once:

```
~/.local/bin/notify -a claude -n claude-desktop "ship: <key> settled" "<pr url> - aftercare stopped"
```

End the turn there; the session stays open. New comments after this go
unwatched until `mm ship <pr>` resumes, which runs a tick and schedules the
watch again.
