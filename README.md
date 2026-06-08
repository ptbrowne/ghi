# ghi

A CLI for working on GitHub Issues locally — fetch them as Markdown files, edit them with any tool (including AI agents), then sync changes back to GitHub.

## Why

GitHub's web UI is fine for reading issues, but it gets in the way when you want to reorganise, split, or rewrite a batch of them. `ghi` brings issues down to plain Markdown files in a local git repo, so you can use your editor, `grep`, or an AI agent to work on them freely — then push everything back in one `sync`.

The workflow:

1. **Fetch** — `ghi fetch <numbers>` pulls specific issues to `~/.ghi/issues/<owner>/<repo>/` as Markdown files with YAML frontmatter. Each fetch is committed automatically.
2. **Work** — edit the files however you like: rename them, split one file into two, add/change frontmatter fields, rewrite the body. An AI agent works well here.
3. **Sync** — `ghi sync` pushes every file back to GitHub: updating existing issues, creating new ones, and updating project fields (Status, Priority, Sprint). Each sync is committed automatically.
4. **Prune** — `ghi prune` removes local files for issues that are closed on GitHub.

## Features

- **Issues as local files** — fetched to `~/.ghi/issues/<owner>/<repo>/` as `<number>-<slug>.md`. The body is plain Markdown; metadata lives in YAML frontmatter.
- **Automatic version control** — the issues directory is a git repo. Every `fetch` and `sync` commits automatically, giving you a full history of local edits.
- **Project field sync** — on first use, `ghi` queries GitHub Projects V2 to discover the available Status, Priority, and Sprint field options and caches them locally. `sync` writes these fields back when you change them in frontmatter.
- **Conflict detection** — `sync` checks whether the GitHub body changed since you last fetched. If it did, it skips that file and asks you to re-fetch, preventing overwrites.
- **New issue creation** — files without a `number` in frontmatter are created as new GitHub issues on `sync`, then renamed to include the assigned number.

## Requirements

- [Go](https://go.dev/) 1.21+
- [gh](https://cli.github.com/) — GitHub CLI, authenticated (`gh auth login`)

## Install

Download the latest binary for your architecture from the [releases page](https://github.com/ptbrowne/ghi/releases), then move it to somewhere on your PATH:

```sh
# Apple Silicon
curl -L https://github.com/ptbrowne/ghi/releases/latest/download/ghi-darwin-arm64 -o /usr/local/bin/ghi
chmod +x /usr/local/bin/ghi

# Intel
curl -L https://github.com/ptbrowne/ghi/releases/latest/download/ghi-darwin-amd64 -o /usr/local/bin/ghi
chmod +x /usr/local/bin/ghi
```

## Build from source

```sh
git clone https://github.com/ptbrowne/ghi.git
cd ghi
go build -o ghi .
# optionally move to somewhere on your PATH:
mv ghi /usr/local/bin/ghi
```

## Usage

Run all commands from inside a GitHub repository, or pass `--repo` explicitly.

```sh
# Fetch issues #12, #34, and #55 to local Markdown files
ghi fetch 12 34 55

# Show the local issues directory path
ghi path

# Sync all local files back to GitHub (preview first with --dry-run)
ghi sync --dry-run
ghi sync

# Remove local files for issues that are closed on GitHub
ghi prune --dry-run
ghi prune

# Override the repo (useful outside a git directory)
ghi --repo owner/repo fetch 12
```

## Advanced

### File format

Each issue is a Markdown file with YAML frontmatter:

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

Edit the `title`, `labels`, `milestone`, `project.status`, `project.priority`, `project.sprint`, and body freely. `ghi sync` will write them back to GitHub. Do not edit `number`, `item_id`, or `github_sha1` manually.

### Project discovery

`ghi` automatically discovers the GitHub Project linked to your repo on first use and caches the field definitions in `~/.ghi/cache/`. To refresh the cache, run `ghi discover`.
