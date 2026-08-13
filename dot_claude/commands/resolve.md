---
description: Address inline REVIEW comments left in the code, then remove them
argument-hint: "[path filter]"
---

I've left review comments in the code as source comments — the local equivalent of
inline comments on a GitHub PR. Each marker looks like `REVIEW:146 why not use X?`
or `REVIEW:146-148 extract this` and sits on its own line **directly below** the
line(s) it comments on. The numbers are the commented line range as of when I wrote
the comment; if edits have shifted lines since, trust the marker's position (the
lines immediately above it) over the absolute numbers.

Find them all (include untracked files; `$ARGUMENTS` may narrow the paths):

```
git grep -n --untracked 'REVIEW:' -- $ARGUMENTS
```

Handle each one, in file order:

1. **Read the surrounding code first.** The marker is a comment on the line range
   above it — "this" and "here" refer to that code.

2. **Classify and act:**
   - **Question** (asks why/what/how, requests nothing): answer it in your reply.
     Only change code if the honest answer is "you're right, that's wrong".
   - **Change request**: verify the concern before implementing — check the callers,
     the types, the guards that already exist. If it holds, make the fix. If it
     doesn't, push back with specifics, citing `file:line`. Disagreeing is a valid
     outcome; don't manufacture a fix because a comment asserts a problem.
   - **Direct instruction** ("rename this", "extract this", "use X instead"): just
     do it.

3. **Delete the marker comment** once handled — fixed, answered, or declined alike.
   Your reply is the record; the code shouldn't be.

Skip matches that clearly aren't review markers (e.g. the string appearing in docs or
test fixtures) and say you skipped them.

When done: typecheck/lint if you changed code, then finish with a summary that quotes
each comment with its location and outcome — fixed, answered, or declined, with one
line of why. Don't commit; I'll review the result and either leave another round of
comments or commit myself.
