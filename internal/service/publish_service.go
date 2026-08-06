package service

import (
	"context"
	"time"

	"checkut-cms-server/internal/model"
	"checkut-cms-server/internal/repository"
	"checkut-cms-server/internal/supabase"
)

// rowInfo is the diff-relevant projection of any content row.
type rowInfo struct {
	ID        string
	Label     string
	Status    string
	CreatedAt *time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

// classify decides a row's diff action: "created", "updated", "deleted", or "".
// lastSync nil means first publish -> all active rows are created.
// supabaseIDs is the set of ids currently present online (for delete detection).
func classify(r rowInfo, lastSync *time.Time, supabaseIDs map[string]bool) string {
	if r.DeletedAt != nil || r.Status == model.StatusArchived {
		previouslySynced := supabaseIDs[r.ID]
		if !previouslySynced && lastSync != nil && r.CreatedAt != nil && !r.CreatedAt.After(*lastSync) {
			previouslySynced = true
		}
		if previouslySynced {
			return "deleted"
		}
		return ""
	}
	if lastSync == nil || (r.CreatedAt != nil && r.CreatedAt.After(*lastSync)) {
		return "created"
	}
	if r.UpdatedAt != nil && r.UpdatedAt.After(*lastSync) {
		return "updated"
	}
	return ""
}

func diffGroup(rows []rowInfo, lastSync *time.Time, supabaseIDs map[string]bool) model.DiffGroup {
	var g model.DiffGroup
	g.Created = []model.ChangeItem{}
	g.Updated = []model.ChangeItem{}
	g.Deleted = []model.ChangeItem{}
	for _, r := range rows {
		action := classify(r, lastSync, supabaseIDs)
		if action == "" {
			continue
		}
		item := model.ChangeItem{ID: r.ID, Label: r.Label, Change: action}
		switch action {
		case "created":
			g.Created = append(g.Created, item)
		case "updated":
			g.Updated = append(g.Updated, item)
		case "deleted":
			g.Deleted = append(g.Deleted, item)
		}
	}
	return g
}

type PublishService struct {
	repo *repository.PublishRepo
	meta *repository.PublishMetaRepo
	supa *supabase.Client
}

func NewPublishService(repo *repository.PublishRepo, meta *repository.PublishMetaRepo, supa *supabase.Client) *PublishService {
	return &PublishService{repo: repo, meta: meta, supa: supa}
}

// ComputeDiff builds the full publish diff (created/updated/deleted per table).
func (s *PublishService) ComputeDiff(ctx context.Context) (*model.PublishDiff, error) {
	lastSync, err := s.meta.GetLastSyncedAt(ctx)
	if err != nil {
		return nil, err
	}

	destIDs, err := s.supa.SelectIDs(ctx, "destinations")
	if err != nil {
		return nil, err
	}
	attrIDs, err := s.supa.SelectIDs(ctx, "attractions")
	if err != nil {
		return nil, err
	}
	itIDs, err := s.supa.SelectIDs(ctx, "itineraries")
	if err != nil {
		return nil, err
	}
	dayIDs, err := s.supa.SelectIDs(ctx, "itinerary_days")
	if err != nil {
		return nil, err
	}
	actIDs, err := s.supa.SelectIDs(ctx, "itinerary_activities")
	if err != nil {
		return nil, err
	}

	dests, err := s.repo.AllDestinations(ctx)
	if err != nil {
		return nil, err
	}
	attrs, err := s.repo.AllAttractions(ctx)
	if err != nil {
		return nil, err
	}
	its, err := s.repo.AllItineraries(ctx)
	if err != nil {
		return nil, err
	}
	days, err := s.repo.AllDays(ctx)
	if err != nil {
		return nil, err
	}
	acts, err := s.repo.AllActivities(ctx)
	if err != nil {
		return nil, err
	}

	diff := &model.PublishDiff{}
	diff.Destinations = diffGroup(toRows(dests), lastSync, destIDs)

	attrInfos := make([]rowInfo, 0, len(attrs))
	for _, a := range attrs {
		attrInfos = append(attrInfos, rowOf(a))
	}
	diff.Attractions = diffGroup(attrInfos, lastSync, attrIDs)

	// Itineraries: parent-dominant. Non-published -> delete if online, else ignore.
	diff.Itineraries.Created = []model.ChangeItem{}
	diff.Itineraries.Updated = []model.ChangeItem{}
	diff.Itineraries.Deleted = []model.ChangeItem{}
	publishedItineraries := map[string]bool{}
	for _, it := range its {
		if it.Status != model.StatusPublished || it.DeletedAt != nil {
			if itIDs[it.ID] {
				diff.Itineraries.Deleted = append(diff.Itineraries.Deleted,
					model.ChangeItem{ID: it.ID, Label: it.Title, Change: "deleted"})
			}
			continue
		}
		publishedItineraries[it.ID] = true
		switch classify(toRow(it), lastSync, itIDs) {
		case "created":
			diff.Itineraries.Created = append(diff.Itineraries.Created,
				model.ChangeItem{ID: it.ID, Label: it.Title, Change: "created"})
		case "updated":
			diff.Itineraries.Updated = append(diff.Itineraries.Updated,
				model.ChangeItem{ID: it.ID, Label: it.Title, Change: "updated"})
		}
	}

	// Days & activities: only for published itineraries.
	dayItinerary := map[string]string{}
	for _, d := range days {
		dayItinerary[d.ID] = d.ItineraryID
	}
	var dayRows, actRows []rowInfo
	for _, d := range days {
		if publishedItineraries[d.ItineraryID] {
			dayRows = append(dayRows, rowOfDay(d))
		}
	}
	diff.ItineraryDays = diffGroup(dayRows, lastSync, dayIDs)
	for _, a := range acts {
		if publishedItineraries[dayItinerary[a.DayID]] {
			actRows = append(actRows, rowOfAct(a))
		}
	}
	diff.ItineraryActivities = diffGroup(actRows, lastSync, actIDs)

	return diff, nil
}

// Run executes the publish: applies the diff to Supabase and refreshes last_synced_at.
func (s *PublishService) Run(ctx context.Context) (*model.PublishResult, error) {
	diff, err := s.ComputeDiff(ctx)
	if err != nil {
		return nil, err
	}

	res := &model.PublishResult{
		OK:      true,
		Applied: map[string]int{},
		Errors:  []string{},
	}
	errf := func(table string, e error) {
		res.Errors = append(res.Errors, table+": "+e.Error())
	}

	data, err := loadPublishData(ctx, s)
	if err != nil {
		return nil, err
	}

	// --- 1. destinations ---
	destByID := indexDestinations(data.dests)
	destSupabase, err := s.supa.SelectIDs(ctx, "destinations")
	if err != nil {
		return nil, err
	}
	upsertDests := pickDestinations(destByID, diff.Destinations.Created, diff.Destinations.Updated)
	if len(upsertDests) > 0 {
		if err := s.supa.Upsert(ctx, "destinations", upsertDests); err != nil {
			errf("destinations", err)
		} else {
			res.Applied["destinations"] = len(upsertDests)
		}
	}
	for _, c := range diff.Destinations.Deleted {
		if err := s.supa.DeleteByIDs(ctx, "destinations", []string{c.ID}); err != nil {
			errf("destinations", err)
		}
	}

	// --- 2. attractions (upsert only if parent destination exists online) ---
	attrByID := indexAttractions(data.attrs)
	upsertAttrs := pickAttractions(attrByID, diff.Attractions.Created, diff.Attractions.Updated)
	upsertAttrs = filterAttrsByParent(upsertAttrs, destSupabase)
	if len(upsertAttrs) > 0 {
		if err := s.supa.Upsert(ctx, "attractions", upsertAttrs); err != nil {
			errf("attractions", err)
		} else {
			res.Applied["attractions"] = len(upsertAttrs)
		}
	}
	for _, c := range diff.Attractions.Deleted {
		if err := s.supa.DeleteByIDs(ctx, "attractions", []string{c.ID}); err != nil {
			errf("attractions", err)
		}
	}

	// --- 3. itineraries: upsert published; delete children first, then self ---
	itByID := indexItineraries(data.its)
	upsertIts := pickItineraries(itByID, diff.Itineraries.Created, diff.Itineraries.Updated)
	if len(upsertIts) > 0 {
		if err := s.supa.Upsert(ctx, "itineraries", upsertIts); err != nil {
			errf("itineraries", err)
		} else {
			res.Applied["itineraries"] = len(upsertIts)
		}
	}
	dayIDsByIT, actIDsByDay := childIndexes(data.days, data.acts)
	for _, c := range diff.Itineraries.Deleted {
		for _, did := range dayIDsByIT[c.ID] {
			if aIDs := actIDsByDay[did]; len(aIDs) > 0 {
				if err := s.supa.DeleteByIDs(ctx, "itinerary_activities", aIDs); err != nil {
					errf("itinerary_activities", err)
				}
			}
			if err := s.supa.DeleteByIDs(ctx, "itinerary_days", []string{did}); err != nil {
				errf("itinerary_days", err)
			}
		}
		if err := s.supa.DeleteByIDs(ctx, "itineraries", []string{c.ID}); err != nil {
			errf("itineraries", err)
		}
	}

	// --- 4. days & activities (diff already limited to published itineraries) ---
	dayByID := indexDays(data.days)
	upsertDays := pickDays(dayByID, diff.ItineraryDays.Created, diff.ItineraryDays.Updated)
	if len(upsertDays) > 0 {
		if err := s.supa.Upsert(ctx, "itinerary_days", upsertDays); err != nil {
			errf("itinerary_days", err)
		} else {
			res.Applied["itinerary_days"] = len(upsertDays)
		}
	}
	for _, c := range diff.ItineraryDays.Deleted {
		if err := s.supa.DeleteByIDs(ctx, "itinerary_days", []string{c.ID}); err != nil {
			errf("itinerary_days", err)
		}
	}
	actByID := indexActs(data.acts)
	upsertActs := pickActs(actByID, diff.ItineraryActivities.Created, diff.ItineraryActivities.Updated)
	if len(upsertActs) > 0 {
		if err := s.supa.Upsert(ctx, "itinerary_activities", upsertActs); err != nil {
			errf("itinerary_activities", err)
		} else {
			res.Applied["itinerary_activities"] = len(upsertActs)
		}
	}
	for _, c := range diff.ItineraryActivities.Deleted {
		if err := s.supa.DeleteByIDs(ctx, "itinerary_activities", []string{c.ID}); err != nil {
			errf("itinerary_activities", err)
		}
	}

	if len(res.Errors) == 0 {
		if err := s.meta.SetLastSyncedAt(ctx, time.Now().UTC()); err != nil {
			res.Errors = append(res.Errors, "publish_meta: "+err.Error())
		}
	}
	if len(res.Errors) > 0 {
		res.OK = false
	}
	return res, nil
}

// ---- row projections ----

func toRows(dests []*model.Destination) []rowInfo {
	out := make([]rowInfo, 0, len(dests))
	for _, d := range dests {
		out = append(out, rowInfo{d.ID, d.Title, d.Status, d.CreatedAt, d.UpdatedAt, d.DeletedAt})
	}
	return out
}

func toRow(it *model.Itinerary) rowInfo {
	return rowInfo{it.ID, it.Title, it.Status, it.CreatedAt, it.UpdatedAt, it.DeletedAt}
}
