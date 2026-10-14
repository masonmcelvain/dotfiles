---
name: review-pr
description: Help me review a peer's pull request - brief me on its intent and risk, answer my questions about it, keep my findings in a review file I can edit, and post the review for me. Use when I ask to review someone else's PR.
argument-hint: "<PR number or URL>"
---

# Review a PR

A peer has asked me to review their change. I'm the reviewer and the review is
mine: I decide whether the change is right, and everything that gets posted goes
out under my name. Your job is to make me fast and accurate at that. Help me
understand what the change is for, find the evidence I ask for, keep track of what
I've decided, and post the review when I'm done.

What I'm judging is the intent of the change: what it's for, whether it does that,
and what else it changes along the way. Reading every line for defects is not the
goal, and a pile of model-generated nits is worse than none. You can be thorough
in what you read and still be sparing in what you raise.

`$ARGUMENTS` is a PR number or URL. If it's empty, ask which PR.

## The review file

Everything lives in one markdown file:
`~/reviews/<owner>-<repo>-<number>.md` (create `~/reviews` if needed). I'll have
it open in my editor and change it, so **re-read it before every write**, and edit
only the part you mean to change. Never overwrite my wording.

If the file already exists, this is a resumed review. Read it, then skip to
[Resuming](#resuming).

```markdown
---
pr: https://github.com/<owner>/<repo>/pull/<n>
head: <sha the review was written against>
verdict: comment        # approve | request-changes | comment
---

## Review

<The top-level review body. Starts empty; I fill it, or ask you to draft it.>

## Comments

### path/to/file.php:42
<One inline comment. Its anchor is the line in the new version of the file.>

### path/to/file.ts:10-14
<A range of lines in the new version.>

### path/to/old.py:7 (old)
<A deleted line, anchored to the old version.>

---

## Notes (never posted)

**Intent:** <one or two sentences>

### Still to decide
- [ ] <question we need answered to judge the change>
- [x] <answered question> - <the answer, with file:line>

### Candidates
- <findings from a mechanical pass that I haven't accepted yet>
```

Everything above the bare `---` line gets posted. Everything below it is your
working memory and stays private. I can add, delete or reword anything above the
line, including comments of my own that you never saw.

## 1. Set up

```
gh pr view <pr> --json number,url,title,body,author,baseRefName,headRefName,headRefOid,files,additions,deletions,closingIssuesReferences,reviews,comments,statusCheckRollup
```

Check out the PR in the current checkout so you can read and run its code.
If `git status --porcelain` shows uncommitted changes, stop and ask me before
touching anything. Otherwise note the current branch so you can tell me how to get
back to it, then run `gh pr checkout <pr>`.

Read the PR description, any linked issues (`gh issue view`), and the existing
review threads, so you don't raise something that's already been discussed. Then
read the diff from the merge base (`git diff origin/<base>...HEAD`) and enough of
the surrounding code to understand what it touches: callers, tests and the
interfaces it changes. For a large PR, send the wider code reading to `Explore`
agents and keep your own context for the conversation.

## 2. Brief me

Write the review file with an empty Review and Comments section, the intent in
Notes, and a starting "Still to decide" checklist. Then brief me in chat, short
and claim first:

1. **Intent.** What problem this solves and for whom, in one or two sentences.
   If the PR doesn't say, infer it, and say that you inferred it.
2. **Approach.** How the change gets there, at the level of design rather than files.
3. **Unmentioned changes.** Existing behavior this changes that the description
   doesn't mention.
4. **Hard to reverse.** Migrations, schema, persisted data, public contracts and
   external side effects. Say "nothing" when nothing applies.
5. **Where the risk is.** The few places where a mistake would hurt, each with one
   line of why.
6. **Reading order.** The files I should read, in an order that makes sense of
   the change, skipping the mechanical parts.

End with the status line (below). Don't list defects in the briefing. If you
noticed something that looks wrong, it becomes an item in "Still to decide", for
me to look at.

## 3. Answer my questions

I'll grill you about the change. For each question:

- Lead with the answer, then the evidence, citing `file:line`. If you can't settle
  it from the code (it depends on runtime data, production config or the author's
  intent), say what would settle it. That might be a question for the author.
- Read the code before answering. Don't answer from a plausible reading of a name.
  Running things is fine: tests, a REPL, a script in the scratchpad. Keep any
  edits to the checkout throwaway, and revert them before you finish.
- If an answer settles a "Still to decide" item, check it off with the answer.
  If it raises a new one, add it.
- If an answer looks like a problem with the change, say so plainly and offer
  to record it ("Worth a comment?"). Don't record it yourself.

**Keep the plot.** We're here to decide on this change. Tangents are fine, and
often where the real understanding comes from. If I've spent three or more
exchanges on something that doesn't bear on this PR, answer, then say briefly that
we've drifted and name the next open item. Don't refuse and don't lecture.

**Status line.** End every reply with one line on where the review stands, e.g.:

> 2 comments · open: rollback of the migration, cache invalidation · not yet read: `jobs/`

## 4. Record findings

Only I decide what goes in the review. When I say to record something ("note
that", "flag this", "comment on line 42"), add it under Comments with the right
anchor, drafted the way I'd write it:

- Claim first, then the reason, then a suggestion if there is one. Plain words.
  No praise padding, no hedging stack, no 🤖. This is my review.
- Before it goes in, check that a finding holds up. Restate the mechanism: this
  state, this path, this wrong result. Check that it can actually happen here,
  given the callers, guards and types, and how much it would matter if it did. If
  it falls apart, tell me rather than recording it.
- Mark the weight only when it isn't obvious from the wording: "Blocking:",
  "Nit:", or "Question:" at the start.
- Anchor to the most specific line in the diff. A finding about the change as a
  whole goes into the Review section instead.

If I ask you to draft the top-level review or set the verdict, write it into the
file and tell me to look. If I ask for a mechanical bug pass, run the `code-review`
skill on the PR, without `--comment` or `--fix`. Put what it finds under
Candidates, each with your one-line take on whether it holds. Candidates move up
into Comments only when I accept them.

## 5. Post the review

When I say to post:

1. **Re-read the file.** I've probably edited it. Use only what's above the bare
   `---` line.
2. **Check the head.** If the PR has new commits since the `head` in the
   frontmatter, tell me what changed (`git fetch`, then a diff between the two
   shas, or say it was force-pushed if the old one is gone), and check whether any
   comment is now stale. Post against the current head.
3. **Check every anchor** against the diff's hunk headers (`gh pr diff <pr>`).
   GitHub rejects inline comments on lines outside the diff. Move any such comment
   into the review body as a `path:line` bullet, and tell me which ones you moved.
4. **Show me the exact payload** in chat: verdict, body and each comment. Then wait
   for a go-ahead, unless my "post it" already said to submit.
5. **Post** with one call, building the JSON with `jq` so quoting can't break it:

   ```
   gh api repos/<owner>/<repo>/pulls/<n>/reviews --method POST --input payload.json
   ```

   The payload has `commit_id` (the head sha), `body`, and `comments`. Each comment
   has `path`, `body`, `line` and `side` (`RIGHT`, or `LEFT` for `(old)`), plus
   `start_line`/`start_side` for a range.

   **By default, omit `event`.** That creates a *pending* review that only I can
   see, so I can submit it from GitHub with one last look. Include `event`
   (`APPROVE`, `REQUEST_CHANGES` or `COMMENT`, from `verdict`) only when I say to
   submit it. GitHub allows one pending review per PR. If one already exists, the
   call fails: tell me, and don't delete mine.

6. Give me the review's URL. Record the posted head in the frontmatter as
   `head:`. Move what you posted below the line, under `### Posted <date>: <url>`,
   so that Review and Comments start empty for the next round and nothing gets
   posted twice. Remind me which branch I was on before the checkout.

## Resuming

When the review file already exists:

- Check out the PR again (the same dirty-tree rule applies), and re-read the file,
  including my edits.
- If the head moved since `head`, lead with what changed in the new commits and
  whether it touches any recorded comment or open item. That is the re-review.
- Check whether anyone replied to the threads from the posted review (`gh api
  repos/<owner>/<repo>/pulls/<n>/comments`), and summarize the replies.
- Then give the status line and carry on from step 3.
