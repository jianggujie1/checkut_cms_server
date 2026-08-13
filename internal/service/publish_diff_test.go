package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"checkut-cms-server/internal/model"
)

func ts(t time.Time) *time.Time { return &t }

func TestClassify_firstPublish_allActiveCreated(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		row  rowInfo
		want string
	}{
		{"active", rowInfo{ID: "1", Status: model.StatusPublished, CreatedAt: ts(now), UpdatedAt: ts(now)}, "created"},
		{"archived on first publish is not pushed", rowInfo{ID: "1", Status: model.StatusArchived, DeletedAt: ts(now)}, ""},
		{"deleted not previously synced is ignored", rowInfo{ID: "1", Status: model.StatusArchived, DeletedAt: ts(now), CreatedAt: ts(now)}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classify(c.row, nil, map[string]bool{}); got != c.want {
				t.Fatalf("classify = %q, want %q", got, c.want)
			}
		})
	}
}

func TestClassify_afterSync_created_updated_deleted(t *testing.T) {
	lastSync := time.Now().Add(-24 * time.Hour)
	now := time.Now()

	t.Run("created after last sync", func(t *testing.T) {
		r := rowInfo{ID: "1", Status: model.StatusPublished, CreatedAt: ts(now), UpdatedAt: ts(now)}
		if got := classify(r, &lastSync, nil); got != "created" {
			t.Fatalf("got %q want created", got)
		}
	})
	t.Run("updated after last sync", func(t *testing.T) {
		r := rowInfo{ID: "1", Status: model.StatusPublished, CreatedAt: ts(lastSync.Add(-1 * time.Hour)), UpdatedAt: ts(now)}
		if got := classify(r, &lastSync, nil); got != "updated" {
			t.Fatalf("got %q want updated", got)
		}
	})
	t.Run("unchanged is empty", func(t *testing.T) {
		r := rowInfo{ID: "1", Status: model.StatusPublished, CreatedAt: ts(lastSync.Add(-2 * time.Hour)), UpdatedAt: ts(lastSync.Add(-1 * time.Hour))}
		if got := classify(r, &lastSync, nil); got != "" {
			t.Fatalf("got %q want empty", got)
		}
	})
	t.Run("deleted online -> deleted", func(t *testing.T) {
		r := rowInfo{ID: "1", Status: model.StatusArchived, DeletedAt: ts(now), CreatedAt: ts(lastSync.Add(-2 * time.Hour))}
		if got := classify(r, &lastSync, map[string]bool{"1": true}); got != "deleted" {
			t.Fatalf("got %q want deleted", got)
		}
	})
	t.Run("deleted, previously synced (created before lastSync) -> deleted", func(t *testing.T) {
		r := rowInfo{ID: "x", Status: model.StatusArchived, DeletedAt: ts(now), CreatedAt: ts(lastSync.Add(-2 * time.Hour))}
		if got := classify(r, &lastSync, map[string]bool{}); got != "deleted" {
			t.Fatalf("got %q want deleted", got)
		}
	})
	t.Run("deleted but created after lastSync and not online -> no diff", func(t *testing.T) {
		r := rowInfo{ID: "x", Status: model.StatusArchived, DeletedAt: ts(now), CreatedAt: ts(lastSync.Add(2 * time.Hour))}
		if got := classify(r, &lastSync, map[string]bool{}); got != "" {
			t.Fatalf("got %q want empty", got)
		}
	})
}

func TestDiffGroup_splitsGroups(t *testing.T) {
	lastSync := time.Now().Add(-24 * time.Hour)
	old := time.Now().Add(-48 * time.Hour)
	rows := []rowInfo{
		{ID: "c", Status: model.StatusPublished, CreatedAt: ts(time.Now())},
		{ID: "u", Status: model.StatusPublished, CreatedAt: ts(old), UpdatedAt: ts(time.Now())},
		{ID: "d", Status: model.StatusArchived, DeletedAt: ts(time.Now()), CreatedAt: ts(old)},
		{ID: "unchanged", Status: model.StatusPublished, CreatedAt: ts(old)},
	}
	g := diffGroup(rows, &lastSync, map[string]bool{"d": true})
	if len(g.Created) != 1 || g.Created[0].ID != "c" {
		t.Fatalf("created = %+v", g.Created)
	}
	if len(g.Updated) != 1 || g.Updated[0].ID != "u" {
		t.Fatalf("updated = %+v", g.Updated)
	}
	if len(g.Deleted) != 1 || g.Deleted[0].ID != "d" {
		t.Fatalf("deleted = %+v", g.Deleted)
	}
}
func TestSupaPayload_OmitsStatus(t *testing.T) {
	dests := []*model.Destination{{ID: "d1", Title: "Dest 1", Status: model.StatusPublished}}
	b, err := json.Marshal(toSupaDestinations(dests))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"status"`) {
		t.Fatalf("supaDestination json contains status: %s", string(b))
	}

	attrs := []*model.Attraction{{ID: "a1", Title: "Attr 1", Status: model.StatusPublished}}
	b, err = json.Marshal(toSupaAttractions(attrs))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"status"`) {
		t.Fatalf("supaAttraction json contains status: %s", string(b))
	}

	its := []*model.Itinerary{{ID: "i1", Title: "It 1", Status: model.StatusPublished}}
	b, err = json.Marshal(toSupaItineraries(its))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"status"`) {
		t.Fatalf("supaItinerary json contains status: %s", string(b))
	}

	days := []*model.ItineraryDay{{ID: "day1", Title: strPtr("Day 1"), Status: model.StatusPublished}}
	b, err = json.Marshal(toSupaDays(days))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"status"`) {
		t.Fatalf("supaItineraryDay json contains status: %s", string(b))
	}

	acts := []*model.ItineraryActivity{{ID: "act1", Title: "Act 1", Status: model.StatusPublished}}
	b, err = json.Marshal(toSupaActs(acts))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"status"`) {
		t.Fatalf("supaItineraryActivity json contains status: %s", string(b))
	}
}

func strPtr(s string) *string { return &s }
