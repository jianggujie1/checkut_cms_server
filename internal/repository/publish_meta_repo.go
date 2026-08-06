package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PublishMetaRepo struct {
	db *pgxpool.Pool
}

func NewPublishMetaRepo(db *pgxpool.Pool) *PublishMetaRepo { return &PublishMetaRepo{db: db} }

// GetLastSyncedAt returns the last sync timestamp (nil on first run).
func (r *PublishMetaRepo) GetLastSyncedAt(ctx context.Context) (*time.Time, error) {
	var t *time.Time
	err := r.db.QueryRow(ctx,
		"select last_synced_at from publish_meta where id=1").Scan(&t)
	return t, err
}

func (r *PublishMetaRepo) SetLastSyncedAt(ctx context.Context, t time.Time) error {
	_, err := r.db.Exec(ctx,
		"update publish_meta set last_synced_at=$1 where id=1", t)
	return err
}
