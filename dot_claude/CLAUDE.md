# Development guidelines

## General

- Use `rg`, don't use `grep`
- Use the GitHub CLI (`gh`) for all GitHub-related tasks.
  - View issues and pull requests
  - Debug GitHub Actions failures

## Commit guidelines

Use conventional commit messages following this pattern:

```
<type>[optional scope]: <description>

[optional body]
```

- Prefer `feat` and `fix` for the type, but other types are allowed including `build`, `chore`, `ci`, `docs`, `style`, `refactor`, `perf`, and `test`.
- Use a concise scope when appropriate, but it is not always necessary.
- Use the imperative tense, not the passive tense. For example, "add Polish language" instead of "adds Polish language".
- Be brief and use complete sentences in commit bodies.

**Example commit with scope**:

```
feat(lang): add Polish language
```

**Example commit with body**:

```
fix: mobile dropdown flickering

Prevent the mobile product option dropdown from flickering when first opened.
```

Use the current git status, diff, branch name, and/or recent commit messages when drafting new commit messages.

- Current git status: !`git status`
- Current git diff: !`git diff`
- Current branch: !`git branch --show-current`
- Recent commits: !`git log --oneline -10`

## Pull request guidelines

When writing pull requests, follow these guidelines:

- Title should be in title case, without conventional commit type prefix
- QA section should use bullet points
- If the PR closes an issue, you should write it at the top of PR body (`Closes #1234`)
- If a PR is related to an issue but does not resolve it completely, then add at the top "Connects #1234"

**Example PR template**:

```
Closes #1234

## Summary

Brief description of the changes and their purpose.

## QA

- Verify feature works as expected
- Test different scenarios
- Check for any potential side effects
```
