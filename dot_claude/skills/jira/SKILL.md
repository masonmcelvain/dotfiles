---
name: jira
description: Read and write Jira Cloud tickets (work items) with Atlassian's acli - view, search with JQL, create, edit, comment, assign and transition. Use when a request names Jira, a ticket, or a key like ABC-123.
---

# Jira (`acli`)

`acli` is Atlassian's official CLI, and only works with Jira Cloud. Jira calls
tickets "work items", so every ticket command is `acli jira workitem ...`.
Pull requests live in Bitbucket; use the `bkt` skill for those.

If a command fails on auth, run `acli jira auth status` and ask me to log in
with `! acli jira auth login --web`; never handle a token yourself.

## Reading

```bash
acli jira workitem view ABC-123                       # key, type, summary, status, assignee, description
acli jira workitem view ABC-123 --fields '*all' --json
acli jira workitem view ABC-123 --fields summary,comment
acli jira workitem comment list --key ABC-123 --json

acli jira workitem search --jql 'assignee = currentUser() AND statusCategory != Done' --json
acli jira workitem search --jql 'project = ABC ORDER BY updated DESC' --limit 20 --fields key,summary,status
acli jira workitem search --jql 'project = ABC' --count
```

Prefer `--json` and narrow `--fields` when reading for yourself; the default
table truncates. `--paginate` fetches every page of a search, so pair it with a
tight JQL query. Other read-only groups: `acli jira project list`,
`acli jira board`, `acli jira sprint`, `acli jira filter`.

## Writing

Every command below changes Jira, which other people see and get notified
about. Only run one when I asked for that change, and report the key or URL
afterwards. Before creating a ticket on `nordicnaturals.atlassian.net`, load
the `nordic` skill for the fields it requires.

```bash
acli jira workitem create --project ABC --type Task --summary "..." --description-file desc.txt --json
acli jira workitem create --project ABC --type Bug --summary "..." --parent ABC-100 --label backend,cli
acli jira workitem edit --key ABC-123 --summary "..." --yes
acli jira workitem edit --key ABC-123 --description-file desc.txt --yes
acli jira workitem comment create --key ABC-123 --body-file comment.txt
acli jira workitem assign --key ABC-123 --assignee @me --yes
acli jira workitem transition --key ABC-123 --status "In Progress" --yes
```

- `edit`, `assign` and `transition` prompt for confirmation without `--yes`,
  which hangs a non-interactive shell. Pass `--yes` only with an explicit
  `--key`; never with `--jql` or `--filter`, which can match far more tickets
  than intended. List what a query matches first.
- Descriptions and comments are plain text or Atlassian Document Format (ADF)
  JSON, not Markdown, so `**bold**` and backticks show up literally. Write
  multi-line text to a file in the scratchpad and pass `--description-file` or
  `--body-file`, rather than quoting it inline.
- Status names in `transition --status` are per-project workflow names. If one
  is rejected, check the ticket's current status and ask me which to use.
- `create --generate-json` and `edit --generate-json` print a template for
  setting fields that have no flag, used with `--from-json`.

### Custom fields

Some projects require custom fields, and Jira only enforces that later: the
create succeeds, then assigning or transitioning the ticket fails. Set them at
create time, because `acli` can only set custom fields on `create`. `edit` has
no way to, so a ticket created without them has to be filled in through the
web UI.

Pass them in `additionalAttributes` of a `--from-json` file, keyed by field
ID. Rich-text fields take ADF, the same as `description`:

```json
{
  "projectKey": "ABC",
  "type": "Task",
  "summary": "...",
  "description": {"type": "doc", "version": 1, "content": [
    {"type": "paragraph", "content": [{"type": "text", "text": "..."}]}
  ]},
  "additionalAttributes": {
    "customfield_12345": {"type": "doc", "version": 1, "content": [
      {"type": "bulletList", "content": [
        {"type": "listItem", "content": [
          {"type": "paragraph", "content": [{"type": "text", "text": "First criterion"}]}
        ]}
      ]}
    ]}
  }
}
```

```bash
acli jira workitem create --from-json workitem.json --json
```

`acli` can't list fields by name. To find an ID, view a recent ticket of the
same type that has the field filled in, with `--fields '*all' --json`, and match
the `customfield_*` values to what the web UI shows.

`acli jira workitem <command> --help` has the rest, including `link`,
`attachment`, `watcher`, `clone` and `archive`.
