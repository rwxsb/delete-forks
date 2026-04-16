package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gh "github.com/suheybbecerek/delete-forks/github"
	"github.com/suheybbecerek/delete-forks/tui"
)

func main() {
	if !ghInstalled() {
		fmt.Println(errorText("  'gh' CLI is not installed."))
		fmt.Println("  Install it: " + linkText("https://cli.github.com"))
		os.Exit(1)
	}

	// Check if logged in, if not, launch gh auth login
	if _, err := ghAuthToken(); err != nil {
		fmt.Println(boldText("🔐 GitHub CLI login required"))
		fmt.Println()
		fmt.Println(dimText("  Launching 'gh auth login'..."))
		fmt.Println()
		if err := ghLogin(); err != nil {
			fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
			os.Exit(1)
		}
	}

	// Ensure delete_repo scope is granted
	if err := ghEnsureScopes(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	client := gh.NewClient()

	user, err := client.GetUser()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get user: %v\n", err)
		os.Exit(1)
	}

	model := tui.NewModel(client, user)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ghInstalled checks if the gh CLI is available on PATH.
func ghInstalled() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// ghAuthToken retrieves the current token from gh CLI.
func ghAuthToken() (string, error) {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("empty token")
	}
	return token, nil
}

// ghLogin runs interactive gh auth login with the delete_repo scope.
func ghLogin() error {
	cmd := exec.Command("gh", "auth", "login", "--scopes", "repo,delete_repo")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ghEnsureScopes checks that delete_repo scope is present, refreshes if not.
func ghEnsureScopes() error {
	out, err := exec.Command("gh", "auth", "status").CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh auth status failed: %w", err)
	}
	if !strings.Contains(string(out), "delete_repo") {
		fmt.Println(dimText("  Requesting delete_repo scope..."))
		cmd := exec.Command("gh", "auth", "refresh", "--scopes", "repo,delete_repo")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add delete_repo scope: %w", err)
		}
	}
	return nil
}

func errorText(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render(s)
}

func boldText(s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF6B6B")).Render(s)
}

func dimText(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(s)
}

func linkText(s string) string {
	return lipgloss.NewStyle().Bold(true).Underline(true).Render(s)
}

func codeText(s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575")).Render(s)
}

func successText(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render(s)
}
