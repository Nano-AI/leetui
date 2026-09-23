package tui

import (
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestAuditPackReplacesPlan(t *testing.T) {
	m := newTestModel(t, false)
	m.plan = planSel{Slug: "plan", Name: "Old plan"}
	next, _ := m.applyPack(store.Company{Slug: "google", Name: "Google"}, leetcode.SixMonths, 1)
	m = next.(Model)
	if m.plan.Active() || m.lastColLabel() != "asked by" {
		t.Fatal("company pack retained study-plan badge and chapter column")
	}
}

func TestAuditTodoFilterCanEscape(t *testing.T) {
	m := newTestModel(t, false)
	m.filter.TodoOnly = true
	next, _ := m.Update(key("esc"))
	if next.(Model).filter.TodoOnly {
		t.Fatal("esc cannot clear My List")
	}
}

func TestAuditArrowAndPageKeysScrollHelp(t *testing.T) {
	for _, k := range []tea.KeyMsg{key("down"), {Type: tea.KeyPgDown}} {
		m := newTestModel(t, false)
		m.width, m.height, m.mode = 80, 24, modeHelp
		m.rows = []store.Row{{Slug: "a"}, {Slug: "b"}}
		next, _ := m.Update(k)
		m = next.(Model)
		if m.helpScroll == 0 || m.cursor != 0 {
			t.Fatalf("%s moved board instead of help: scroll=%d cursor=%d", k, m.helpScroll, m.cursor)
		}
	}
}

func TestAuditArrowScrollsStatement(t *testing.T) {
	m := newTestModel(t, false)
	m.width, m.height, m.mode, m.focus = 80, 24, modeSolve, paneDetail
	m.rows = []store.Row{{Slug: "a"}, {Slug: "b"}}
	m.detailMD = strings.Repeat("statement\n", 100)
	m.detail = &store.Detail{}
	m.detail.Slug = "a"
	next, _ := m.Update(key("down"))
	m = next.(Model)
	if m.detailScroll != 1 || m.cursor != 0 {
		t.Fatalf("arrow changed problem: scroll=%d cursor=%d", m.detailScroll, m.cursor)
	}
}

func TestAuditAnnotationFiltersRestoreCuratedOrder(t *testing.T) {
	for _, base := range []string{"plan", "contest", "frequency"} {
		m := newTestModel(t, false)
		m.filter.Sort = base
		switch base {
		case "plan":
			m.filter.Plan = "plan"
		case "contest":
			m.filter.Contest = "contest"
		case "frequency":
			m.filter.Companies = []string{"google"}
		}
		for i := 0; i < 3; i++ {
			next, _ := m.cycleMarkFilter()
			m = next.(Model)
		}
		if m.filter.Sort != base {
			t.Errorf("mark filter lost %s order: %q", base, m.filter.Sort)
		}
		m.filter.Sort = base
		for i := 0; i < 2; i++ {
			next, _ := m.toggleTodoFilter()
			m = next.(Model)
		}
		if m.filter.Sort != base {
			t.Errorf("todo filter lost %s order: %q", base, m.filter.Sort)
		}
	}
}

func TestAuditRegistryRefreshClampsFilteredSelection(t *testing.T) {
	m := newTestModel(t, false)
	m.planFilter.SetValue("match")
	m.planIdx = 1
	next, _ := m.Update(plansMsg{plans: []store.Plan{{Slug: "match", Name: "match"}, {Slug: "other"}}})
	m = next.(Model)
	if m.planIdx != 0 {
		t.Fatal("plan selection points outside filtered results")
	}
	m.contestFilter.SetValue("match")
	m.contestIdx = 1
	next, _ = m.Update(contestsMsg{contests: []store.Contest{{Slug: "match"}, {Slug: "other"}}})
	if next.(Model).contestIdx != 0 {
		t.Fatal("contest selection points outside filtered results")
	}
}

func TestAuditCuratedBrowsersFitNarrowTerminal(t *testing.T) {
	m := newTestModel(t, false)
	m.width, m.height = 40, 24
	for _, mode := range []mode{modeCompany, modePlan, modeContest} {
		m.mode = mode
		for i, line := range strings.Split(m.View(), "\n") {
			if w := lipgloss.Width(line); w > m.width {
				t.Errorf("mode %v line %d: %d columns exceeds %d", mode, i, w, m.width)
				break
			}
		}
	}
}
