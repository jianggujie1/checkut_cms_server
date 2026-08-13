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

type ItineraryRepo struct {
	db *pgxpool.Pool
}

func NewItineraryRepo(db *pgxpool.Pool) *ItineraryRepo { return &ItineraryRepo{db: db} }

const itineraryCols = `id, user_id, destination_id, title, total_days, cities_count, activities_count, status, created_at, updated_at, deleted_at`

func scanItinerary(row pgx.Row) (*model.Itinerary, error) {
	var it model.Itinerary
	err := row.Scan(
		&it.ID, &it.UserID, &it.DestinationID, &it.Title, &it.TotalDays, &it.CitiesCount, &it.ActivitiesCount,
		&it.Status, &it.CreatedAt, &it.UpdatedAt, &it.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

const dayCols = `id, itinerary_id, day_number, title, subtitle, image_url, route_line, status, created_at, updated_at, deleted_at`

func scanDay(row pgx.Row) (*model.ItineraryDay, error) {
	var d model.ItineraryDay
	err := row.Scan(
		&d.ID, &d.ItineraryID, &d.DayNumber, &d.Title, &d.Subtitle, &d.ImageURL, &d.RouteLine,
		&d.Status, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

const activityCols = `id, day_id, attraction_id, time, title, location, description, tip, images, video_url, latitude, longitude, poi_info, source_url, status, created_at, updated_at, deleted_at`

func scanActivity(row pgx.Row) (*model.ItineraryActivity, error) {
	var a model.ItineraryActivity
	err := row.Scan(
		&a.ID, &a.DayID, &a.AttractionID, &a.Time, &a.Title, &a.Location, &a.Description, &a.Tip,
		&a.Images, &a.VideoURL, &a.Latitude, &a.Longitude, &a.POIInfo, &a.SourceURL,
		&a.Status, &a.CreatedAt, &a.UpdatedAt, &a.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ---- queries ----

func (r *ItineraryRepo) List(ctx context.Context, p ListParams) ([]*model.Itinerary, int64, error) {
	where := []string{"deleted_at is null"}
	args := []any{}
	if p.Status != "" {
		args = append(args, p.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if p.Q != "" {
		args = append(args, "%"+p.Q+"%")
		where = append(where, fmt.Sprintf("title ilike $%d", len(args)))
	}

	var total int64
	if err := r.db.QueryRow(ctx,
		"select count(*) from itineraries where "+strings.Join(where, " and "), args...).
		Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, p.PageSize, (p.Page-1)*p.PageSize)
	q := "select " + itineraryCols + " from itineraries where " +
		strings.Join(where, " and ") +
		fmt.Sprintf(" order by created_at desc limit $%d offset $%d", len(args)-1, len(args))
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []*model.Itinerary{}
	for rows.Next() {
		it, err := scanItinerary(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, it)
	}
	return items, total, rows.Err()
}

func (r *ItineraryRepo) Get(ctx context.Context, id string) (*model.Itinerary, error) {
	it, err := scanItinerary(r.db.QueryRow(ctx,
		"select "+itineraryCols+" from itineraries where id = $1 and deleted_at is null", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return it, err
}

func (r *ItineraryRepo) GetAny(ctx context.Context, id string) (*model.Itinerary, error) {
	it, err := scanItinerary(r.db.QueryRow(ctx,
		"select "+itineraryCols+" from itineraries where id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return it, err
}

// GetWithTree loads an itinerary with its days[].activities[] (excluding soft-deleted children).
func (r *ItineraryRepo) GetWithTree(ctx context.Context, id string) (*model.ItineraryWithTree, error) {
	it, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	tree := &model.ItineraryWithTree{Itinerary: *it}

	dayRows, err := r.db.Query(ctx,
		"select "+dayCols+" from itinerary_days where itinerary_id=$1 and deleted_at is null order by day_number", id)
	if err != nil {
		return nil, err
	}
	defer dayRows.Close()

	var days []*model.ItineraryDayWithActivities
	days = []*model.ItineraryDayWithActivities{}
	for dayRows.Next() {
		d, err := scanDay(dayRows)
		if err != nil {
			return nil, err
		}
		days = append(days, &model.ItineraryDayWithActivities{ItineraryDay: *d})
	}
	if err := dayRows.Err(); err != nil {
		return nil, err
	}

	for _, day := range days {
		day.Activities = []*model.ItineraryActivity{}
		actRows, err := r.db.Query(ctx,
			"select "+activityCols+" from itinerary_activities where day_id=$1 and deleted_at is null order by created_at", day.ID)
		if err != nil {
			return nil, err
		}
		for actRows.Next() {
			a, err := scanActivity(actRows)
			if err != nil {
				actRows.Close()
				return nil, err
			}
			day.Activities = append(day.Activities, a)
		}
		actRows.Close()
		if err := actRows.Err(); err != nil {
			return nil, err
		}
	}

	tree.Days = days
	return tree, nil
}

// UpdateScalars persists the itinerary-level editable fields (title, destination_id).
func (r *ItineraryRepo) UpdateScalars(ctx context.Context, it *model.Itinerary) error {
	tag, err := r.db.Exec(ctx,
		`update itineraries set title=$1, destination_id=$2, cities_count=$3, updated_at=now() where id=$4 and deleted_at is null`,
		it.Title, it.DestinationID, FormatCitiesCount(it.CitiesCount), it.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ItineraryRepo) SetStatus(ctx context.Context, id, status string) (*model.Itinerary, error) {
	var updated model.Itinerary
	err := r.db.QueryRow(ctx,
		`update itineraries set status=$2 where id=$1 and deleted_at is null returning `+itineraryCols,
		id, status,
	).Scan(
		&updated.ID, &updated.UserID, &updated.DestinationID, &updated.Title, &updated.TotalDays,
		&updated.CitiesCount, &updated.ActivitiesCount, &updated.Status, &updated.CreatedAt,
		&updated.UpdatedAt, &updated.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// SoftDelete soft-deletes an itinerary and cascades to its days and activities.
func (r *ItineraryRepo) SoftDelete(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`update itineraries set deleted_at=now(), status='archived' where id=$1 and deleted_at is null`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx,
		`update itinerary_days set deleted_at=now(), status='archived' where itinerary_id=$1 and deleted_at is null`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`update itinerary_activities set deleted_at=now(), status='archived'
		 where day_id in (select id from itinerary_days where itinerary_id=$1) and deleted_at is null`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ---- import helpers ----

func (r *ItineraryRepo) UpsertItineraryForImport(ctx context.Context, it *model.Itinerary) error {
	_, err := r.db.Exec(ctx,
		`insert into itineraries (id, user_id, destination_id, title, total_days, cities_count, activities_count, status, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 on conflict (id) do update set
		   user_id=excluded.user_id, destination_id=excluded.destination_id, title=excluded.title,
		   total_days=excluded.total_days, cities_count=excluded.cities_count, activities_count=excluded.activities_count,
		   status=excluded.status, deleted_at=null`,
		it.ID, it.UserID, it.DestinationID, it.Title, it.TotalDays, it.CitiesCount, it.ActivitiesCount, it.Status, it.CreatedAt)
	return err
}
func (r *ItineraryRepo) UpsertDayForImport(ctx context.Context, d *model.ItineraryDay) error {
	_, err := r.db.Exec(ctx,
		`insert into itinerary_days (id, itinerary_id, day_number, title, subtitle, image_url, route_line, status, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 on conflict (id) do update set
		   itinerary_id=excluded.itinerary_id, day_number=excluded.day_number, title=excluded.title,
		   subtitle=excluded.subtitle, image_url=excluded.image_url, route_line=excluded.route_line, status=excluded.status, deleted_at=null`,
		d.ID, d.ItineraryID, d.DayNumber, d.Title, d.Subtitle, d.ImageURL, d.RouteLine, d.Status, d.CreatedAt)
	return err
}

func (r *ItineraryRepo) UpsertActivityForImport(ctx context.Context, a *model.ItineraryActivity) error {
	_, err := r.db.Exec(ctx,
		`insert into itinerary_activities (id, day_id, attraction_id, time, title, location, description, tip, images, video_url, latitude, longitude, poi_info, source_url, status, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		 on conflict (id) do update set
		   day_id=excluded.day_id, attraction_id=excluded.attraction_id, time=excluded.time,
		   title=excluded.title, location=excluded.location, description=excluded.description,
		   tip=excluded.tip, images=excluded.images, video_url=excluded.video_url,
		   latitude=excluded.latitude, longitude=excluded.longitude, poi_info=excluded.poi_info,
		   source_url=excluded.source_url, status=excluded.status, deleted_at=null`,
		a.ID, a.DayID, a.AttractionID, a.Time, a.Title, a.Location, a.Description, a.Tip,
		a.Images, a.VideoURL, a.Latitude, a.Longitude, a.POIInfo, a.SourceURL, a.Status, a.CreatedAt)
	return err
}

// CreateTree inserts an itinerary plus its days and activities. It assigns ids and
// renumbers day_number (1-based), recomputing counters. Returns the stored tree.
func (r *ItineraryRepo) CreateTree(ctx context.Context, incoming *model.ItineraryWithTree) (*model.ItineraryWithTree, error) {
	tree, err := PrepareTree(incoming)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var stored model.Itinerary
	err = tx.QueryRow(ctx,
		`insert into itineraries (user_id, destination_id, title, total_days, cities_count, activities_count, status)
		 values ($1,$2,$3,$4,$5,$6,$7) returning `+itineraryCols,
		tree.UserID, tree.DestinationID, tree.Title, tree.TotalDays, tree.CitiesCount, tree.ActivitiesCount, tree.Status,
	).Scan(
		&stored.ID, &stored.UserID, &stored.DestinationID, &stored.Title, &stored.TotalDays,
		&stored.CitiesCount, &stored.ActivitiesCount, &stored.Status, &stored.CreatedAt,
		&stored.UpdatedAt, &stored.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	tree.ID = stored.ID
	tree.CreatedAt = stored.CreatedAt
	tree.UpdatedAt = stored.UpdatedAt
	tree.DeletedAt = stored.DeletedAt

	for _, day := range tree.Days {
		day.ItineraryID = stored.ID
		var sday model.ItineraryDay
		err := tx.QueryRow(ctx,
			`insert into itinerary_days (itinerary_id, day_number, title, subtitle, image_url, route_line, status)
			 values ($1,$2,$3,$4,$5,$6,$7) returning `+dayCols,
			day.ItineraryID, day.DayNumber, day.Title, day.Subtitle, day.ImageURL, day.RouteLine, day.Status,
		).Scan(
			&sday.ID, &sday.ItineraryID, &sday.DayNumber, &sday.Title, &sday.Subtitle, &sday.ImageURL, &sday.RouteLine,
			&sday.Status, &sday.CreatedAt, &sday.UpdatedAt, &sday.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		day.ID = sday.ID
		day.CreatedAt = sday.CreatedAt
		day.UpdatedAt = sday.UpdatedAt

		for _, act := range day.Activities {
			act.DayID = sday.ID
			var sa model.ItineraryActivity
			err := tx.QueryRow(ctx,
				`insert into itinerary_activities (day_id, attraction_id, time, title, location, description, tip, images, video_url, latitude, longitude, poi_info, source_url, status)
				 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) returning `+activityCols,
				act.DayID, act.AttractionID, act.Time, act.Title, act.Location, act.Description, act.Tip,
				act.Images, act.VideoURL, act.Latitude, act.Longitude, act.POIInfo, act.SourceURL, act.Status,
			).Scan(
				&sa.ID, &sa.DayID, &sa.AttractionID, &sa.Time, &sa.Title, &sa.Location, &sa.Description, &sa.Tip,
				&sa.Images, &sa.VideoURL, &sa.Latitude, &sa.Longitude, &sa.POIInfo, &sa.SourceURL,
				&sa.Status, &sa.CreatedAt, &sa.UpdatedAt, &sa.DeletedAt,
			)
			if err != nil {
				return nil, err
			}
			act.ID = sa.ID
			act.CreatedAt = sa.CreatedAt
			act.UpdatedAt = sa.UpdatedAt
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return tree, nil
}

// ApplyTreePlan executes a prepared plan inside a transaction: updates/inserts/deletes
// days and activities, then updates the itinerary counters.
func (r *ItineraryRepo) ApplyTreePlan(ctx context.Context, plan *TreePlan) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	for _, d := range plan.DaysToUpdate {
		if _, err := tx.Exec(ctx,
			`update itinerary_days set day_number=$1, title=$2, subtitle=$3, image_url=$4, route_line=$5
			 where id=$6 and deleted_at is null`,
			d.DayNumber, d.Title, d.Subtitle, d.ImageURL, d.RouteLine, d.ID); err != nil {
			return err
		}
	}
	// Soft-delete removed days (and their activities) BEFORE inserting new ones so the
	// partial unique index on (itinerary_id, day_number) doesn't see stale active rows.
	for _, id := range plan.DaysToDelete {
		if _, err := tx.Exec(ctx,
			`update itinerary_days set deleted_at=now(), status='archived' where id=$1 and deleted_at is null`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`update itinerary_activities set deleted_at=now(), status='archived' where day_id=$1 and deleted_at is null`, id); err != nil {
			return err
		}
	}
	for _, d := range plan.DaysToInsert {
		if _, err := tx.Exec(ctx,
			`insert into itinerary_days (id, itinerary_id, day_number, title, subtitle, image_url, route_line, status)
			 values ($1,$2,$3,$4,$5,$6,$7,$8) on conflict (id) do nothing`,
			d.ID, plan.ItineraryID, d.DayNumber, d.Title, d.Subtitle, d.ImageURL, d.RouteLine, d.Status); err != nil {
			return err
		}
	}

	for _, id := range plan.ActivitiesToDelete {
		if _, err := tx.Exec(ctx,
			`update itinerary_activities set deleted_at=now(), status='archived' where id=$1 and deleted_at is null`, id); err != nil {
			return err
		}
	}
	for _, a := range plan.ActivitiesToUpdate {
		if _, err := tx.Exec(ctx,
			`update itinerary_activities set day_id=$1, attraction_id=$2, time=$3, title=$4, location=$5, description=$6, tip=$7, images=$8, video_url=$9, latitude=$10, longitude=$11, poi_info=$12, source_url=$13
			 where id=$14 and deleted_at is null`,
			a.DayID, a.AttractionID, a.Time, a.Title, a.Location, a.Description, a.Tip, a.Images, a.VideoURL, a.Latitude, a.Longitude, a.POIInfo, a.SourceURL, a.ID); err != nil {
			return err
		}
	}
	for _, a := range plan.ActivitiesToInsert {
		if _, err := tx.Exec(ctx,
			`insert into itinerary_activities (id, day_id, attraction_id, time, title, location, description, tip, images, video_url, latitude, longitude, poi_info, source_url, status)
			 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) on conflict (id) do nothing`,
			a.ID, a.DayID, a.AttractionID, a.Time, a.Title, a.Location, a.Description, a.Tip, a.Images, a.VideoURL, a.Latitude, a.Longitude, a.POIInfo, a.SourceURL, a.Status); err != nil {
			return err
		}
	}
	for _, id := range plan.ActivitiesToDelete {
		if _, err := tx.Exec(ctx,
			`update itinerary_activities set deleted_at=now(), status='archived' where id=$1 and deleted_at is null`, id); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx,
		`update itineraries set total_days=$1, activities_count=$2, cities_count=$3, updated_at=now() where id=$4`,
		plan.TotalDays, plan.ActivitiesCount, plan.CitiesCount, plan.ItineraryID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
