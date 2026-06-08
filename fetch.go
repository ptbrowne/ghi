package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdFetch(repo string, numbers []int) error {
	config, err := loadCache(repo)
	if err != nil {
		return err
	}

	dir := issuesPath(repo)
	if err := ensureIssuesRepo(dir); err != nil {
		return err
	}

	var written []string
	var nums []string
	for _, number := range numbers {
		fmt.Printf("Fetching #%d...\n", number)
		issue, err := ghGetIssue(repo, number)
		if err != nil {
			return err
		}

		pf, err := fetchProjectFields(issue.NodeID, config.ProjectID)
		if err != nil {
			return err
		}

		labels := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			labels[i] = l.Name
		}

		meta := IssueMeta{
			Number:     issue.Number,
			Title:      issue.Title,
			State:      issue.State,
			Labels:     labels,
			Project:    pf,
			GithubSha1: bodySHA1(issue.Body),
		}
		if issue.Milestone != nil {
			meta.Milestone = issue.Milestone.Title
		}

		name := fmt.Sprintf("%d-%s.md", number, slugify(issue.Title))
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(writeFM(meta, issue.Body)), 0644); err != nil {
			return err
		}
		written = append(written, path)
		nums = append(nums, fmt.Sprintf("#%d", number))
		fmt.Printf("  → %s\n", path)
	}

	if err := gitStageIn(dir, written...); err != nil {
		return err
	}
	return gitCommitIn(dir, "fetch: "+strings.Join(nums, ", "))
}

func fetchProjectFields(nodeID, projectID string) (*ProjectMeta, error) {
	data, err := ghGraphQL(fmt.Sprintf(`{
      node(id: "%s") {
        ... on Issue {
          projectItems(first: 5) {
            nodes {
              id
              project { id }
              fieldValues(first: 20) {
                nodes {
                  ... on ProjectV2ItemFieldSingleSelectValue {
                    field { ... on ProjectV2SingleSelectField { name } }
                    name
                    optionId
                  }
                  ... on ProjectV2ItemFieldIterationValue {
                    field { ... on ProjectV2IterationField { name } }
                    title
                    iterationId
                  }
                }
              }
            }
          }
        }
      }
    }`, nodeID))
	if err != nil {
		return nil, err
	}

	nodes, _ := deepGet(data, "data", "node", "projectItems", "nodes").([]any)
	for _, rawItem := range nodes {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		proj, _ := item["project"].(map[string]any)
		if proj["id"] != projectID {
			continue
		}
		itemID, ok := item["id"].(string)
		if !ok {
			continue
		}
		pm := &ProjectMeta{ItemID: itemID}
		fvNodes, _ := deepGet(item, "fieldValues", "nodes").([]any)
		for _, rawFV := range fvNodes {
			fv, _ := rawFV.(map[string]any)
			if len(fv) == 0 {
				continue
			}
			field, _ := fv["field"].(map[string]any)
			name, _ := field["name"].(string)
			switch name {
			case "Status":
				pm.Status, _ = fv["name"].(string)
			case "Project Priority":
				pm.Priority, _ = fv["name"].(string)
			case "Sprint":
				pm.Sprint, _ = fv["title"].(string)
			}
		}
		return pm, nil
	}
	return nil, nil
}
