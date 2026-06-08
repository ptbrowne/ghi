package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdPrune(repo string, dryRun bool) error {
	dir := issuesPath(repo)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", dir, err)
	}

	var removed []string
	var labels []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		meta, body, err := parseFM(string(data))
		if err != nil {
			return err
		}
		if meta.Number == 0 || meta.GithubSha1 == "" {
			continue
		}
		if bodySHA1(body) != meta.GithubSha1 {
			continue
		}
		label := fmt.Sprintf("#%d", meta.Number)
		if dryRun {
			fmt.Printf("[dry-run] prune %s  %s\n", label, meta.Title)
			continue
		}
		fmt.Printf("Pruning %s: %s\n", label, meta.Title)
		if err := os.Remove(path); err != nil {
			return err
		}
		removed = append(removed, path)
		labels = append(labels, label)
	}

	if len(removed) > 0 {
		if err := gitStageIn(dir, removed...); err != nil {
			return err
		}
		return gitCommitIn(dir, "prune: "+strings.Join(labels, ", "))
	}
	return nil
}
