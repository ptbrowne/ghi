package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	args := os.Args[1:]
	repo := ""

	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--repo" {
			repo = args[i+1]
			args = append(args[:i], args[i+2:]...)
			break
		}
	}

	if repo == "" {
		var err error
		repo, err = getRepo()
		if err != nil {
			fatal(err)
		}
	}

	cmd, rest := args[0], args[1:]
	var err error

	switch cmd {
	case "fetch":
		if len(rest) == 0 {
			fatal(fmt.Errorf("fetch requires at least one issue number"))
		}
		nums := make([]int, len(rest))
		for i, a := range rest {
			nums[i], err = strconv.Atoi(a)
			if err != nil {
				fatal(fmt.Errorf("invalid issue number: %s", a))
			}
		}
		err = cmdFetch(repo, nums)
	case "sync":
		dryRun := false
		for _, a := range rest {
			if a == "--dry-run" {
				dryRun = true
			}
		}
		err = cmdSync(repo, dryRun)
	case "discover":
		var config *ProjectConfig
		config, err = discover(repo)
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
  ghi [--repo OWNER/REPO] discover`)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
