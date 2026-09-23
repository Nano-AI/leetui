package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestAuditBoardNavigationIgnoresHiddenDetailFocus(t *testing.T) {
	for _, k := range []tea.KeyMsg{key("down"), key("j"), {Type: tea.KeyPgDown}} {
		m := newTestModel(t, false)
		m.width, m.height, m.mode, m.focus = 80, 24, modeBoard, paneDetail
		m.rows = []store.Row{{Slug: "a"}, {Slug: "b"}}
		next, _ := m.Update(k)
		if next.(Model).cursor != 1 {
			t.Errorf("%s scrolled the hidden pane instead of the board", k)
		}
	}
}

func TestAuditHomeEndScrollStatement(t *testing.T) {
	m := newTestModel(t, false)
	m.width, m.height, m.mode, m.focus = 80, 24, modeSolve, paneDetail
	m.rows = []store.Row{{Slug: "a"}, {Slug: "b"}}
	m.detailMD = strings.Repeat("body\n", 100)
	m.detail = &store.Detail{}
	m.detail.Slug = "a"
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = next.(Model)
	if m.cursor != 0 || m.detailScroll != m.detailMaxScroll() {
		t.Fatal("end moved the problem cursor instead of scrolling the statement")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if next.(Model).detailScroll != 0 {
		t.Fatal("home did not return to the start of the statement")
	}
}

func TestAuditEditorialRejectsOldResize(t *testing.T) {
	m := newTestModel(t, true)
	m.rows = []store.Row{{Slug: "two-sum"}}
	cacheEditorial(t, m, "two-sum", &leetcode.Solution{
		ID: "7", CanSeeDetail: true, Content: strings.Repeat("A long paragraph. ", 30),
	})
	m.width = 120
	old := m.fetchEditorial("two-sum")()
	m.width = 80
	fresh := m.fetchEditorial("two-sum")()
	next, _ := m.Update(fresh)
	m = next.(Model)
	want := m.editorialMD
	next, _ = m.Update(old)
	if next.(Model).editorialMD != want {
		t.Fatal("late wide editorial replaced the narrow render")
	}
	stale := old.(editorialMsg)
	stale.err, stale.editorial = errors.New("old request failed"), nil
	next, cmd := m.Update(stale)
	if cmd != nil || next.(Model).editorialMD != want {
		t.Fatal("stale editorial error cleared the current reading or raised a notice")
	}
}

func TestAuditArrowScrollsEditorialWithoutChangingProblem(t *testing.T) {
	m := newTestModel(t, false)
	m.width, m.height, m.mode, m.focus = 80, 24, modeSolve, paneDetail
	m.rows = []store.Row{{Slug: "a"}, {Slug: "b"}}
	m.showEditorial = true
	m.editorial = &store.Editorial{Slug: "a"}
	m.editorialMD = strings.Repeat("editorial\n", 100)
	next, _ := m.Update(key("down"))
	m = next.(Model)
	if m.currentSlug() != "a" || m.detailScroll != 1 || m.editorial == nil {
		t.Fatal("arrow navigation changed the problem instead of scrolling its editorial")
	}
}

func TestAuditNarrowCuratedRowsKeepStatus(t *testing.T) {
	m := newTestModel(t, false)
	m.width, m.height = 40, 24
	m.mode = modePlan
	m.plans = []store.Plan{{Name: strings.Repeat("Long plan ", 8), PremiumOnly: true}}
	if !strings.Contains(m.View(), "premium") {
		t.Error("long plan name clipped its premium gate")
	}
	m.plans[0].Name = strings.Repeat("算法", 30)
	if !strings.Contains(m.View(), "premium") {
		t.Error("wide-glyph plan name clipped its premium gate")
	}
	m.mode = modeContest
	m.contests = []store.Contest{{Title: strings.Repeat("Long contest ", 8),
		StartTime: time.Now().Add(-time.Minute).Unix(), Duration: 3600}}
	if !strings.Contains(m.View(), "left") {
		t.Error("long contest title clipped its live countdown")
	}
	m.contests[0].Title = strings.Repeat("竞赛", 30)
	if !strings.Contains(m.View(), "left") {
		t.Error("wide-glyph contest title clipped its live countdown")
	}
}
