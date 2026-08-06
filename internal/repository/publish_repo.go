package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"checkut-cms-server/internal/model"
)

// PublishRepo exposes the bulk reads and writes needed by sync/import and publish.
type PublishRepo struct {
	db *pgxpool.Pool
}

func NewPublishRepo(db *pgxpool.Pool) *PublishRepo { return &PublishRepo{db: db} }

// Counts returns per-table row counts of non-deleted rows.
func (r *PublishRepo) Counts(ctx context.Context) (map[string]int64, error) {
	tables := []string{"destinations", "attractions", "itineraries", "itinerary_days", "itinerary_activities"}
	out := make(map[string]int64, len(tables))
	for _, t := range tables {
		var n int64
		if err := r.db.QueryRow(ctx,
			fmt.Sprintf("select count(*) from %s where deleted_at is null", t)).Scan(&n); err != nil {
			return nil, err
		}
		out[t] = n
	}
	return out, nil
}

func (r *PublishRepo) ImportDestination(ctx context.Context, d *model.Destination) error {
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

func (r *PublishRepo) ImportAttraction(ctx context.Context, a *model.Attraction) error {
	_, err := r.db.Exec(ctx,
		`insert into attractions (id, destination_id, title, subtitle, image_url, duration, tag, status, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 on conflict (id) do update set
		   destination_id=excluded.destination_id, title=excluded.title, subtitle=excluded.subtitle,
		   image_url=excluded.image_url, duration=excluded.duration, tag=excluded.tag,
		   status=excluded.status, deleted_at=null`,
		a.ID, a.DestinationID, a.Title, a.Subtitle, a.ImageURL, a.Duration, a.Tag, a.Status, a.CreatedAt)
	return err
}

func (r *PublishRepo) ImportItinerary(ctx context.Context, it *model.Itinerary) error {
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

func (r *PublishRepo) ImportDay(ctx context.Context, d *model.ItineraryDay) error {
	_, err := r.db.Exec(ctx,
		`insert into itinerary_days (id, itinerary_id, day_number, title, subtitle, image_url, status, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8)
		 on conflict (id) do update set
		   itinerary_id=excluded.itinerary_id, day_number=excluded.day_number, title=excluded.title,
		   subtitle=excluded.subtitle, image_url=excluded.image_url, status=excluded.status, deleted_at=null`,
		d.ID, d.ItineraryID, d.DayNumber, d.Title, d.Subtitle, d.ImageURL, d.Status, d.CreatedAt)
	return err
}

func (r *PublishRepo) ImportActivity(ctx context.Context, a *model.ItineraryActivity) error {
	_, err := r.db.Exec(ctx,
		`insert into itinerary_activities (id, day_id, attraction_id, time, title, location, description, tip, status, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 on conflict (id) do update set
		   day_id=excluded.day_id, attraction_id=excluded.attraction_id, time=excluded.time,
		   title=excluded.title, location=excluded.location, description=excluded.description,
		   tip=excluded.tip, status=excluded.status, deleted_at=null`,
		a.ID, a.DayID, a.AttractionID, a.Time, a.Title, a.Location, a.Description, a.Tip, a.Status, a.CreatedAt)
	return err
}

func (r *PublishRepo) AllDestinations(ctx context.Context) ([]*model.Destination, error) {
	rows, err := r.db.Query(ctx, "select "+destinationCols+" from destinations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Destination
	for rows.Next() {
		d, err := scanDestination(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *PublishRepo) AllAttractions(ctx context.Context) ([]*model.Attraction, error) {
	rows, err := r.db.Query(ctx, "select "+attractionCols+" from attractions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Attraction
	for rows.Next() {
		a, err := scanAttraction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PublishRepo) AllItineraries(ctx context.Context) ([]*model.Itinerary, error) {
	rows, err := r.db.Query(ctx, "select "+itineraryCols+" from itineraries")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Itinerary
	for rows.Next() {
		it, err := scanItinerary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *PublishRepo) AllDays(ctx context.Context) ([]*model.ItineraryDay, error) {
	rows, err := r.db.Query(ctx, "select "+dayCols+" from itinerary_days")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ItineraryDay
	for rows.Next() {
		d, err := scanDay(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *PublishRepo) AllActivities(ctx context.Context) ([]*model.ItineraryActivity, error) {
	rows, err := r.db.Query(ctx, "select "+activityCols+" from itinerary_activities")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ItineraryActivity
	for rows.Next() {
		a, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
