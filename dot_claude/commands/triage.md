---
description: Evaluate a review comment against the code, then propose a fix if it's a real problem
argument-hint: "<review comment>"
---

Triage this review comment:

<review-comment>
$ARGUMENTS
</review-comment>

Read the code the comment is about before forming an opinion. Then work through it in
order — stop at the first step that fails:

1. **Mechanism.** Restate the failure the comment claims, as a concrete sequence: this
   input or state, this code path, this wrong result. If the comment is vague, pin down
   the specific version of it that the code actually supports. Verify each link against
   the real code, not a plausible reading of it — check the callers, the types, the
   guards that already exist.

2. **Reality.** Can that sequence happen here? A mechanism can be textbook-correct and
   still be unreachable in this codebase — dead branch, caller that already validates,
   invariant held elsewhere, type that makes it impossible. Say which.

3. **Significance.** If it can happen, how much does it matter — what breaks, how often,
   how loudly, how recoverable. A real-but-negligible issue is worth naming as such
   rather than fixing.

Only if all three hold, brainstorm fixes:

- List 2–4 genuinely different approaches, not variations on one. Include the cheap
  narrow fix and at least one that addresses the underlying cause.
- For each: what it costs, what it risks, what it leaves unsolved.
- Recommend one and say why it beats the others. Note what would change your pick.

Rules:

- Disagreeing is a valid outcome, and so is partial agreement — "the mechanism is real
  but the impact isn't" or "the concern is right but the diagnosis isn't." Say which
  parts hold and which don't. Don't manufacture a problem because a comment asserts one,
  and don't dismiss one because the fix looks annoying.
- Cite specific code as `file.ts:42` when you claim something is or isn't reachable.
- If you can't settle it from the code — depends on runtime data, external behavior, or
  intent you don't have — say what you'd need to check instead of guessing.
- Propose only. Don't edit files unless I ask you to.
