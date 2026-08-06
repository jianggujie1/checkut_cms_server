package service

import (
	"context"
	"strings"

	"checkut-cms-server/internal/model"
	"checkut-cms-server/internal/repository"
)

type ItineraryService struct {
	repo *repository.ItineraryRepo
}

func NewItineraryService(repo *repository.ItineraryRepo) *ItineraryService {
	return &ItineraryService{repo: repo}
}

func (s *ItineraryService) List(ctx context.Context, p repository.ListParams) (*model.Page[*model.Itinerary], error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	items, total, err := s.repo.List(ctx, p)
	if err != nil {
		return nil, err
	}
	return &model.Page[*model.Itinerary]{Items: items, Total: total, Page: p.Page, PageSize: p.PageSize}, nil
}

func (s *ItineraryService) GetTree(ctx context.Context, id string) (*model.ItineraryWithTree, error) {
	return s.repo.GetWithTree(ctx, id)
}

func (s *ItineraryService) Create(ctx context.Context, in *model.ItineraryWithTree) (*model.ItineraryWithTree, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, ErrInvalid
	}
	if strings.TrimSpace(in.DestinationID) == "" {
		return nil, ErrInvalid
	}
	if in.Status == "" {
		in.Status = model.StatusDraft
	}
	return s.repo.CreateTree(ctx, in)
}

// Update performs the whole-tree replacement by reconciling against the stored tree.
func (s *ItineraryService) Update(ctx context.Context, in *model.ItineraryWithTree) (*model.ItineraryWithTree, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, ErrInvalid
	}
	if strings.TrimSpace(in.DestinationID) == "" {
		return nil, ErrInvalid
	}
	if in.Status == "" {
		in.Status = model.StatusDraft
	}

	current, err := s.repo.GetWithTree(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	plan := repository.ReconcileTree(current, in)
	if err := s.repo.ApplyTreePlan(ctx, plan); err != nil {
		return nil, err
	}

	// Persist itinerary scalar fields, then re-read the full tree.
	if err := s.repo.UpdateScalars(ctx, &in.Itinerary); err != nil {
		return nil, err
	}
	return s.repo.GetWithTree(ctx, in.ID)
}

func (s *ItineraryService) SetStatus(ctx context.Context, id, status string) (*model.Itinerary, error) {
	if !model.ValidStatus(status) {
		return nil, ErrInvalid
	}
	return s.repo.SetStatus(ctx, id, status)
}

func (s *ItineraryService) Delete(ctx context.Context, id string) error {
	return s.repo.SoftDelete(ctx, id)
}
