// Header status pills and group header counts must describe the list the user
// is actually looking at.
//
// rebuildFlatItems partitions archived vs active, but the counters were derived
// from the unfiltered instance set: getStatusCounts iterated the whole render
// snapshot and group stats used len(g.Sessions). After a bulk archive (172
// sessions, 2026-08-13) the deck rendered ~68 visible error rows under a header
// pill reading "✕ 267 error", with group headers like "NewChio (64)" above 24
// visible rows — which reads as "the archive silently did nothing".

package ui

import (
	"testing"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// buildDeck wires instances into a Home the way the render path expects.
func buildDeck(t *testing.T, instances []*session.Instance) *Home {
	t.Helper()
	home := NewHome()
	home.width, home.height = 120, 40
	home.initialLoading = false
	home.instancesMu.Lock()
	home.instances = instances
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(instances)
	home.refreshSessionRenderSnapshot(instances)
	home.rebuildFlatItems()
	return home
}

func TestStatusCounts_ExcludeArchivedSessions(t *testing.T) {
	var instances []*session.Instance
	// 2 visible errors + 5 archived errors: the pill must read 2, not 7.
	for i, spec := range []struct {
		name     string
		archived bool
	}{
		{"live-1", false}, {"live-2", false},
		{"gone-1", true}, {"gone-2", true}, {"gone-3", true}, {"gone-4", true}, {"gone-5", true},
	} {
		in := session.NewInstanceWithTool(spec.name, "/tmp/x", "claude")
		in.GroupPath = "alpha"
		in.Status = session.StatusError
		if spec.archived {
			in.ArchivedAt = time.Now().Add(-time.Duration(i) * time.Minute)
		}
		instances = append(instances, in)
	}

	home := buildDeck(t, instances)
	_, _, _, _, errored := home.countSessionStatuses()
	if errored != 2 {
		t.Errorf("header error pill = %d, want 2 (5 archived sessions must not be counted); "+
			"a bulk archive would otherwise still read as a deck full of errors", errored)
	}
}

func TestGroupHeaderCount_ExcludesArchived_AndFollowsArchivedView(t *testing.T) {
	var instances []*session.Instance
	for i, archived := range []bool{false, false, false, true, true, true, true} {
		in := session.NewInstanceWithTool("s", "/tmp/x", "claude")
		in.GroupPath = "alpha"
		in.Status = session.StatusIdle
		if archived {
			in.ArchivedAt = time.Now().Add(-time.Duration(i) * time.Minute)
		}
		instances = append(instances, in)
	}

	// Active view: header counts the 3 visible sessions, not all 7.
	home := buildDeck(t, instances)
	stats := home.buildGroupRenderStats(home.getSessionRenderSnapshot())
	if got := stats["alpha"].sessionCount; got != 3 {
		t.Errorf("active view: group header count = %d, want 3 (4 archived excluded)", got)
	}

	// Archived view (^): the same header should describe the archived list.
	home.statusFilter = FilterModeArchived
	stats = home.buildGroupRenderStats(home.getSessionRenderSnapshot())
	if got := stats["alpha"].sessionCount; got != 4 {
		t.Errorf("archived view: group header count = %d, want 4 (the archived sessions)", got)
	}
}
