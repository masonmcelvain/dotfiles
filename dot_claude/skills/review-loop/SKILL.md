---
name: review-loop
description: Have a reviewer agent review the local changes, address its comments, and ping-pong until the two of you converge, before a PR goes out to people
argument-hint: "[base branch]"
disable-model-invocation: true
---

# Review loop

The changes are made but nobody else has seen them yet. Before I open a PR, run a
review round-trip with a second agent: it reviews the changeset and sends you its
comments, you fix or push back on each one, it re-reviews, and so on until you
converge. You are the author in this exchange; the reviewer never edits anything.

## The changeset

Same scope as my `hkr` alias: everything this branch added, diffed from the merge
base so upstream drift stays out, plus whatever is uncommitted or untracked.
`$ARGUMENTS` overrides the base branch; otherwise use `origin/HEAD`, falling back to
`main`, then `master`:

```
base=$(git merge-base <base-branch> HEAD)
git diff "$base"                                  # committed + uncommitted
git ls-files --others --exclude-standard          # untracked, read these whole
```

If that comes back empty, say so and stop. There's nothing to review.

## Round 1: the review

Spawn one reviewer with the `code-reviewer` agent type and keep its handle - every
later round goes to the same agent, so it remembers what it already said. Tell it:

- The exact commands above, so it reviews this scope and not its default
  `main..HEAD`. Untracked files aren't in the diff; it has to read them.
- To read the surrounding code, not just the hunks, before claiming a defect.
- To number findings `R1-1`, `R1-2`, ... (round-finding) so we can refer to them
  across rounds. Each one: `file:line`, a severity (**critical** for bugs, security,
  data loss; **improvement** for performance, correctness-adjacent, maintainability;
  **nitpick** for style), the concrete mechanism - this input or state, this path,
  this wrong result - and a suggested fix.
- To end with a verdict: `APPROVE` when nothing above nitpick is open, otherwise
  `REQUEST_CHANGES`.
- Not to manufacture findings. A short clean review is a valid review.

## Your turn: address the findings

Work from the reviewer's list, in file order. For each finding:

1. **Verify before acting.** Read the code it points at. Check the callers, the
   types, the guards that already exist. A finding can be textbook-correct and still
   unreachable here.
2. **Fix** if it holds - the narrowest change that resolves it. Stay inside the
   changeset; a real problem in code this branch didn't touch gets noted for me, not
   fixed on the side.
3. **Decline** if it doesn't hold, with specifics: which link in the mechanism
   breaks, citing `file:line`. Disagreeing is a valid outcome. Don't manufacture a
   fix because a comment asserts a problem, and don't dismiss one because the fix
   looks annoying.
4. **Nitpicks**: take them when they're cheap and clearly better, decline the rest
   in one line. They never keep the loop going on their own.

After a round with edits, typecheck and lint before replying. Don't commit.

## Reply and re-review

Continue the same reviewer with `SendMessage` (load it with `ToolSearch` if the
schema isn't in your context). One message per round, listing every finding:

```
R1-1 FIXED - <what changed, where>
R1-2 DECLINED - <why, with file:line>
R1-3 FIXED (nitpick) - <one line>
```

Then ask it to re-review the current changeset with the same commands:

- Confirm each `FIXED` actually resolves the finding and didn't regress anything.
  Reopen it under a new number if not.
- Accept or contest each `DECLINED`. Contest once, with new evidence, or accept.
  Don't restate the original claim.
- Raise new findings only in code that changed since its last look, unless it
  genuinely missed something the first time.
- Same format, next round number, same verdict rule.

## Converging

The loop ends when any of these is true:

- The reviewer says `APPROVE`.
- Every remaining item is **stalemated**: you declined it, the reviewer contested,
  and you declined again on the same facts. Freeze those - no third exchange - and
  hand them to me.
- Four rounds have run. Anything still open is handed to me the same way.

If a round produces nothing but nitpicks, treat it as an `APPROVE` with notes.

## Report

The reviewer's messages never reach me directly, so finish with a summary that
stands on its own:

- Rounds run and the final verdict.
- Each finding once, with its location and outcome: fixed, declined, or stalemated,
  with one line of why. Fixed items are in the diff; I'll read those there.
- Stalemates get both positions in a sentence each. Those are my call.
- Anything out of scope the reviewer or you noticed but left alone.

If a hunk session is open (`hunk session list` is non-empty), also leave
each stalemate on its line so it reads against the code:

```
hunk session comment add --repo . --file <path> --new-line <n> \
   --summary "<the short version>" --rationale "<both positions>" --author claude
```

Don't commit. I'll read the result and either send it another round with
`/resolve` or open the PR.
