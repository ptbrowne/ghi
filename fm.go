package main

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type IssueMeta struct {
	Number     int          `yaml:"number,omitempty"`
	Title      string       `yaml:"title"`
	State      string       `yaml:"state,omitempty"`
	Labels     []string     `yaml:"labels,omitempty"`
	Milestone  string       `yaml:"milestone,omitempty"`
	Project    *ProjectMeta `yaml:"project,omitempty"`
	GithubSha1 string       `yaml:"github_sha1,omitempty"`
}

type ProjectMeta struct {
	ItemID   string `yaml:"item_id,omitempty"`
	Status   string `yaml:"status,omitempty"`
	Priority string `yaml:"priority,omitempty"`
	Sprint   string `yaml:"sprint,omitempty"`
}

func parseFM(text string) (IssueMeta, string, error) {
	if !strings.HasPrefix(text, "---\n") {
		return IssueMeta{}, text, nil
	}
	rest := text[4:]
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return IssueMeta{}, text, fmt.Errorf("unclosed frontmatter")
	}
	var meta IssueMeta
	if err := yaml.Unmarshal([]byte(rest[:end]), &meta); err != nil {
		return IssueMeta{}, text, err
	}
	body := strings.TrimLeft(rest[end+5:], "\n")
	return meta, body, nil
}

func writeFM(meta IssueMeta, body string) string {
	b, _ := yaml.Marshal(meta)
	front := strings.TrimRight(string(b), "\n")
	return "---\n" + front + "\n---\n\n" + strings.TrimLeft(body, "\n")
}
