package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Client struct{}

type Repo struct {
	FullName    string `json:"nameWithOwner"`
	Name        string `json:"name"`
	HTMLURL     string `json:"url"`
	Fork        bool   `json:"isFork"`
	Description string `json:"description"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
	StargazersCount int `json:"stargazerCount"`
	Language        *struct {
		Name string `json:"name"`
	} `json:"primaryLanguage"`
	Parent *struct {
		FullName string `json:"nameWithOwner"`
	} `json:"parent"`
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) GetUser() (string, error) {
	out, err := exec.Command("gh", "api", "user", "-q", ".login").Output()
	if err != nil {
		return "", fmt.Errorf("gh api user: %w", err)
	}
	login := strings.TrimSpace(string(out))
	if login == "" {
		return "", fmt.Errorf("empty login")
	}
	return login, nil
}

func (c *Client) ListForks(username string) ([]Repo, error) {
	var allForks []Repo

	// List user's own forks
	userForks, err := c.listForksForOwner(username)
	if err != nil {
		return nil, fmt.Errorf("listing user forks: %w", err)
	}
	allForks = append(allForks, userForks...)

	// List orgs the user belongs to and get forks from each
	orgs, err := c.listOrgs()
	if err != nil {
		return nil, fmt.Errorf("listing orgs: %w", err)
	}
	for _, org := range orgs {
		orgForks, err := c.listForksForOwner(org)
		if err != nil {
			continue // skip orgs we can't list
		}
		allForks = append(allForks, orgForks...)
	}

	return allForks, nil
}

func (c *Client) listForksForOwner(owner string) ([]Repo, error) {
	fields := "name,nameWithOwner,url,isFork,description,stargazerCount,primaryLanguage,parent"
	out, err := exec.Command("gh", "repo", "list", owner,
		"--fork", "--limit", "1000",
		"--json", fields,
	).Output()
	if err != nil {
		// Retry without parent field — it fails on SAML-protected orgs
		fields = "name,nameWithOwner,url,isFork,description,stargazerCount,primaryLanguage"
		out, err = exec.Command("gh", "repo", "list", owner,
			"--fork", "--limit", "1000",
			"--json", fields,
		).Output()
		if err != nil {
			return nil, fmt.Errorf("gh repo list %s: %w", owner, err)
		}
	}

	var repos []Repo
	if err := json.Unmarshal(out, &repos); err != nil {
		return nil, fmt.Errorf("parsing repos for %s: %w", owner, err)
	}

	// Fill in owner field from the full name
	for i := range repos {
		parts := strings.SplitN(repos[i].FullName, "/", 2)
		if len(parts) == 2 {
			repos[i].Owner.Login = parts[0]
		}
	}

	return repos, nil
}

func (c *Client) listOrgs() ([]string, error) {
	out, err := exec.Command("gh", "api", "user/orgs", "-q", ".[].login").Output()
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

func (c *Client) DeleteRepo(owner, name string) error {
	fullName := owner + "/" + name
	out, err := exec.Command("gh", "repo", "delete", fullName, "--yes").CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh repo delete %s: %s", fullName, strings.TrimSpace(string(out)))
	}
	return nil
}
