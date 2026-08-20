---
description: Address the review comments left in a live hunk session, then clear them
argument-hint: "[file filter]"
---

I'm reading this changeset in hunk and have left inline comments on the hunks I want
you to look at — the local equivalent of inline comments on a GitHub PR. The TUI is
mine and it's already open: talk to it through `hunk session`, and don't launch
`hunk diff` yourself.

Collect my comments first, in one call, before you change anything (`$ARGUMENTS` may
narrow it to one file):

```
hunk session comment list --repo . --type user --json
```

`--type user` is what separates my comments from any an agent left. Work from that
captured list for the rest of the run — your own edits shift the lines underneath,
and the session reloads as they do, so a list fetched later won't line up with this
one.

Each comment carries an id, a file path, and a line target. For the surrounding diff:

```
hunk session review --repo . --include-patch --json
```

Handle each one, in file order:

1. **Read the surrounding code first.** The comment is anchored to a line in the
   diff — "this" and "here" mean that hunk, not the file at large.

2. **Classify and act:**
   - **Question** (asks why/what/how, requests nothing): answer it. Only change code
     if the honest answer is "you're right, that's wrong".
   - **Change request**: verify the concern before implementing — check the callers,
     the types, the guards that already exist. If it holds, make the fix. If it
     doesn't, push back with specifics, citing `file:line`. Disagreeing is a valid
     outcome; don't manufacture a fix because a comment asserts a problem.
   - **Direct instruction** ("rename this", "extract this", "use X instead"): just
     do it.

3. **Answer where I'm looking.** For anything I don't get to read as a diff — an
   answer to a question, or a push-back — leave it on the same line so it renders
   against the code:

   ```
   hunk session comment add --repo . --file <path> --new-line <n> \
      --summary "<the short version>" --rationale "<the reasoning>" --author claude
   ```

   Fixes don't need one; I'll see those in the diff itself. I toggle these with `a`
   and clear them with `hunk session comment clear --repo . --yes`.

4. **Remove my comment** once handled — fixed, answered, or declined alike:
   `hunk session comment rm --repo . <comment-id>`. Your reply is the record.

If the session isn't running (`comment list` errors, or `hunk session list` comes
back empty), stop and tell me — don't fall back to grepping the tree for markers.

When done: typecheck/lint if you changed code, then finish with a summary that quotes
each comment with its location and outcome — fixed, answered, or declined, with one
line of why. Don't commit unless I say so; I'll review the result and potentially
leave another round of comments.
