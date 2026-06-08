package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

func cmdSync(repo string, dryRun bool) error {
	config, err := loadCache(repo)
	if err != nil {
		return err
	}
	owner := strings.SplitN(repo, "/", 2)[0]

	dir := issuesPath(repo)
	if err := ensureIssuesRepo(dir); err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)

	if len(files) == 0 {
		return fmt.Errorf("no .md files in %s", dir)
	}

	if dryRun {
		for _, path := range files {
			data, _ := os.ReadFile(path)
			meta, _, _ := parseFM(string(data))
			tag := "NEW"
			if meta.Number > 0 {
				tag = fmt.Sprintf("#%d", meta.Number)
			}
			fmt.Printf("[dry-run] %6s  %s\n", tag, meta.Title)
		}
		return nil
	}

	type result struct {
		path  string
		label string
		log   bytes.Buffer
		err   error
	}
	results := make([]result, len(files))
	var wg sync.WaitGroup
	for i, path := range files {
		wg.Add(1)
		go func(i int, path string) {
			defer wg.Done()
			r := &results[i]
			r.path, r.label, r.err = syncFile(path, dir, repo, config, owner, &r.log)
		}(i, path)
	}
	wg.Wait()

	var updated []string
	var labels []string
	for _, r := range results {
		io.Copy(os.Stdout, &r.log)
		if r.err != nil {
			return r.err
		}
		if r.path != "" {
			updated = append(updated, r.path)
			labels = append(labels, r.label)
		}
	}

	if len(updated) > 0 {
		if err := gitStageIn(dir, updated...); err != nil {
			return err
		}
		return gitCommitIn(dir, "sync: "+strings.Join(labels, ", "))
	}
	return nil
}

func syncFile(path, dir, repo string, config *ProjectConfig, owner string, log io.Writer) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	meta, body, err := parseFM(string(data))
	if err != nil {
		return "", "", err
	}

	if meta.Number > 0 {
		if meta.GithubSha1 != "" {
			issue, err := ghGetIssue(repo, meta.Number)
			if err != nil {
				return "", "", err
			}
			if bodySHA1(issue.Body) != meta.GithubSha1 {
				fmt.Fprintf(log, "  SKIP #%d: GitHub body changed since fetch. Re-fetch to proceed.\n", meta.Number)
				return "", "", nil
			}
		}

		fmt.Fprintf(log, "Updating #%d: %s\n", meta.Number, meta.Title)
		if err := ghEditIssue(repo, meta.Number, meta.Title, body, meta.Labels, meta.Milestone); err != nil {
			return "", "", err
		}

		refreshed, err := ghGetIssue(repo, meta.Number)
		if err != nil {
			return "", "", err
		}
		meta.GithubSha1 = bodySHA1(refreshed.Body)

		itemID := ""
		if meta.Project != nil {
			itemID = meta.Project.ItemID
		}
		if itemID == "" {
			itemID, err = ghProjectItemEnsure(config.ProjectNumber, owner, refreshed.HTMLURL, refreshed.NodeID, config.ProjectID)
			if err != nil {
				return "", "", err
			}
			if meta.Project == nil {
				meta.Project = &ProjectMeta{}
			}
			meta.Project.ItemID = itemID
		}
		if itemID != "" {
			if err := applyProjectFields(config, itemID, meta.Project); err != nil {
				return "", "", err
			}
		}

		label := fmt.Sprintf("#%d", meta.Number)
		return path, label, os.WriteFile(path, []byte(writeFM(meta, body)), 0644)

	} else {
		fmt.Fprintf(log, "Creating: %s\n", meta.Title)
		number, url, err := ghCreateIssue(repo, meta.Title, body, meta.Labels, meta.Milestone)
		if err != nil {
			return "", "", err
		}
		fmt.Fprintf(log, "  → #%d  %s\n", number, url)

		refreshed, err := ghGetIssue(repo, number)
		if err != nil {
			return "", "", err
		}
		meta.GithubSha1 = bodySHA1(refreshed.Body)

		itemID, err := ghProjectItemEnsure(config.ProjectNumber, owner, url, refreshed.NodeID, config.ProjectID)
		if err != nil {
			return "", "", err
		}
		meta.Number = number
		if meta.Project == nil {
			meta.Project = &ProjectMeta{}
		}
		meta.Project.ItemID = itemID

		if err := applyProjectFields(config, itemID, meta.Project); err != nil {
			return "", "", err
		}

		newPath := filepath.Join(dir, fmt.Sprintf("%d-%s.md", number, slugify(meta.Title)))
		if err := os.WriteFile(newPath, []byte(writeFM(meta, body)), 0644); err != nil {
			return "", "", err
		}
		os.Remove(path)

		label := fmt.Sprintf("#%d", number)
		return newPath, label, nil
	}
}
