---
description: Run a local code review loop before pushing to remote
---

Before I push this branch, I want to do a local review cycle. Here's the process:

1. Use the code-reviewer agent to review all commits on this branch that aren't on main (or the base branch if I specify one: $ARGUMENTS).
2. Show me the reviewer's findings.
3. For any issues marked Critical or Improvements:
   - Tell me what you plan to do about each one.
   - If you agree with the feedback, make the fix and amend the relevant commit (or add a fixup commit).
   - If you disagree, explain why.
4. After addressing feedback, run the code-reviewer agent again to verify.
5. Repeat until the reviewer returns APPROVE or until I tell you to stop.
6. When approved, show me a final summary and ask if I want to push.

Important: don't push automatically. Always ask me first.
