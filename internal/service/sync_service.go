package service

import (
	"context"
	"fmt"
	"time"

	"checkut-cms-server/internal/model"
	"checkut-cms-server/internal/repository"
	"checkut-cms-server/internal/supabase"
)

type SyncService struct {
	repo *repository.PublishRepo
	meta *repository.PublishMetaRepo
	supa *supabase.Client
}

func NewSyncService(repo *repository.PublishRepo, meta *repository.PublishMetaRepo, supa *supabase.Client) *SyncService {
	return &SyncService{repo: repo, meta: meta, supa: supa}
}

// Status reports whether sync is configured and local counts.
func (s *SyncService) Status(ctx context.Context) (*model.SyncStatus, error) {
	last, err := s.meta.GetLastSyncedAt(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := s.repo.Counts(ctx)
	if err != nil {
		return nil, err
	}
	return &model.SyncStatus{
		Configured:   true,
		LastSyncedAt: last,
		LocalCounts:  counts,
	}, nil
}

// Import pulls all five tables from Supabase and upserts them locally as published.
// Idempotent; per-table errors are collected without rolling back.
func (s *SyncService) Import(ctx context.Context) (*model.ImportResult, error) {
	res := &model.ImportResult{
		Imported: map[string]int{},
		Errors:   []string{},
	}

	importSteps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"destinations", s.importDestinations},
		{"attractions", s.importAttractions},
		{"itineraries", s.importItineraries},
		{"itinerary_days", s.importDays},
		{"itinerary_activities", s.importActivities},
	}

	for _, step := range importSteps {
		if err := step.fn(ctx); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", step.name, err))
		}
	}

	if len(res.Errors) == 0 {
		if err := s.meta.SetLastSyncedAt(ctx, time.Now().UTC()); err != nil {
			res.Errors = append(res.Errors, "publish_meta: "+err.Error())
		}
	}
	return res, nil
}

func (s *SyncService) importDestinations(ctx context.Context) error {
	var rows []model.Destination
	if err := s.supa.FetchAll(ctx, "destinations", "*", &rows); err != nil {
		return err
	}
	for i := range rows {
		d := &rows[i]
		if d.Status == "" {
			d.Status = model.StatusPublished
		}
		if err := s.repo.ImportDestination(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

func (s *SyncService) importAttractions(ctx context.Context) error {
	var rows []model.Attraction
	if err := s.supa.FetchAll(ctx, "attractions", "*", &rows); err != nil {
		return err
	}
	for i := range rows {
		a := &rows[i]
		if a.Status == "" {
			a.Status = model.StatusPublished
		}
		if err := s.repo.ImportAttraction(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

func (s *SyncService) importItineraries(ctx context.Context) error {
	var rows []model.Itinerary
	if err := s.supa.FetchAll(ctx, "itineraries", "*", &rows); err != nil {
		return err
	}
	for i := range rows {
		it := &rows[i]
		if it.Status == "" {
			it.Status = model.StatusPublished
		}
		if err := s.repo.ImportItinerary(ctx, it); err != nil {
			return err
		}
	}
	return nil
}

func (s *SyncService) importDays(ctx context.Context) error {
	var rows []model.ItineraryDay
	if err := s.supa.FetchAll(ctx, "itinerary_days", "*", &rows); err != nil {
		return err
	}
	for i := range rows {
		d := &rows[i]
		if d.Status == "" {
			d.Status = model.StatusPublished
		}
		if err := s.repo.ImportDay(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

func (s *SyncService) importActivities(ctx context.Context) error {
	var rows []model.ItineraryActivity
	if err := s.supa.FetchAll(ctx, "itinerary_activities", "*", &rows); err != nil {
		return err
	}
	for i := range rows {
		a := &rows[i]
		if a.Status == "" {
			a.Status = model.StatusPublished
		}
		if err := s.repo.ImportActivity(ctx, a); err != nil {
			return err
		}
	}
	return nil
}
