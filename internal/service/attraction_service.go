package service

import (
	"context"
	"strings"

	"checkut-cms-server/internal/model"
	"checkut-cms-server/internal/repository"
)

type AttractionService struct {
	repo *repository.AttractionRepo
}

func NewAttractionService(repo *repository.AttractionRepo) *AttractionService {
	return &AttractionService{repo: repo}
}

func (s *AttractionService) List(ctx context.Context, p repository.AttractionListParams) (*model.Page[*model.Attraction], error) {
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
	return &model.Page[*model.Attraction]{Items: items, Total: total, Page: p.Page, PageSize: p.PageSize}, nil
}

func (s *AttractionService) Get(ctx context.Context, id string) (*model.Attraction, error) {
	return s.repo.Get(ctx, id)
}

func (s *AttractionService) Create(ctx context.Context, a *model.Attraction) (*model.Attraction, error) {
	if strings.TrimSpace(a.Title) == "" {
		return nil, ErrInvalid
	}
	if strings.TrimSpace(a.DestinationID) == "" {
		return nil, ErrInvalid
	}
	if a.Status == "" {
		a.Status = model.StatusDraft
	}
	return s.repo.Create(ctx, a)
}

func (s *AttractionService) Update(ctx context.Context, a *model.Attraction) (*model.Attraction, error) {
	if strings.TrimSpace(a.Title) == "" {
		return nil, ErrInvalid
	}
	if strings.TrimSpace(a.DestinationID) == "" {
		return nil, ErrInvalid
	}
	return s.repo.Update(ctx, a)
}

func (s *AttractionService) SetStatus(ctx context.Context, id, status string) (*model.Attraction, error) {
	if !model.ValidStatus(status) {
		return nil, ErrInvalid
	}
	return s.repo.SetStatus(ctx, id, status)
}

func (s *AttractionService) Delete(ctx context.Context, id string) error {
	return s.repo.SoftDelete(ctx, id)
}
