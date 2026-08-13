package repository

import (
	"testing"

	"checkut-cms-server/internal/model"
)

func day(id string, n int, acts ...*model.ItineraryActivity) *model.ItineraryDayWithActivities {
	return &model.ItineraryDayWithActivities{
		ItineraryDay: model.ItineraryDay{ID: id, DayNumber: int32(n)},
		Activities:   acts,
	}
}

func act(id, dayID string) *model.ItineraryActivity {
	return &model.ItineraryActivity{ID: id, DayID: dayID}
}

func TestReconcileTree_inserts_updates_deletes(t *testing.T) {
	current := &model.ItineraryWithTree{
		Itinerary: model.Itinerary{ID: "it1"},
		Days: []*model.ItineraryDayWithActivities{
			day("d1", 1, act("a1", "d1"), act("a2", "d1")),
			day("d2", 2, act("a3", "d2")),
		},
	}

	incoming := &model.ItineraryWithTree{
		Itinerary: model.Itinerary{ID: "it1"},
		Days: []*model.ItineraryDayWithActivities{
			// d1 keeps a1, a2 removed -> deleted; d1 updated
			day("d1", 1, act("a1", "d1"), act("", "")), // new activity
			// new day d3 with no id
			day("", 2, act("", "")),
		},
	}

	plan := ReconcileTree(current, incoming)

	// day_number renumbered 1-based
	if got := plan.Days[0].DayNumber; got != 1 {
		t.Fatalf("first day day_number = %d, want 1", got)
	}
	if got := plan.Days[1].DayNumber; got != 2 {
		t.Fatalf("second day day_number = %d, want 2", got)
	}
	// d2 dropped -> delete
	if len(plan.DaysToDelete) != 1 || plan.DaysToDelete[0] != "d2" {
		t.Fatalf("DaysToDelete = %v, want [d2]", plan.DaysToDelete)
	}
	// a2 dropped -> delete; a3 dropped because its day d2 is dropped -> delete
	if len(plan.ActivitiesToDelete) != 2 ||
		(plan.ActivitiesToDelete[0] != "a2" && plan.ActivitiesToDelete[1] != "a2") ||
		(plan.ActivitiesToDelete[0] != "a3" && plan.ActivitiesToDelete[1] != "a3") {
		t.Fatalf("ActivitiesToDelete = %v, want [a2 a3]", plan.ActivitiesToDelete)
	}
	// 1 new day
	if len(plan.DaysToInsert) != 1 {
		t.Fatalf("DaysToInsert len = %d, want 1", len(plan.DaysToInsert))
	}
	// 2 new activities (one in d1, one in new day)
	if len(plan.ActivitiesToInsert) != 2 {
		t.Fatalf("ActivitiesToInsert len = %d, want 2", len(plan.ActivitiesToInsert))
	}
	// counters: 2 days, 3 activities (d1 has a1+new, new day has 1)
	if *plan.TotalDays != "2" || *plan.ActivitiesCount != "3" {
		t.Fatalf("counters = days=%s acts=%s, want 2/3", *plan.TotalDays, *plan.ActivitiesCount)
	}
	// new day got an id
	if plan.DaysToInsert[0].ID == "" {
		t.Fatal("new day must receive an id")
	}
	// new activities point at their (final) day
	for _, a := range plan.ActivitiesToInsert {
		if a.DayID == "" {
			t.Fatal("new activity must receive a day_id")
		}
	}
}

func TestReconcileTree_restores_softdeleted_by_id(t *testing.T) {
	// Current has d1. Incoming references d1 again with id set -> treated as update/restore,
	// not insert, and not deleted.
	current := &model.ItineraryWithTree{
		Itinerary: model.Itinerary{ID: "it1"},
		Days:      []*model.ItineraryDayWithActivities{day("d1", 1)},
	}
	incoming := &model.ItineraryWithTree{
		Itinerary: model.Itinerary{ID: "it1"},
		Days:      []*model.ItineraryDayWithActivities{day("d1", 1)},
	}
	plan := ReconcileTree(current, incoming)
	if len(plan.DaysToDelete) != 0 {
		t.Fatalf("DaysToDelete = %v, want none (restored)", plan.DaysToDelete)
	}
	if len(plan.DaysToUpdate) != 1 {
		t.Fatalf("DaysToUpdate len = %d, want 1", len(plan.DaysToUpdate))
	}
}

func TestPrepareTree_assigns_ids_and_renumbers(t *testing.T) {
	in := &model.ItineraryWithTree{
		Itinerary: model.Itinerary{Title: "t"},
		Days: []*model.ItineraryDayWithActivities{
			day("", 99, act("", "")),
			day("", 0),
		},
	}
	out, err := PrepareTree(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Days[0].DayNumber != 1 || out.Days[1].DayNumber != 2 {
		t.Fatalf("day numbers = %d,%d want 1,2", out.Days[0].DayNumber, out.Days[1].DayNumber)
	}
	if out.Days[0].ID == "" || out.Days[0].Activities[0].ID == "" {
		t.Fatal("ids must be assigned")
	}
	if out.Days[0].Activities[0].DayID != out.Days[0].ID {
		t.Fatal("activity must be linked to its day")
	}
	if *out.TotalDays != "2" || *out.ActivitiesCount != "1" {
		t.Fatalf("counters = %s/%s, want 2/1", *out.TotalDays, *out.ActivitiesCount)
	}
}
func TestFormatCitiesCount_PureNumbers(t *testing.T) {
	c1 := "1"
	if got := FormatCitiesCount(&c1); got == nil || *got != "1" {
		t.Fatalf("got %v, want 1", strOrNil(got))
	}
	c2 := "1 City"
	if got := FormatCitiesCount(&c2); got == nil || *got != "1" {
		t.Fatalf("got %v, want 1", strOrNil(got))
	}
	c3 := "2 Cities"
	if got := FormatCitiesCount(&c3); got == nil || *got != "2" {
		t.Fatalf("got %v, want 2", strOrNil(got))
	}
}

func strOrNil(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
