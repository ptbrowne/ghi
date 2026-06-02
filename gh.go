package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type ghIssueJSON struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	Body    string `json:"body"`
	NodeID  string `json:"node_id"`
	HTMLURL string `json:"html_url"`
	Labels  []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
}

func run(args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func runPassthrough(args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getRepo() (string, error) {
	u, err := run("git", "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("could not get git remote: %w", err)
	}
	re := regexp.MustCompile(`.*github\.com[:/]`)
	u = re.ReplaceAllString(u, "")
	return strings.TrimSuffix(u, ".git"), nil
}

func gitStageIn(dir string, paths ...string) error {
	return runPassthrough(append([]string{"git", "-C", dir, "add", "--"}, paths...)...)
}

func gitCommitIn(dir, msg string) error {
	// Check if there's anything staged before committing
	cmd := exec.Command("git", "-C", dir, "diff", "--cached", "--quiet")
	if cmd.Run() == nil {
		return nil // nothing staged
	}
	return runPassthrough("git", "-C", dir, "commit", "-m", msg)
}

func ghGetIssue(repo string, number int) (*ghIssueJSON, error) {
	out, err := run("gh", "api", fmt.Sprintf("repos/%s/issues/%d", repo, number))
	if err != nil {
		return nil, err
	}
	var issue ghIssueJSON
	return &issue, json.Unmarshal([]byte(out), &issue)
}

func ghGraphQL(query string) (map[string]any, error) {
	out, err := run("gh", "api", "graphql", "-f", "query="+query)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	return result, json.Unmarshal([]byte(out), &result)
}

func ghCreateIssue(repo, title, body string, labels []string, milestone string) (int, string, error) {
	args := []string{"gh", "issue", "create", "--repo", repo, "--title", title, "--body", body}
	for _, l := range labels {
		args = append(args, "--label", l)
	}
	if milestone != "" {
		args = append(args, "--milestone", milestone)
	}
	url, err := run(args...)
	if err != nil {
		return 0, "", err
	}
	parts := strings.Split(strings.TrimRight(url, "/"), "/")
	n, _ := strconv.Atoi(parts[len(parts)-1])
	return n, url, nil
}

func ghEditIssue(repo string, number int, title, body string, labels []string, milestone string) error {
	args := []string{"gh", "issue", "edit", strconv.Itoa(number), "--repo", repo,
		"--title", title, "--body", body}
	if milestone != "" {
		args = append(args, "--milestone", milestone)
	} else {
		args = append(args, "--remove-milestone")
	}
	if err := runPassthrough(args...); err != nil {
		return err
	}
	// Replace labels via REST (gh CLI only adds/removes, doesn't set the full list)
	patchArgs := []string{"gh", "api", fmt.Sprintf("repos/%s/issues/%d", repo, number), "-X", "PATCH"}
	for _, l := range labels {
		patchArgs = append(patchArgs, "-F", "labels[]="+l)
	}
	_, err := run(patchArgs...)
	return err
}

func ghProjectItemAdd(projectNumber int, owner, issueURL string) (string, error) {
	out, err := run("gh", "project", "item-add", strconv.Itoa(projectNumber),
		"--owner", owner, "--url", issueURL, "--format", "json")
	if err != nil {
		return "", err
	}
	var result struct {
		ID string `json:"id"`
	}
	return result.ID, json.Unmarshal([]byte(out), &result)
}

func bodySHA1(body string) string {
	h := sha1.New()
	h.Write([]byte(body))
	return fmt.Sprintf("%x", h.Sum(nil))
}
