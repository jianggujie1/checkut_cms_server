package service

import (
	"context"

	"checkut-cms-server/internal/model"
)

func rowOf(a *model.Attraction) rowInfo {
	return rowInfo{a.ID, a.Title, a.Status, a.CreatedAt, a.UpdatedAt, a.DeletedAt}
}

func rowOfDay(d *model.ItineraryDay) rowInfo {
	return rowInfo{d.ID, strOr(d.Title), d.Status, d.CreatedAt, d.UpdatedAt, d.DeletedAt}
}

func rowOfAct(a *model.ItineraryActivity) rowInfo {
	return rowInfo{a.ID, a.Title, a.Status, a.CreatedAt, a.UpdatedAt, a.DeletedAt}
}

func strOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// --- destination helpers ---

func indexDestinations(rows []*model.Destination) map[string]*model.Destination {
	m := make(map[string]*model.Destination, len(rows))
	for _, r := range rows {
		m[r.ID] = r
	}
	return m
}

func pickDestinations(m map[string]*model.Destination, created, updated []model.ChangeItem) []*model.Destination {
	var out []*model.Destination
	for _, c := range append(append([]model.ChangeItem{}, created...), updated...) {
		if d, ok := m[c.ID]; ok {
			out = append(out, d)
		}
	}
	return out
}

// --- attraction helpers ---

func indexAttractions(rows []*model.Attraction) map[string]*model.Attraction {
	m := make(map[string]*model.Attraction, len(rows))
	for _, r := range rows {
		m[r.ID] = r
	}
	return m
}

func pickAttractions(m map[string]*model.Attraction, created, updated []model.ChangeItem) []*model.Attraction {
	var out []*model.Attraction
	for _, c := range append(append([]model.ChangeItem{}, created...), updated...) {
		if a, ok := m[c.ID]; ok {
			out = append(out, a)
		}
	}
	return out
}

func filterAttrsByParent(rows []*model.Attraction, parentOnline map[string]bool) []*model.Attraction {
	out := rows[:0]
	for _, a := range rows {
		if parentOnline[a.DestinationID] {
			out = append(out, a)
		}
	}
	return out
}

// --- itinerary helpers ---

func indexItineraries(rows []*model.Itinerary) map[string]*model.Itinerary {
	m := make(map[string]*model.Itinerary, len(rows))
	for _, r := range rows {
		m[r.ID] = r
	}
	return m
}

func pickItineraries(m map[string]*model.Itinerary, created, updated []model.ChangeItem) []*model.Itinerary {
	var out []*model.Itinerary
	for _, c := range append(append([]model.ChangeItem{}, created...), updated...) {
		if it, ok := m[c.ID]; ok {
			out = append(out, it)
		}
	}
	return out
}

// --- day helpers ---

func indexDays(rows []*model.ItineraryDay) map[string]*model.ItineraryDay {
	m := make(map[string]*model.ItineraryDay, len(rows))
	for _, r := range rows {
		m[r.ID] = r
	}
	return m
}

func pickDays(m map[string]*model.ItineraryDay, created, updated []model.ChangeItem) []*model.ItineraryDay {
	var out []*model.ItineraryDay
	for _, c := range append(append([]model.ChangeItem{}, created...), updated...) {
		if d, ok := m[c.ID]; ok {
			out = append(out, d)
		}
	}
	return out
}

// --- activity helpers ---

func indexActs(rows []*model.ItineraryActivity) map[string]*model.ItineraryActivity {
	m := make(map[string]*model.ItineraryActivity, len(rows))
	for _, r := range rows {
		m[r.ID] = r
	}
	return m
}

func pickActs(m map[string]*model.ItineraryActivity, created, updated []model.ChangeItem) []*model.ItineraryActivity {
	var out []*model.ItineraryActivity
	for _, c := range append(append([]model.ChangeItem{}, created...), updated...) {
		if a, ok := m[c.ID]; ok {
			out = append(out, a)
		}
	}
	return out
}

// childIndexes maps itinerary->day ids and day->activity ids.
func childIndexes(days []*model.ItineraryDay, acts []*model.ItineraryActivity) (map[string][]string, map[string][]string) {
	byIT := map[string][]string{}
	byDay := map[string][]string{}
	for _, d := range days {
		byIT[d.ItineraryID] = append(byIT[d.ItineraryID], d.ID)
	}
	for _, a := range acts {
		byDay[a.DayID] = append(byDay[a.DayID], a.ID)
	}
	return byIT, byDay
}

// publishData is all five content tables loaded for a publish run.
type publishData struct {
	dests []*model.Destination
	attrs []*model.Attraction
	its   []*model.Itinerary
	days  []*model.ItineraryDay
	acts  []*model.ItineraryActivity
}

func loadPublishData(ctx context.Context, s *PublishService) (*publishData, error) {
	var d publishData
	var err error
	if d.dests, err = s.repo.AllDestinations(ctx); err != nil {
		return nil, err
	}
	if d.attrs, err = s.repo.AllAttractions(ctx); err != nil {
		return nil, err
	}
	if d.its, err = s.repo.AllItineraries(ctx); err != nil {
		return nil, err
	}
	if d.days, err = s.repo.AllDays(ctx); err != nil {
		return nil, err
	}
	if d.acts, err = s.repo.AllActivities(ctx); err != nil {
		return nil, err
	}
	return &d, nil
}
