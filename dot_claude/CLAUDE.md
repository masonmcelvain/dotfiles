# Development guidelines

## General

- Use the GitHub CLI (`gh`) for all GitHub-related tasks when possible
- Be sure to typecheck and lint when you’re done making a series of code changes

When importing types in React, prefer `import { type TypeName } from React;` over using `React.TypeName` in the code.

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
- The first line must be 50 characters or less

**Example commit with scope**:

```
feat(lang): add Polish language
```

**Example commit with body**:

```
fix: mobile dropdown flickering

Prevent the mobile product option dropdown from flickering when first opened.
```

## Pull request guidelines

When writing pull requests, follow these guidelines:

- Title should be in 'Title case with Pronouns capitalized', without conventional commit type prefix
- If the PR closes an issue, you should write it at the top of PR body (`- close #1234`)
- If a PR is related to an issue but does not resolve it completely, then add at the top "- connect #1234"

**Example PR template**:

```
- close #1234

Brief description of the changes and their purpose.
```

## Browser Automation

Use `agent-browser` for web automation. Run `agent-browser --help` for all commands.

Core workflow:
1. `agent-browser open <url>` - Navigate to page
2. `agent-browser snapshot -i` - Get interactive elements with refs (@e1, @e2)
3. `agent-browser click @e1` / `fill @e2 "text"` - Interact using refs
4. Re-snapshot after page changes
