package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ProjectConfig struct {
	Repo          string         `json:"repo"`
	ProjectID     string         `json:"project_id"`
	ProjectNumber int            `json:"project_number"`
	ProjectTitle  string         `json:"project_title"`
	Fields        []ProjectField `json:"fields"`
}

type ProjectField struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Type          string        `json:"type"`
	Options       []FieldOption `json:"options,omitempty"`
	Configuration *IterConfig   `json:"configuration,omitempty"`
}

type FieldOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IterConfig struct {
	Iterations          []Iteration `json:"iterations"`
	CompletedIterations []Iteration `json:"completedIterations"`
}

type Iteration struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func cachePath(repo string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "cache", "split-issues",
		strings.ReplaceAll(repo, "/", "--")+".json")
}

func loadCache(repo string) (*ProjectConfig, error) {
	data, err := os.ReadFile(cachePath(repo))
	if err != nil {
		fmt.Printf("No cache for %s — running discovery...\n", repo)
		return discover(repo)
	}
	var c ProjectConfig
	return &c, json.Unmarshal(data, &c)
}

func (c *ProjectConfig) fieldByName(name string) *ProjectField {
	for i := range c.Fields {
		if c.Fields[i].Name == name {
			return &c.Fields[i]
		}
	}
	return nil
}

func (c *ProjectConfig) optionID(fieldName, optionName string) string {
	if f := c.fieldByName(fieldName); f != nil {
		for _, o := range f.Options {
			if o.Name == optionName {
				return o.ID
			}
		}
	}
	return ""
}

func (c *ProjectConfig) iterationID(sprintTitle string) string {
	if f := c.fieldByName("Sprint"); f != nil && f.Configuration != nil {
		for _, it := range f.Configuration.Iterations {
			if it.Title == sprintTitle {
				return it.ID
			}
		}
	}
	return ""
}

func applyProjectFields(config *ProjectConfig, itemID string, project *ProjectMeta) error {
	if project == nil || config == nil {
		return nil
	}
	pid := config.ProjectID
	var clauses []string

	addSingleSelect := func(alias, fieldName, valueName string) {
		if valueName == "" {
			return
		}
		f := config.fieldByName(fieldName)
		opt := config.optionID(fieldName, valueName)
		if f != nil && opt != "" {
			clauses = append(clauses, fmt.Sprintf(
				`%s: updateProjectV2ItemFieldValue(input: {projectId: "%s", itemId: "%s", fieldId: "%s", value: {singleSelectOptionId: "%s"}}) {projectV2Item {id}}`,
				alias, pid, itemID, f.ID, opt,
			))
		}
	}

	addIteration := func(alias, sprintTitle string) {
		if sprintTitle == "" {
			return
		}
		f := config.fieldByName("Sprint")
		it := config.iterationID(sprintTitle)
		if f != nil && it != "" {
			clauses = append(clauses, fmt.Sprintf(
				`%s: updateProjectV2ItemFieldValue(input: {projectId: "%s", itemId: "%s", fieldId: "%s", value: {iterationId: "%s"}}) {projectV2Item {id}}`,
				alias, pid, itemID, f.ID, it,
			))
		}
	}

	addSingleSelect("setStatus", "Status", project.Status)
	addSingleSelect("setPriority", "Project Priority", project.Priority)
	addIteration("setSprint", project.Sprint)

	if len(clauses) > 0 {
		_, err := ghGraphQL("mutation {" + strings.Join(clauses, " ") + "}")
		return err
	}
	return nil
}

func discover(repo string) (*ProjectConfig, error) {
	owner := strings.SplitN(repo, "/", 2)[0]

	out, err := run("gh", "project", "list", "--owner", owner, "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	var pl struct {
		Projects []struct {
			ID     string `json:"id"`
			Number int    `json:"number"`
			Title  string `json:"title"`
		} `json:"projects"`
	}
	if err := json.Unmarshal([]byte(out), &pl); err != nil {
		return nil, err
	}
	if len(pl.Projects) == 0 {
		return nil, fmt.Errorf("no projects found for %s", owner)
	}
	p := pl.Projects[0]

	out, err = run("gh", "project", "field-list", strconv.Itoa(p.Number),
		"--owner", owner, "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("listing fields: %w", err)
	}
	var fl struct {
		Fields []ProjectField `json:"fields"`
	}
	if err := json.Unmarshal([]byte(out), &fl); err != nil {
		return nil, err
	}

	for i, field := range fl.Fields {
		if field.Type != "ProjectV2IterationField" {
			continue
		}
		data, err := ghGraphQL(fmt.Sprintf(`{
          node(id: "%s") {
            ... on ProjectV2IterationField {
              configuration {
                iterations { id title startDate duration }
                completedIterations { id title startDate duration }
              }
            }
          }
        }`, field.ID))
		if err != nil {
			return nil, err
		}
		if cfg := deepGet(data, "data", "node", "configuration"); cfg != nil {
			b, _ := json.Marshal(cfg)
			var iterCfg IterConfig
			if json.Unmarshal(b, &iterCfg) == nil {
				fl.Fields[i].Configuration = &iterCfg
			}
		}
	}

	config := &ProjectConfig{
		Repo:          repo,
		ProjectID:     p.ID,
		ProjectNumber: p.Number,
		ProjectTitle:  p.Title,
		Fields:        fl.Fields,
	}

	cp := cachePath(repo)
	os.MkdirAll(filepath.Dir(cp), 0755)
	b, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(cp, b, 0644); err != nil {
		return nil, err
	}
	fmt.Printf("Saved: %s\n", cp)
	return config, nil
}

func deepGet(m map[string]any, keys ...string) any {
	var v any = m
	for _, k := range keys {
		mm, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = mm[k]
	}
	return v
}
