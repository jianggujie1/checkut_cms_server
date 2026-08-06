package service

import (
	"context"
	"errors"
	"strings"

	"checkut-cms-server/internal/model"
	"checkut-cms-server/internal/repository"
)

// ErrInvalid is returned for validation failures.
var ErrInvalid = errors.New("invalid request")

type DestinationService struct {
	repo *repository.DestinationRepo
}

func NewDestinationService(repo *repository.DestinationRepo) *DestinationService {
	return &DestinationService{repo: repo}
}

func (s *DestinationService) List(ctx context.Context, p repository.ListParams) (*model.Page[*model.Destination], error) {
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
	return &model.Page[*model.Destination]{Items: items, Total: total, Page: p.Page, PageSize: p.PageSize}, nil
}

func (s *DestinationService) Get(ctx context.Context, id string) (*model.Destination, error) {
	return s.repo.Get(ctx, id)
}

func (s *DestinationService) Create(ctx context.Context, d *model.Destination) (*model.Destination, error) {
	if err := validateDestination(d); err != nil {
		return nil, err
	}
	if d.Status == "" {
		d.Status = model.StatusDraft
	}
	if d.Tags == nil {
		d.Tags = []string{}
	}
	return s.repo.Create(ctx, d)
}

func (s *DestinationService) Update(ctx context.Context, d *model.Destination) (*model.Destination, error) {
	if err := validateDestination(d); err != nil {
		return nil, err
	}
	if d.Tags == nil {
		d.Tags = []string{}
	}
	return s.repo.Update(ctx, d)
}

func (s *DestinationService) SetStatus(ctx context.Context, id, status string) (*model.Destination, error) {
	if !model.ValidStatus(status) {
		return nil, ErrInvalid
	}
	return s.repo.SetStatus(ctx, id, status)
}

func (s *DestinationService) Delete(ctx context.Context, id string) error {
	return s.repo.SoftDelete(ctx, id)
}

func validateDestination(d *model.Destination) error {
	if strings.TrimSpace(d.Title) == "" {
		return ErrInvalid
	}
	if d.Status != "" && !model.ValidStatus(d.Status) {
		return ErrInvalid
	}
	return nil
}
