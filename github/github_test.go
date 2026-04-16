package github

import (
	"os/exec"
	"strings"
	"testing"
)

// These tests require `gh` CLI to be installed and authenticated.
// They are integration tests that verify real `gh` commands work.

func skipIfNoGh(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh CLI not installed, skipping integration test")
	}
	// Check if authenticated
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		t.Skip("gh CLI not authenticated, skipping integration test")
	}
}

func TestGetUser(t *testing.T) {
	skipIfNoGh(t)

	c := NewClient()
	login, err := c.GetUser()
	if err != nil {
		t.Fatalf("GetUser() error: %v", err)
	}
	if login == "" {
		t.Fatal("GetUser() returned empty login")
	}
	t.Logf("Authenticated as: %s", login)
}

func TestListForks(t *testing.T) {
	skipIfNoGh(t)

	c := NewClient()
	login, err := c.GetUser()
	if err != nil {
		t.Fatalf("GetUser() error: %v", err)
	}

	forks, err := c.ListForks(login)
	if err != nil {
		t.Fatalf("ListForks() error: %v", err)
	}

	t.Logf("Found %d fork(s)", len(forks))
	for _, f := range forks {
		if f.FullName == "" {
			t.Error("fork has empty FullName")
		}
		if f.Name == "" {
			t.Error("fork has empty Name")
		}
		if f.Owner.Login == "" {
			t.Error("fork has empty Owner.Login")
		}
		// Verify it's actually a fork or came from --fork filter
		parts := strings.SplitN(f.FullName, "/", 2)
		if len(parts) != 2 {
			t.Errorf("unexpected FullName format: %s", f.FullName)
		}
	}
}

func TestListForksForOwner(t *testing.T) {
	skipIfNoGh(t)

	c := NewClient()
	login, err := c.GetUser()
	if err != nil {
		t.Fatalf("GetUser() error: %v", err)
	}

	forks, err := c.listForksForOwner(login)
	if err != nil {
		t.Fatalf("listForksForOwner() error: %v", err)
	}

	t.Logf("Found %d fork(s) for %s", len(forks), login)
	for _, f := range forks {
		if f.Owner.Login != login {
			t.Errorf("expected owner %s, got %s for %s", login, f.Owner.Login, f.FullName)
		}
	}
}

func TestListOrgs(t *testing.T) {
	skipIfNoGh(t)

	c := NewClient()
	orgs, err := c.listOrgs()
	if err != nil {
		t.Fatalf("listOrgs() error: %v", err)
	}

	t.Logf("Found %d org(s)", len(orgs))
	for _, org := range orgs {
		if org == "" {
			t.Error("empty org name")
		}
	}
}

func TestDeleteRepoNonExistent(t *testing.T) {
	skipIfNoGh(t)

	c := NewClient()
	login, err := c.GetUser()
	if err != nil {
		t.Fatalf("GetUser() error: %v", err)
	}

	// Attempt to delete a repo that doesn't exist — should fail
	err = c.DeleteRepo(login, "this-repo-definitely-does-not-exist-12345")
	if err == nil {
		t.Error("expected error when deleting non-existent repo")
	}
}
