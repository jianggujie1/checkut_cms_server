package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"checkut-cms-server/internal/model"
)

func toValidUUID(id string) string {
	if id == "" {
		return ""
	}
	if len(id) == 36 && id[8] == '-' && id[13] == '-' && id[18] == '-' && id[23] == '-' {
		return id
	}
	h := md5.Sum([]byte(id))
	h[6] = (h[6] & 0x0f) | 0x40
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

func toValidUUIDPtr(id *string) *string {
	if id == nil {
		return nil
	}
	v := toValidUUID(*id)
	return &v
}

func rowOf(a *model.Attraction) rowInfo {
	return rowInfo{toValidUUID(a.ID), a.Title, a.Status, a.CreatedAt, a.UpdatedAt, a.DeletedAt}
}

func rowOfDay(d *model.ItineraryDay) rowInfo {
	return rowInfo{toValidUUID(d.ID), strOr(d.Title), d.Status, d.CreatedAt, d.UpdatedAt, d.DeletedAt}
}

func rowOfAct(a *model.ItineraryActivity) rowInfo {
	return rowInfo{toValidUUID(a.ID), a.Title, a.Status, a.CreatedAt, a.UpdatedAt, a.DeletedAt}
}

func strOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// --- destination helpers ---

func indexDestinations(rows []*model.Destination) map[string]*model.Destination {
	m := make(map[string]*model.Destination, len(rows))
	for _, r := range rows {
		m[r.ID] = r
	}
	return m
}

func pickDestinations(m map[string]*model.Destination, created, updated []model.ChangeItem) []*model.Destination {
	var out []*model.Destination
	for _, c := range append(append([]model.ChangeItem{}, created...), updated...) {
		if d, ok := m[c.ID]; ok {
			out = append(out, d)
		}
	}
	return out
}

// --- attraction helpers ---
func indexAttractions(rows []*model.Attraction) map[string]*model.Attraction {
	m := make(map[string]*model.Attraction, len(rows))
	for _, r := range rows {
		m[toValidUUID(r.ID)] = r
	}
	return m
}

func pickAttractions(m map[string]*model.Attraction, created, updated []model.ChangeItem) []*model.Attraction {
	var out []*model.Attraction
	for _, c := range append(append([]model.ChangeItem{}, created...), updated...) {
		if a, ok := m[c.ID]; ok {
			out = append(out, a)
		}
	}
	return out
}

func filterAttrsByParent(rows []*model.Attraction, parentOnline map[string]bool) []*model.Attraction {
	out := rows[:0]
	for _, a := range rows {
		if parentOnline[a.DestinationID] {
			out = append(out, a)
		}
	}
	return out
}

// --- itinerary helpers ---

func indexItineraries(rows []*model.Itinerary) map[string]*model.Itinerary {
	m := make(map[string]*model.Itinerary, len(rows))
	for _, r := range rows {
		m[r.ID] = r
	}
	return m
}

func pickItineraries(m map[string]*model.Itinerary, created, updated []model.ChangeItem) []*model.Itinerary {
	var out []*model.Itinerary
	for _, c := range append(append([]model.ChangeItem{}, created...), updated...) {
		if it, ok := m[c.ID]; ok {
			out = append(out, it)
		}
	}
	return out
}

// --- day helpers ---
func indexDays(rows []*model.ItineraryDay) map[string]*model.ItineraryDay {
	m := make(map[string]*model.ItineraryDay, len(rows))
	for _, r := range rows {
		m[toValidUUID(r.ID)] = r
	}
	return m
}

func pickDays(m map[string]*model.ItineraryDay, created, updated []model.ChangeItem) []*model.ItineraryDay {
	var out []*model.ItineraryDay
	for _, c := range append(append([]model.ChangeItem{}, created...), updated...) {
		if d, ok := m[c.ID]; ok {
			out = append(out, d)
		}
	}
	return out
}

// --- activity helpers ---
func indexActs(rows []*model.ItineraryActivity) map[string]*model.ItineraryActivity {
	m := make(map[string]*model.ItineraryActivity, len(rows))
	for _, r := range rows {
		m[toValidUUID(r.ID)] = r
	}
	return m
}

func pickActs(m map[string]*model.ItineraryActivity, created, updated []model.ChangeItem) []*model.ItineraryActivity {
	var out []*model.ItineraryActivity
	for _, c := range append(append([]model.ChangeItem{}, created...), updated...) {
		if a, ok := m[c.ID]; ok {
			out = append(out, a)
		}
	}
	return out
}

func childIndexes(days []*model.ItineraryDay, acts []*model.ItineraryActivity) (map[string][]string, map[string][]string) {
	byIT := map[string][]string{}
	byDay := map[string][]string{}
	for _, d := range days {
		byIT[d.ItineraryID] = append(byIT[d.ItineraryID], toValidUUID(d.ID))
	}
	for _, a := range acts {
		byDay[toValidUUID(a.DayID)] = append(byDay[toValidUUID(a.DayID)], toValidUUID(a.ID))
	}
	return byIT, byDay
}

// publishData is all five content tables loaded for a publish run.
type publishData struct {
	dests []*model.Destination
	attrs []*model.Attraction
	its   []*model.Itinerary
	days  []*model.ItineraryDay
	acts  []*model.ItineraryActivity
}

func loadPublishData(ctx context.Context, s *PublishService) (*publishData, error) {
	var d publishData
	var err error
	if d.dests, err = s.repo.AllDestinations(ctx); err != nil {
		return nil, err
	}
	if d.attrs, err = s.repo.AllAttractions(ctx); err != nil {
		return nil, err
	}
	if d.its, err = s.repo.AllItineraries(ctx); err != nil {
		return nil, err
	}
	if d.days, err = s.repo.AllDays(ctx); err != nil {
		return nil, err
	}
	if d.acts, err = s.repo.AllActivities(ctx); err != nil {
		return nil, err
	}
	return &d, nil
}
// --- Supabase payload DTOs (without CMS status column) ---

type supaDestination struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Country      *string    `json:"country"`
	Continent    *string    `json:"continent"`
	Rating       *float64   `json:"rating"`
	ReviewsCount *int32     `json:"reviews_count"`
	ImageURL     *string    `json:"image_url"`
	Description  *string    `json:"description"`
	Tags         []string   `json:"tags"`
	BestTime     *string    `json:"best_time"`
	Duration     *string    `json:"duration"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

type supaAttraction struct {
	ID            string     `json:"id"`
	DestinationID string     `json:"destination_id"`
	Title         string     `json:"title"`
	Subtitle      *string    `json:"subtitle"`
	ImageURL      *string    `json:"image_url"`
	Images        []string   `json:"images"`
	VideoURL      *string    `json:"video_url"`
	Latitude      *float64   `json:"latitude"`
	Longitude     *float64   `json:"longitude"`
	Duration      *string    `json:"duration"`
	Tag           *string    `json:"tag"`
	SourceURL     *string    `json:"source_url"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
}

type supaItinerary struct {
	ID              string     `json:"id"`
	UserID          *string    `json:"user_id"`
	DestinationID   string     `json:"destination_id"`
	Title           string     `json:"title"`
	TotalDays       *string    `json:"total_days"`
	CitiesCount     *string    `json:"cities_count"`
	ActivitiesCount *string    `json:"activities_count"`
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at"`
}

type supaItineraryDay struct {
	ID          string     `json:"id"`
	ItineraryID string     `json:"itinerary_id"`
	DayNumber   int32      `json:"day_number"`
	Title       *string    `json:"title"`
	Subtitle    *string    `json:"subtitle"`
	ImageURL    *string    `json:"image_url"`
	RouteLine   []any      `json:"route_line"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

type supaItineraryActivity struct {
	ID           string         `json:"id"`
	DayID        string         `json:"day_id"`
	AttractionID *string        `json:"attraction_id"`
	Time         *string        `json:"time"`
	Title        string         `json:"title"`
	Location     *string        `json:"location"`
	Description  *string        `json:"description"`
	Tip          *string        `json:"tip"`
	Images       []string       `json:"images"`
	VideoURL     *string        `json:"video_url"`
	Latitude     *float64       `json:"latitude"`
	Longitude    *float64       `json:"longitude"`
	POIInfo      map[string]any `json:"poi_info"`
	SourceURL    *string        `json:"source_url"`
	CreatedAt    *time.Time     `json:"created_at"`
	UpdatedAt    *time.Time     `json:"updated_at"`
	DeletedAt    *time.Time     `json:"deleted_at"`
}

func ensureImageURL(img *string, images []string) *string {
	if img != nil && strings.TrimSpace(*img) != "" {
		return img
	}
	for _, u := range images {
		if strings.TrimSpace(u) != "" {
			trimmed := strings.TrimSpace(u)
			return &trimmed
		}
	}
	empty := ""
	return &empty
}

func ensureDayImageURL(img *string) *string {
	if img != nil && strings.TrimSpace(*img) != "" {
		return img
	}
	empty := ""
	return &empty
}

func toSupaDestinations(in []*model.Destination) []supaDestination {
	out := make([]supaDestination, len(in))
	for i, d := range in {
		out[i] = supaDestination{
			ID:           d.ID,
			Title:        d.Title,
			Country:      d.Country,
			Continent:    d.Continent,
			Rating:       d.Rating,
			ReviewsCount: d.ReviewsCount,
			ImageURL:     ensureDayImageURL(d.ImageURL),
			Description:  d.Description,
			Tags:         d.Tags,
			BestTime:     d.BestTime,
			Duration:     d.Duration,
			CreatedAt:    d.CreatedAt,
			UpdatedAt:    d.UpdatedAt,
			DeletedAt:    d.DeletedAt,
		}
	}
	return out
}

func toSupaAttractions(in []*model.Attraction) []supaAttraction {
	out := make([]supaAttraction, len(in))
	for i, a := range in {
		out[i] = supaAttraction{
			ID:            toValidUUID(a.ID),
			DestinationID: a.DestinationID,
			Title:         a.Title,
			Subtitle:      a.Subtitle,
			ImageURL:      ensureImageURL(a.ImageURL, a.Images),
			Images:        a.Images,
			VideoURL:      a.VideoURL,
			Latitude:      a.Latitude,
			Longitude:     a.Longitude,
			Duration:      a.Duration,
			Tag:           a.Tag,
			SourceURL:     a.SourceURL,
			CreatedAt:     a.CreatedAt,
			UpdatedAt:     a.UpdatedAt,
			DeletedAt:     a.DeletedAt,
		}
	}
	return out
}

func toSupaItineraries(in []*model.Itinerary) []supaItinerary {
	out := make([]supaItinerary, len(in))
	for i, it := range in {
		out[i] = supaItinerary{
			ID:              it.ID,
			UserID:          it.UserID,
			DestinationID:   it.DestinationID,
			Title:           it.Title,
			TotalDays:       it.TotalDays,
			CitiesCount:     it.CitiesCount,
			ActivitiesCount: it.ActivitiesCount,
			CreatedAt:       it.CreatedAt,
			UpdatedAt:       it.UpdatedAt,
			DeletedAt:       it.DeletedAt,
		}
	}
	return out
}

func toSupaDays(in []*model.ItineraryDay) []supaItineraryDay {
	out := make([]supaItineraryDay, len(in))
	for i, d := range in {
		out[i] = supaItineraryDay{
			ID:          toValidUUID(d.ID),
			ItineraryID: d.ItineraryID,
			DayNumber:   d.DayNumber,
			Title:       d.Title,
			Subtitle:    d.Subtitle,
			ImageURL:    ensureDayImageURL(d.ImageURL),
			RouteLine:   d.RouteLine,
			CreatedAt:   d.CreatedAt,
			UpdatedAt:   d.UpdatedAt,
			DeletedAt:   d.DeletedAt,
		}
	}
	return out
}

func toSupaActs(in []*model.ItineraryActivity) []supaItineraryActivity {
	out := make([]supaItineraryActivity, len(in))
	for i, a := range in {
		out[i] = supaItineraryActivity{
			ID:           toValidUUID(a.ID),
			DayID:        toValidUUID(a.DayID),
			AttractionID: toValidUUIDPtr(a.AttractionID),
			Time:         a.Time,
			Title:        a.Title,
			Location:     a.Location,
			Description:  a.Description,
			Tip:          a.Tip,
			Images:       a.Images,
			VideoURL:     a.VideoURL,
			Latitude:     a.Latitude,
			Longitude:    a.Longitude,
			POIInfo:      a.POIInfo,
			SourceURL:    a.SourceURL,
			CreatedAt:    a.CreatedAt,
			UpdatedAt:    a.UpdatedAt,
			DeletedAt:    a.DeletedAt,
		}
	}
	return out
}
