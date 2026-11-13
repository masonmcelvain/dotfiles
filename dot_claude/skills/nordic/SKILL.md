---
name: nordic
description: Nordic Naturals conventions for branches, commits and pull requests - the Jira key (US-1234) is the branch name and leads every commit message and PR title, and PRs go to Bitbucket with the changeset, QA and deploy steps in the description. Use before naming a branch, committing, or opening or editing a PR in a repo whose origin is in the nordicnaturals Bitbucket workspace, and before creating a ticket in nordicnaturals.atlassian.net Jira.
---

# Nordic Naturals

These conventions override my GitHub-flavored commit and PR guidelines in
CLAUDE.md wherever the two disagree. Everything they don't mention, such as
leading with the motivation and citing evidence, still applies.

## Is this a Nordic repo?

Check the remote, not the path. Nordic repos live in `~/code/`, but so may
others.

```bash
git remote get-url origin   # ...bitbucket.org:nordicnaturals/<repo>.git or .../nordicnaturals/<repo>
```

If origin isn't in the `nordicnaturals` workspace, none of this applies.

## The issue key

Every piece of work hangs off a Jira issue key like `US-1234` (other projects
use other prefixes). Take it from the branch name, the task or the issue I
linked. Never invent one. If there's no key yet, ask me for it
before the first commit, or, when you can't wait for me, use a placeholder
branch and reword the commits once I give you one (only while they're unpushed).

Jira is `https://nordicnaturals.atlassian.net`; read issues with the `jira`
skill, and link them as `https://nordicnaturals.atlassian.net/browse/US-1234`.
Don't change Jira (status, comments, assignee) unless I ask.

When I ask for a new US ticket, fill in its two required fields when you
create it. Without them, Jira takes the ticket but won't let anyone assign it,
and `acli` can't add them afterwards. Pass them in `additionalAttributes` with
`--from-json`, as ADF (see the `jira` skill's Custom fields section):

- **Acceptance Criteria** (`customfield_11249`): a bullet list of checkable
  outcomes.
- **Deployment Instructions** (`customfield_11250`): what has to happen at
  deploy, which app(s) it ships in, or `None.`

If either ID is rejected, look them up again on a recent US ticket that has
them filled in.

Pick the work type for what the ticket is, not `Task` by default:

- **Bug**: something that worked, or should work, is broken.
- **Enhancement**: an improvement to something that already exists.
- **Technical Story**: new engineering work, including tooling and refactors.
- **Discovery**: an investigation or spike whose output is an answer, not code.
- **Task**: a chore that fits none of the above.

Leave `Story`, `New Feature` and `Epic` to product unless I ask for one, and
never use `Test` (Zephyr's) or `Sub-task` without a `--parent`.

## Branches

The branch name is the key and nothing else: `US-1234`. One branch per issue.

## Commits

The key, a space, then a conventional commit subject:

```
US-1234 fix: keep the cart open after a failed update

Show the error inside the cart instead of closing it, so the shopper can
see what went wrong and retry without losing their place.
```

- Every commit on the branch carries the key, not just the first.
- The 50-character limit applies to the part after the key.
- Don't put the Jira URL in the commit; the key already links it.

## Pull requests

PRs live in Bitbucket Cloud; drive them with the `bkt` skill.

- **Title**: the key, then the same conventional subject a commit would get,
  in lowercase: `US-1234 fix: keep the cart open after a failed update`. For a
  one-commit PR, that's the commit's subject.
- **Target**: origin's default branch (`master`) unless the issue or I say
  otherwise. Some work targets a `release-x.y.z` branch instead; if the issue
  mentions a release, ask which.
- **Reviewers**: pass `--with-default-reviewers`. Bitbucket only adds a repo's
  default reviewers to PRs made in its web UI, not to ones made through the API.

The description is Markdown, laid out like this, with empty sections dropped
except Deploy:

```
https://nordicnaturals.atlassian.net/browse/US-1234

What changed and why, in one to three sentences, for a reviewer who hasn't
read the issue.

## Changes

- Bulleted changeset, grouped by area, only when the PR does several things

## QA

- What was verified and how: the page or path, the viewport, the data used,
  the commands run
- Manual steps a reviewer should repeat, if any

## Deploy

None.
```

`## Deploy` is always there: environment variables to set, CMS or platform
settings to change, scripts to run, the order things must happen in, or
`None.` when it's a plain deploy.

Bitbucket has no flag for a description file, so pass it with
`--description "$(cat pr-body.md)"`. Its API can't attach files to a PR either:
screenshots go in by hand. Name them in the QA section by file name (`See
ac1-card.png in the attachments`), and tell me which folder they're in so I can
drag them into the PR's attachments.

```bash
bkt pr create --title "US-1234 fix: keep the cart open after a failed update" \
  --target master --with-default-reviewers --description "$(cat pr-body.md)"
```

Opening a PR notifies reviewers, so only do it when I've asked for it.
