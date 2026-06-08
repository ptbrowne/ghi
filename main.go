package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var version = "dev"

func issuesPath(repo string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ghi", "issues", repo)
}

func ensureIssuesRepo(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return nil // already a git repo
	}
	return runPassthrough("git", "-C", dir, "init", "-q")
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prev := '-'
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
			prev = c
		} else if prev != '-' {
			b.WriteByte('-')
			prev = '-'
		}
	}
	return strings.Trim(b.String(), "-")
}

func main() {
	// Global flags
	global := flag.NewFlagSet("ghi", flag.ExitOnError)
	repo := global.String("repo", "", "override repo (owner/repo)")
	global.Usage = usage

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	// Parse flags up to the subcommand
	global.Parse(os.Args[1:])
	args := global.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	if *repo == "" {
		r, err := getRepo()
		if err != nil {
			fatal(err)
		}
		*repo = r
	}

	cmd, rest := args[0], args[1:]
	var err error

	switch cmd {
	case "fetch":
		fs := flag.NewFlagSet("fetch", flag.ExitOnError)
		fs.Usage = func() {
			fmt.Fprintln(os.Stderr, "Usage: ghi fetch NUMBER [NUMBER ...]")
		}
		fs.Parse(rest)
		if fs.NArg() == 0 {
			fs.Usage()
			os.Exit(1)
		}
		nums := make([]int, fs.NArg())
		for i, a := range fs.Args() {
			nums[i], err = strconv.Atoi(a)
			if err != nil {
				fatal(fmt.Errorf("invalid issue number: %s", a))
			}
		}
		err = cmdFetch(*repo, nums)

	case "sync":
		fs := flag.NewFlagSet("sync", flag.ExitOnError)
		dryRun := fs.Bool("dry-run", false, "preview without writing")
		fs.Usage = func() {
			fmt.Fprintln(os.Stderr, "Usage: ghi sync [--dry-run]")
		}
		fs.Parse(rest)
		err = cmdSync(*repo, *dryRun)

	case "path":
		fmt.Println(issuesPath(*repo))

	case "prune":
		fs := flag.NewFlagSet("prune", flag.ExitOnError)
		dryRun := fs.Bool("dry-run", false, "preview without removing")
		fs.Usage = func() {
			fmt.Fprintln(os.Stderr, "Usage: ghi prune [--dry-run]")
		}
		fs.Parse(rest)
		err = cmdPrune(*repo, *dryRun)

	case "version":
		fmt.Println(version)
		return

	case "discover":
		var config *ProjectConfig
		config, err = discover(*repo)
		if err == nil {
			fmt.Printf("Project: [%d] %s\n", config.ProjectNumber, config.ProjectTitle)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}

	if err != nil {
		fatal(err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `ghi — GitHub Issues local workflow

Usage:
  ghi [--repo OWNER/REPO] fetch NUMBER [NUMBER ...]
  ghi [--repo OWNER/REPO] sync [--dry-run]
  ghi [--repo OWNER/REPO] prune [--dry-run]
  ghi [--repo OWNER/REPO] discover
  ghi [--repo OWNER/REPO] path
  ghi version`)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
