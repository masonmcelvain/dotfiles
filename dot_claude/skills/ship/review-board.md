# Review board

A board of specialist reviewers looks at the changeset, a skeptic tries to refute
their serious findings, the builder fixes what survives, and the reviewers
re-check, round after round until they converge.

## The changeset

Everything this branch adds, diffed from the merge base so upstream drift stays
out:

```
git fetch -q origin <base>
base=$(git merge-base origin/<base> HEAD)
git diff "$base"                         # what they review
git diff --stat "$base"                  # what you pick lenses from
```

Everything is committed by the time the board runs, so there are no untracked
files to chase.

## Picking lenses

Always run `review-correctness`. Add the others when the diff gives them
something to do:

| Agent | Signals in the diff |
|---|---|
| `review-security` | auth or session code, permission checks, user input reaching SQL, shell, HTML, file paths or URLs; webhooks; crypto; secrets or tokens; CORS, CSP, cookies; new dependencies |
| `review-frontend` | components, templates, CSS, client JS/TS, routing, forms, anything a user sees or tabs through |
| `review-data` | migrations, schema, queries and ORM calls in loops, indexes, caching and invalidation, queues and jobs, anything at scale or on a hot path |
| `review-ops` | CI workflows, Dockerfiles, deploy or infra config, env vars, feature flags, cron, logging, metrics, alerting, build tooling |

Three or four lenses is typical. If you leave one out on a change where it was
borderline, say why in the ledger.

## Round 1

Spawn the lenses in parallel, all in one message. Keep every handle; later rounds
continue the same agents with `SendMessage` so they remember what they said. Each
gets:

- the changeset commands above, with the real base filled in
- the spec: problem, ACs, non-goals, assumptions. Reviewers should know what the
  change is for, and shouldn't flag a non-goal as missing
- its finding prefix: `C` correctness, `S` security, `F` frontend, `D` data, `O`
  ops; findings numbered `<prefix><round>-<n>` (`S1-2`)
- a reminder that a short clean review is a valid review

## Verify before fixing

Reviewers are tuned to find things, and several of them will find some of the
same things. Before anything reaches the builder:

1. **Dedupe.** The same mechanism raised by two lenses becomes one finding,
   under the more specific lens; note the other id.
2. **Refute.** For each **critical** or **improvement** finding, a
   `finding-verifier` tries to disprove it. Batch the findings by lens into one
   verifier each, in parallel. Each verdict is `CONFIRMED`, `REFUTED` or
   `UNCERTAIN`, with `file:line` evidence.
3. **Decide.**
   - `CONFIRMED`: goes to the builder.
   - `REFUTED`: declined, citing the verifier's evidence. The reviewer can
     still contest it.
   - `UNCERTAIN`: fix it if the fix is small and safe; otherwise decline with
     what's unknown, and add it to the QA plan if QA could settle it.
   - **Nitpicks** skip verification: send the cheap, clearly better ones
     along with the fixes, and decline the rest in one line.

Record every finding and its status in the ledger's board table.

## Fix

Send the builder that wrote the code the fix list with `SendMessage`: each
finding's id, location, mechanism and suggested fix. Then run the gate and commit
(`fix: ...` style messages, grouped sensibly; not one commit per finding).

## Re-review

Continue each lens reviewer with one message per round:

```
S1-1 FIXED - <what changed, where>
S1-2 DECLINED - <why, with file:line or the verifier's evidence>
S1-3 FIXED (nitpick) - <one line>
```

Ask it to re-review the current changeset with the same commands:

- confirm each `FIXED` resolves the finding and didn't regress anything,
  reopening it under a new number if not
- accept or contest each `DECLINED`, at most once and with new evidence
- raise new findings only in code that changed since its last look, unless it
  genuinely missed something
- same format, next round number, a verdict at the end

A lens that said `APPROVE` with nothing outstanding drops out of later rounds.
New findings go through verification again.

## Converging

The board is done when one of these holds:

- every lens says `APPROVE`, or a round produces nothing but nitpicks
- everything still open is a stalemate: declined, contested, declined again on
  the same facts. Freeze it; no third exchange
- four rounds have run

Stalemates and anything left open after round four go in the `🤖 Review board`
PR comment with both positions, and in the notification to me.

## Mini review

After the board, smaller changes (a QA fix, an aftercare fix) get a mini review
instead of the full board: one `review-correctness` agent, spawned fresh, on just
the new commits (`git diff <sha-before-fix>..HEAD`), plus a lens whose territory
the fix touches. Same verify-then-fix rules, two rounds at most.
