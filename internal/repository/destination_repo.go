package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"checkut-cms-server/internal/model"
)

type DestinationRepo struct {
	db *pgxpool.Pool
}

func NewDestinationRepo(db *pgxpool.Pool) *DestinationRepo { return &DestinationRepo{db: db} }

const destinationCols = `id, title, country, continent, rating, reviews_count, image_url, description, tags, best_time, duration, status, created_at, updated_at, deleted_at`

func scanDestination(row pgx.Row) (*model.Destination, error) {
	var d model.Destination
	err := row.Scan(
		&d.ID, &d.Title, &d.Country, &d.Continent, &d.Rating, &d.ReviewsCount,
		&d.ImageURL, &d.Description, &d.Tags, &d.BestTime, &d.Duration,
		&d.Status, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

type ListParams struct {
	Page     int
	PageSize int
	Status   string
	Q        string
}

// List returns non-deleted destinations with pagination.
func (r *DestinationRepo) List(ctx context.Context, p ListParams) ([]*model.Destination, int64, error) {
	where := []string{"deleted_at is null"}
	args := []any{}
	add := func(clause string) {
		where = append(where, clause)
	}
	if p.Status != "" {
		args = append(args, p.Status)
		add(fmt.Sprintf("status = $%d", len(args)))
	}
	if p.Q != "" {
		args = append(args, "%"+p.Q+"%")
		add(fmt.Sprintf("title ilike $%d", len(args)))
	}

	var total int64
	if err := r.db.QueryRow(ctx,
		"select count(*) from destinations where "+strings.Join(where, " and "), args...).
		Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, p.PageSize, (p.Page-1)*p.PageSize)
	q := "select " + destinationCols + " from destinations where " +
		strings.Join(where, " and ") +
		fmt.Sprintf(" order by created_at desc limit $%d offset $%d", len(args)-1, len(args))
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []*model.Destination{}
	for rows.Next() {
		d, err := scanDestination(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, d)
	}
	return items, total, rows.Err()
}

func (r *DestinationRepo) Get(ctx context.Context, id string) (*model.Destination, error) {
	d, err := scanDestination(r.db.QueryRow(ctx,
		"select "+destinationCols+" from destinations where id = $1 and deleted_at is null", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

// GetAny returns a destination even if soft-deleted (used by import/publish).
func (r *DestinationRepo) GetAny(ctx context.Context, id string) (*model.Destination, error) {
	d, err := scanDestination(r.db.QueryRow(ctx,
		"select "+destinationCols+" from destinations where id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

func (r *DestinationRepo) Create(ctx context.Context, d *model.Destination) (*model.Destination, error) {
	var created model.Destination
	err := r.db.QueryRow(ctx,
		`insert into destinations (title, country, continent, rating, reviews_count, image_url, description, tags, best_time, duration, status)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 returning `+destinationCols,
		d.Title, d.Country, d.Continent, d.Rating, d.ReviewsCount,
		d.ImageURL, d.Description, d.Tags, d.BestTime, d.Duration, d.Status,
	).Scan(
		&created.ID, &created.Title, &created.Country, &created.Continent, &created.Rating, &created.ReviewsCount,
		&created.ImageURL, &created.Description, &created.Tags, &created.BestTime, &created.Duration,
		&created.Status, &created.CreatedAt, &created.UpdatedAt, &created.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (r *DestinationRepo) Update(ctx context.Context, d *model.Destination) (*model.Destination, error) {
	var updated model.Destination
	err := r.db.QueryRow(ctx,
		`update destinations set title=$1, country=$2, continent=$3, rating=$4, reviews_count=$5,
		   image_url=$6, description=$7, tags=$8, best_time=$9, duration=$10
		 where id=$11 and deleted_at is null
		 returning `+destinationCols,
		d.Title, d.Country, d.Continent, d.Rating, d.ReviewsCount,
		d.ImageURL, d.Description, d.Tags, d.BestTime, d.Duration, d.ID,
	).Scan(
		&updated.ID, &updated.Title, &updated.Country, &updated.Continent, &updated.Rating, &updated.ReviewsCount,
		&updated.ImageURL, &updated.Description, &updated.Tags, &updated.BestTime, &updated.Duration,
		&updated.Status, &updated.CreatedAt, &updated.UpdatedAt, &updated.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (r *DestinationRepo) SetStatus(ctx context.Context, id, status string) (*model.Destination, error) {
	var updated model.Destination
	err := r.db.QueryRow(ctx,
		`update destinations set status=$2 where id=$1 and deleted_at is null returning `+destinationCols,
		id, status,
	).Scan(
		&updated.ID, &updated.Title, &updated.Country, &updated.Continent, &updated.Rating, &updated.ReviewsCount,
		&updated.ImageURL, &updated.Description, &updated.Tags, &updated.BestTime, &updated.Duration,
		&updated.Status, &updated.CreatedAt, &updated.UpdatedAt, &updated.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// SoftDelete sets deleted_at and archives status.
func (r *DestinationRepo) SoftDelete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx,
		`update destinations set deleted_at=now(), status='archived' where id=$1 and deleted_at is null`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// For import: upsert preserving created_at, restoring non-deleted.
func (r *DestinationRepo) UpsertForImport(ctx context.Context, d *model.Destination) error {
	_, err := r.db.Exec(ctx,
		`insert into destinations (id, title, country, continent, rating, reviews_count, image_url, description, tags, best_time, duration, status, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 on conflict (id) do update set
		   title=excluded.title, country=excluded.country, continent=excluded.continent,
		   rating=excluded.rating, reviews_count=excluded.reviews_count, image_url=excluded.image_url,
		   description=excluded.description, tags=excluded.tags, best_time=excluded.best_time,
		   duration=excluded.duration, status=excluded.status, deleted_at=null`,
		d.ID, d.Title, d.Country, d.Continent, d.Rating, d.ReviewsCount,
		d.ImageURL, d.Description, d.Tags, d.BestTime, d.Duration, d.Status, d.CreatedAt)
	return err
}
