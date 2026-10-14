# Aftercare

The PR is up, and now it lives in the world: CI runs, review bots and colleagues
comment, I leave notes, and the base branch moves. Each aftercare tick finds
what's new since the last one and handles it. Most ticks find nothing; when
that's the case, end the turn with a single line and don't notify anyone.

## Each tick

Read the ledger first. Its `last-seen` is where the previous tick stopped.

1. **State.** `gh pr view <pr> --json state,isDraft,mergeable,headRefOid,baseRefName`.
   If the PR is merged or closed: delete the cron job (`CronDelete` with the
   ledger's id), set the phase to `done`, write a last log line, and stop.
2. **Sync.** `git fetch origin`. If the remote branch has commits that aren't
   local, someone (probably me) pushed: `git pull --rebase`, and if that
   conflicts, abort and notify me. Never push over commits you haven't pulled.
3. **CI.** `gh pr checks <pr>`. For each failing check, read the failed log
   (`gh run view <run-id> --log-failed`) and decide:
   - caused by this branch: fix it (builder or yourself if it's trivial), gate,
     commit, push
   - flaky (passes locally, unrelated to the diff, a timeout or a network
     blip): re-run it once with `gh run rerun <run-id> --failed`; if it fails
     again, it's not flaky
   - broken on the base branch too: leave it, note it in the ledger and the
     hand-off, don't fix it here
4. **Conflicts.** If `mergeable` is `CONFLICTING`: on a draft, rebase onto
   `origin/<base>`, resolve, gate, and `git push --force-with-lease`. Once I've
   marked it ready, don't rewrite history: notify me instead.
5. **Comments.** Collect everything newer than `last-seen`:

   ```
   gh api graphql -F owner=<owner> -F repo=<name> -F pr=<n> -f query='
     query($owner: String!, $repo: String!, $pr: Int!) {
       repository(owner: $owner, name: $repo) {
         pullRequest(number: $pr) {
           reviewThreads(first: 100) { nodes {
             id isResolved isOutdated path line
             comments(first: 50) { nodes { id databaseId author { login } body createdAt url } }
           } }
           reviews(last: 50) { nodes { id author { login } state body submittedAt } }
           comments(last: 100) { nodes { id author { login } body createdAt url } }
         }
       }
     }'
   ```

   Skip anything starting with `🤖`; that's yours. Skip resolved threads unless
   someone replied after they were resolved.

## Handling a comment

Who wrote it decides how much latitude you have. `gh api user --jq .login` is me.

- **Me** (my login, no `🤖`): an instruction. Do it: questions get answers,
  change requests get verified and then fixed or pushed back on with
  `file:line`, direct instructions get done.
- **Review bots** (Copilot, CodeRabbit, Bugbot, Sentry and the like): a review
  finding of unknown quality. Triage it: restate the mechanism (this state,
  this path, this wrong result), check that it's reachable here given the
  callers, guards and types, and that it matters, then fix or decline. Bots
  are often right about the bug and wrong about the fix.
- **Other people**: fix clear, verified change requests and reply with what
  changed. A question, a design objection, or anything you'd push back on, you
  don't answer in my name: write your proposed reply in the ledger under
  Escalations and notify me with the comment's URL.

Batch the tick's fixes: builder (continued via `SendMessage` if it's still alive,
otherwise a fresh `builder` with the spec and the comments), gate, a mini review
when the fix is more than a line or two (see [review-board.md](review-board.md)),
commit, push. Then reply on each thread you handled, starting with `🤖`:

```
gh api repos/<owner>/<name>/pulls/<n>/comments/<databaseId>/replies \
  -f body='🤖 Fixed in <sha>: <one line>'
```

Resolve the threads that are fixed, but only mine and the bots':

```
gh api graphql -F id=<thread id> -f query='
  mutation($id: ID!) { resolveReviewThread(input: {threadId: $id}) { thread { isResolved } } }'
```

Other people resolve their own threads. Top-level PR comments and review bodies
get one `🤖` reply comment per tick that covers them, not one each.

If a fix changes what the PR does, update the PR body and re-run the affected QA
items, posting a new evidence comment for them.

## Closing the tick

Set `last-seen` to the newest comment you handled, add a log line, and notify me
only if you escalated something or pushed a change after I'd marked the PR
ready. A tick that only fixed bot nits and CI on a draft doesn't need me.
