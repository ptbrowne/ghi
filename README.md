# ghi

Fetch GitHub Issues as local Markdown files, edit them with any tool, sync back to GitHub.

## Why

The web UI gets in the way when reorganising, splitting, or rewriting a batch of issues. `ghi` brings issues down to plain files in a local git repo — use your editor, `grep`, or an AI agent freely, then push everything back with one `sync`.

Workflow:

1. **Fetch** — `ghi fetch <numbers>` pulls issues to `~/.ghi/issues/<owner>/<repo>/` as Markdown with YAML frontmatter. Auto-committed.
2. **Work** — edit freely: rename files, split one into two, change frontmatter, rewrite the body.
3. **Sync** — `ghi sync` pushes every file back: updates existing issues, creates new ones, writes project fields. Auto-committed.
4. **Prune** — `ghi prune` removes local files for issues closed on GitHub.

## Features

- **Issues as local files** — stored as `<number>-<slug>.md` with plain Markdown body and YAML frontmatter.
- **Automatic version control** — the issues directory is a git repo; every `fetch` and `sync` commits automatically.
- **Project field sync** — discovers GitHub Projects V2 field options (Status, Priority, Sprint) on first use and caches them. `sync` writes them back from frontmatter.
- **Conflict detection** — `sync` skips files where the GitHub body changed since last fetch, preventing overwrites.
- **New issue creation** — files with no `number` in frontmatter are created as new issues on `sync`.

## Requirements

- [gh](https://cli.github.com/) — authenticated (`gh auth login`)

## Install

Download from the [releases page](https://github.com/ptbrowne/ghi/releases):

```sh
# Apple Silicon (M1/M2/M3/M4)
curl -L https://github.com/ptbrowne/ghi/releases/latest/download/ghi-darwin-arm64 -o /usr/local/bin/ghi
chmod +x /usr/local/bin/ghi

# Intel
curl -L https://github.com/ptbrowne/ghi/releases/latest/download/ghi-darwin-amd64 -o /usr/local/bin/ghi
chmod +x /usr/local/bin/ghi
```

macOS blocks unsigned binaries. Remove the quarantine flag after installing:

```sh
xattr -d com.apple.quarantine /usr/local/bin/ghi
```

## Build from source

Requires [Go](https://go.dev/) 1.21+.

```sh
git clone https://github.com/ptbrowne/ghi.git
cd ghi
go build -o ghi .
mv ghi /usr/local/bin/ghi
```

## Usage

Run from inside a GitHub repo, or pass `--repo` explicitly.

```sh
ghi fetch 12 34 55       # fetch issues locally
ghi sync --dry-run       # preview changes
ghi sync                 # push changes to GitHub
ghi prune                # remove closed issues locally
ghi path                 # print the local issues directory
ghi --repo owner/repo fetch 12
```

## Claude Code skill

A Claude Code skill is included at `.claude/skills/ghi.md`. It teaches Claude the fetch/edit/sync workflow, file format, and issue-splitting rules. To install it, copy the file to your Claude skills directory:

```sh
cp .claude/skills/ghi.md ~/.claude/skills/ghi.md
```

Then use it in any project with `/ghi` or by describing what you want ("fetch issue 12 and split it").

## Advanced

### File format

```markdown
---
number: 42
title: Add dark mode support
state: open
labels:
  - enhancement
  - frontend
milestone: v2.0
project:
  item_id: PVTI_abc123
  status: In Progress
  priority: High
  sprint: Sprint 14
github_sha1: a3f1bc...
---

Body text of the issue in Markdown.
```

Edit `title`, `labels`, `milestone`, `project.status`, `project.priority`, `project.sprint`, and body freely. Don't edit `number`, `item_id`, or `github_sha1` manually.

### Project discovery

Project field definitions are auto-discovered on first use and cached in `~/.ghi/cache/`. To refresh, run `ghi discover`.
