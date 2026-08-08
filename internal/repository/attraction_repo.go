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

type AttractionRepo struct {
	db *pgxpool.Pool
}

func NewAttractionRepo(db *pgxpool.Pool) *AttractionRepo { return &AttractionRepo{db: db} }

const attractionCols = `id, destination_id, title, subtitle, image_url, images, video_url, latitude, longitude, duration, tag, source_url, status, created_at, updated_at, deleted_at`

func scanAttraction(row pgx.Row) (*model.Attraction, error) {
	var a model.Attraction
	err := row.Scan(
		&a.ID, &a.DestinationID, &a.Title, &a.Subtitle, &a.ImageURL, &a.Images, &a.VideoURL,
		&a.Latitude, &a.Longitude, &a.Duration, &a.Tag, &a.SourceURL,
		&a.Status, &a.CreatedAt, &a.UpdatedAt, &a.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

type AttractionListParams struct {
	ListParams
	DestinationID string
}

func (r *AttractionRepo) List(ctx context.Context, p AttractionListParams) ([]*model.Attraction, int64, error) {
	where := []string{"deleted_at is null"}
	args := []any{}
	if p.Status != "" {
		args = append(args, p.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if p.Q != "" {
		args = append(args, "%"+p.Q+"%")
		where = append(where, fmt.Sprintf("(title ilike $%d or subtitle ilike $%d)", len(args), len(args)))
	}
	if p.DestinationID != "" {
		args = append(args, p.DestinationID)
		where = append(where, fmt.Sprintf("destination_id = $%d", len(args)))
	}

	var total int64
	if err := r.db.QueryRow(ctx,
		"select count(*) from attractions where "+strings.Join(where, " and "), args...).
		Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, p.PageSize, (p.Page-1)*p.PageSize)
	q := "select " + attractionCols + " from attractions where " +
		strings.Join(where, " and ") +
		fmt.Sprintf(" order by created_at desc limit $%d offset $%d", len(args)-1, len(args))
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []*model.Attraction{}
	for rows.Next() {
		a, err := scanAttraction(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, a)
	}
	return items, total, rows.Err()
}

func (r *AttractionRepo) Get(ctx context.Context, id string) (*model.Attraction, error) {
	a, err := scanAttraction(r.db.QueryRow(ctx,
		"select "+attractionCols+" from attractions where id = $1 and deleted_at is null", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (r *AttractionRepo) GetAny(ctx context.Context, id string) (*model.Attraction, error) {
	a, err := scanAttraction(r.db.QueryRow(ctx,
		"select "+attractionCols+" from attractions where id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (r *AttractionRepo) Create(ctx context.Context, a *model.Attraction) (*model.Attraction, error) {
	var created model.Attraction
	err := r.db.QueryRow(ctx,
		`insert into attractions (destination_id, title, subtitle, image_url, images, video_url, latitude, longitude, duration, tag, source_url, status)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) returning `+attractionCols,
		a.DestinationID, a.Title, a.Subtitle, a.ImageURL, a.Images, a.VideoURL, a.Latitude, a.Longitude, a.Duration, a.Tag, a.SourceURL, a.Status,
	).Scan(
		&created.ID, &created.DestinationID, &created.Title, &created.Subtitle, &created.ImageURL, &created.Images, &created.VideoURL,
		&created.Latitude, &created.Longitude, &created.Duration, &created.Tag, &created.SourceURL,
		&created.Status, &created.CreatedAt, &created.UpdatedAt, &created.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (r *AttractionRepo) Update(ctx context.Context, a *model.Attraction) (*model.Attraction, error) {
	var updated model.Attraction
	err := r.db.QueryRow(ctx,
		`update attractions set destination_id=$1, title=$2, subtitle=$3, image_url=$4, images=$5, video_url=$6, latitude=$7, longitude=$8, duration=$9, tag=$10, source_url=$11
		 where id=$12 and deleted_at is null returning `+attractionCols,
		a.DestinationID, a.Title, a.Subtitle, a.ImageURL, a.Images, a.VideoURL, a.Latitude, a.Longitude, a.Duration, a.Tag, a.SourceURL, a.ID,
	).Scan(
		&updated.ID, &updated.DestinationID, &updated.Title, &updated.Subtitle, &updated.ImageURL, &updated.Images, &updated.VideoURL,
		&updated.Latitude, &updated.Longitude, &updated.Duration, &updated.Tag, &updated.SourceURL,
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

func (r *AttractionRepo) SetStatus(ctx context.Context, id, status string) (*model.Attraction, error) {
	var updated model.Attraction
	err := r.db.QueryRow(ctx,
		`update attractions set status=$2 where id=$1 and deleted_at is null returning `+attractionCols,
		id, status,
	).Scan(
		&updated.ID, &updated.DestinationID, &updated.Title, &updated.Subtitle, &updated.ImageURL, &updated.Images, &updated.VideoURL,
		&updated.Latitude, &updated.Longitude, &updated.Duration, &updated.Tag, &updated.SourceURL,
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

func (r *AttractionRepo) SoftDelete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx,
		`update attractions set deleted_at=now(), status='archived' where id=$1 and deleted_at is null`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *AttractionRepo) UpsertForImport(ctx context.Context, a *model.Attraction) error {
	_, err := r.db.Exec(ctx,
		`insert into attractions (id, destination_id, title, subtitle, image_url, images, video_url, latitude, longitude, duration, tag, source_url, status, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 on conflict (id) do update set
		   destination_id=excluded.destination_id, title=excluded.title, subtitle=excluded.subtitle,
		   image_url=excluded.image_url, images=excluded.images, video_url=excluded.video_url,
		   latitude=excluded.latitude, longitude=excluded.longitude, duration=excluded.duration, tag=excluded.tag,
		   source_url=excluded.source_url, status=excluded.status, deleted_at=null`,
		a.ID, a.DestinationID, a.Title, a.Subtitle, a.ImageURL, a.Images, a.VideoURL, a.Latitude, a.Longitude, a.Duration, a.Tag, a.SourceURL, a.Status, a.CreatedAt)
	return err
}
