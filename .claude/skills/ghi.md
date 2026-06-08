---
name: ghi
description: Work with GitHub issues locally as Markdown files. Use when the user wants to fetch issues, edit them offline, sync changes back to GitHub, split issues by done/undone or by section, or create new issues. Trigger phrases: "fetch issue", "sync issue", "split issue", "create issue from file".
---

# ghi

Fetch → edit → sync. Issues live in `~/.ghi/issues/<owner>/<repo>/` — a git repo `ghi` manages automatically. All commands run from inside the project repo; the remote is auto-discovered.

## Commands

```bash
ghi fetch 12 34          # fetch issues locally (auto-commits)
ghi sync --dry-run       # preview
ghi sync                 # push changes to GitHub (auto-commits)
ghi discover             # refresh project field cache (re-run each sprint)
```

## File format

Existing issue — `~/.ghi/issues/<owner>/<repo>/<number>-<slug>.md`:
```yaml
---
number: 12
title: My issue
state: open
labels: [bug]
milestone: v1.0
project:
  item_id: PVTI_...
  status: In Progress
  priority: P1
  sprint: Sprint 4
github_sha1: a3f2c1...
---

Body content...
```

New issue — `new-<slug>.md` (no `number`, no `github_sha1`; filled in by sync):
```yaml
---
title: New issue
labels: [bug]
project:
  status: To Do
  priority: P2
  sprint: Sprint 4
---

Body content...
```

## Splitting an issue

Two modes:

- **done/undone** — `- [x]` items stay in original; `- [ ]` items move to a new file
- **by-section** — each `###` section with incomplete items becomes its own new file

```bash
ghi fetch 12
# edit files
ghi sync --dry-run && ghi sync
```

### done/undone rules

**Original:**
- Keep `- [x]` lines and indented children; drop `- [ ]` and theirs
- Prune sections where nothing remains
- Append ` 1` to title; set `project.status: "Self Review"`; keep `github_sha1`

**New file** `new-<slug>.md`:
- Keep `- [ ]` lines; drop `- [x]`; prune empty sections
- Prepend `Originally split from <url>\n\n`
- Title = original + ` 2`; same `labels`, `milestone`, `priority`, `sprint`; set `status: "To Do"`; omit `item_id` and `github_sha1`

### by-section rules

**Original:** keep only `- [x]` lines; prune empty sections; append ` 1` to title; set `status: "Self Review"`

**One new file per section** `new-<slug>-<section-slug>.md`:
- Full section content regardless of done/undone state
- Title: `<Original title> - <Section name>`
- Same `labels`, `milestone`, `priority`, `sprint`; set `status: "To Do"`
- Section slug: lowercase, spaces→hyphens, strip special chars

**Both modes:** nested checkboxes follow parent; `<details>` blocks belong to section above.

## Gotchas

- **`project` scope** — run `! gh auth refresh -s project` if you see INSUFFICIENT_SCOPES
- **Sprint cache** — re-run `ghi discover` each new sprint (iteration IDs change)
- **`github_sha1`** — keep it in the original when splitting; sync uses it to guard updates
- **Labels** — frontmatter label list replaces the GitHub label set entirely on sync
