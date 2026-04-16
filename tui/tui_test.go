package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	gh "github.com/suheybbecerek/delete-forks/github"
)

func testModel(forks []gh.Repo) Model {
	m := NewModel(&gh.Client{}, "testuser")
	m.width = 80
	m.height = 24
	// Simulate forks loaded
	m.phase = phaseList
	m.forks = make([]forkItem, len(forks))
	for i, f := range forks {
		m.forks[i] = forkItem{repo: f, state: forkPending}
	}
	return m
}

func makeForks(n int) []gh.Repo {
	forks := make([]gh.Repo, n)
	for i := range forks {
		name := fmt.Sprintf("repo-%d", i)
		forks[i] = gh.Repo{
			FullName: fmt.Sprintf("testuser/%s", name),
			Name:     name,
		}
		forks[i].Owner.Login = "testuser"
	}
	return forks
}

func TestNewModelStartsInLoadingPhase(t *testing.T) {
	m := NewModel(&gh.Client{}, "testuser")
	if m.phase != phaseLoading {
		t.Errorf("expected phaseLoading, got %d", m.phase)
	}
	if m.username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", m.username)
	}
}

func TestForksLoadedTransitionsToList(t *testing.T) {
	m := NewModel(&gh.Client{}, "testuser")
	m.width = 80
	m.height = 24

	forks := makeForks(3)
	updated, _ := m.Update(forksLoadedMsg{forks: forks})
	model := updated.(Model)

	if model.phase != phaseList {
		t.Errorf("expected phaseList, got %d", model.phase)
	}
	if len(model.forks) != 3 {
		t.Errorf("expected 3 forks, got %d", len(model.forks))
	}
}

func TestForksErrorShowsError(t *testing.T) {
	m := NewModel(&gh.Client{}, "testuser")
	m.width = 80
	m.height = 24

	updated, _ := m.Update(forksErrorMsg{err: fmt.Errorf("api failure")})
	model := updated.(Model)

	if model.phase != phaseList {
		t.Errorf("expected phaseList, got %d", model.phase)
	}
	if model.err == nil {
		t.Error("expected error to be set")
	}
}

func TestNavigationKeys(t *testing.T) {
	m := testModel(makeForks(5))

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model := updated.(Model)
	if model.cursor != 1 {
		t.Errorf("expected cursor 1 after j, got %d", model.cursor)
	}

	// Move down again
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(Model)
	if model.cursor != 2 {
		t.Errorf("expected cursor 2 after j, got %d", model.cursor)
	}

	// Move up
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(Model)
	if model.cursor != 1 {
		t.Errorf("expected cursor 1 after k, got %d", model.cursor)
	}

	// Can't go above 0
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(Model)
	if model.cursor != 0 {
		t.Errorf("expected cursor 0 at top, got %d", model.cursor)
	}
}

func TestSpaceTogglesSelection(t *testing.T) {
	m := testModel(makeForks(3))

	// Select item 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	model := updated.(Model)
	if !model.selected[0] {
		t.Error("expected item 0 to be selected")
	}

	// Deselect item 0
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	model = updated.(Model)
	if model.selected[0] {
		t.Error("expected item 0 to be deselected")
	}
}

func TestSelectAll(t *testing.T) {
	m := testModel(makeForks(5))

	// Select all
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model := updated.(Model)
	for i := 0; i < 5; i++ {
		if !model.selected[i] {
			t.Errorf("expected item %d to be selected", i)
		}
	}

	// Deselect all
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)
	if len(model.selected) != 0 {
		t.Errorf("expected all deselected, got %d selected", len(model.selected))
	}
}

func TestEnterWithNoSelectionDoesNothing(t *testing.T) {
	m := testModel(makeForks(3))

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.phase != phaseList {
		t.Errorf("expected to stay in phaseList, got %d", model.phase)
	}
}

func TestEnterWithSelectionGoesToConfirm(t *testing.T) {
	m := testModel(makeForks(3))
	m.selected[0] = true

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.phase != phaseConfirm {
		t.Errorf("expected phaseConfirm, got %d", model.phase)
	}
}

func TestConfirmNoGoesBackToList(t *testing.T) {
	m := testModel(makeForks(3))
	m.selected[0] = true
	m.phase = phaseConfirm

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model := updated.(Model)
	if model.phase != phaseList {
		t.Errorf("expected phaseList after cancel, got %d", model.phase)
	}
}

func TestConfirmYesGoesToDeleting(t *testing.T) {
	m := testModel(makeForks(3))
	m.selected[0] = true
	m.phase = phaseConfirm

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model := updated.(Model)
	if model.phase != phaseDeleting {
		t.Errorf("expected phaseDeleting, got %d", model.phase)
	}
}

func TestDeletedMsgMarksRepoDeleted(t *testing.T) {
	m := testModel(makeForks(3))
	m.selected[0] = true
	m.selected[1] = true
	m.phase = phaseDeleting
	m.deleteTotal = 2
	m.forks[0].state = forkDeleting

	updated, _ := m.Update(forkDeletedMsg{index: 0, err: nil})
	model := updated.(Model)
	if model.forks[0].state != forkDeleted {
		t.Errorf("expected forkDeleted, got %d", model.forks[0].state)
	}
}

func TestDeletedMsgWithErrorMarksFailed(t *testing.T) {
	m := testModel(makeForks(3))
	m.selected[0] = true
	m.phase = phaseDeleting
	m.deleteTotal = 1
	m.forks[0].state = forkDeleting

	updated, _ := m.Update(forkDeletedMsg{index: 0, err: fmt.Errorf("forbidden")})
	model := updated.(Model)
	if model.forks[0].state != forkFailed {
		t.Errorf("expected forkFailed, got %d", model.forks[0].state)
	}
	if model.deleteErrs != 1 {
		t.Errorf("expected 1 error, got %d", model.deleteErrs)
	}
}

func TestAllDeletedMsgTransitionsToDone(t *testing.T) {
	m := testModel(makeForks(1))
	m.phase = phaseDeleting

	updated, _ := m.Update(allDeletedMsg{})
	model := updated.(Model)
	if model.phase != phaseDone {
		t.Errorf("expected phaseDone, got %d", model.phase)
	}
}

func TestSelectedIndicesAreSorted(t *testing.T) {
	m := testModel(makeForks(5))
	m.selected[4] = true
	m.selected[1] = true
	m.selected[3] = true

	indices := m.selectedIndices()
	if len(indices) != 3 {
		t.Fatalf("expected 3 indices, got %d", len(indices))
	}
	if indices[0] != 1 || indices[1] != 3 || indices[2] != 4 {
		t.Errorf("expected [1, 3, 4], got %v", indices)
	}
}

func TestViewListWithManyForks(t *testing.T) {
	m := testModel(makeForks(50))
	m.height = 20

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	// Should contain "showing" since 50 > visible lines
	if !contains(view, "showing") {
		t.Error("expected 'showing' indicator for scrolling")
	}
}

func TestViewConfirmWithManyForks(t *testing.T) {
	m := testModel(makeForks(50))
	m.height = 20
	for i := range m.forks {
		m.selected[i] = true
	}
	m.phase = phaseConfirm

	view := m.View()
	if !contains(view, "... and") {
		t.Error("expected truncation indicator in confirm view")
	}
}

func TestViewDeletingWithManyForks(t *testing.T) {
	m := testModel(makeForks(50))
	m.height = 20
	for i := range m.forks {
		m.selected[i] = true
	}
	m.phase = phaseDeleting
	m.deleteTotal = 50

	view := m.View()
	if !contains(view, "showing") {
		t.Error("expected 'showing' indicator in deleting view")
	}
}

func TestViewEmpty(t *testing.T) {
	m := testModel(nil)
	view := m.View()
	if !contains(view, "No forks found") {
		t.Error("expected 'No forks found' message")
	}
}

func TestViewError(t *testing.T) {
	m := testModel(nil)
	m.err = fmt.Errorf("something broke")
	view := m.View()
	if !contains(view, "something broke") {
		t.Error("expected error message in view")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
