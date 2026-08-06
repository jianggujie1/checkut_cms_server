package model

import (
	"time"
)

// Content status values.
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

func ValidStatus(s string) bool {
	switch s {
	case StatusDraft, StatusPublished, StatusArchived:
		return true
	}
	return false
}

type Destination struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Country      *string  `json:"country"`
	Continent    *string  `json:"continent"`
	Rating       *float64 `json:"rating"`
	ReviewsCount *int32   `json:"reviews_count"`
	ImageURL     *string  `json:"image_url"`
	Description  *string  `json:"description"`
	Tags         []string `json:"tags"`
	BestTime     *string  `json:"best_time"`
	Duration     *string  `json:"duration"`
	Status       string   `json:"status"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

type Attraction struct {
	ID            string     `json:"id"`
	DestinationID string     `json:"destination_id"`
	Title         string     `json:"title"`
	Subtitle      *string    `json:"subtitle"`
	ImageURL      *string    `json:"image_url"`
	Duration      *string    `json:"duration"`
	Tag           *string    `json:"tag"`
	Status        string     `json:"status"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
}

type Itinerary struct {
	ID              string     `json:"id"`
	UserID          *string    `json:"user_id"`
	DestinationID   string     `json:"destination_id"`
	Title           string     `json:"title"`
	TotalDays       *string    `json:"total_days"`
	CitiesCount     *string    `json:"cities_count"`
	ActivitiesCount *string    `json:"activities_count"`
	Status          string     `json:"status"`
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at"`
}

type ItineraryDay struct {
	ID          string     `json:"id"`
	ItineraryID string     `json:"itinerary_id"`
	DayNumber   int32      `json:"day_number"`
	Title       *string    `json:"title"`
	Subtitle    *string    `json:"subtitle"`
	ImageURL    *string    `json:"image_url"`
	Status      string     `json:"status"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

type ItineraryActivity struct {
	ID           string     `json:"id"`
	DayID        string     `json:"day_id"`
	AttractionID *string    `json:"attraction_id"`
	Time         *string    `json:"time"`
	Title        string     `json:"title"`
	Location     *string    `json:"location"`
	Description  *string    `json:"description"`
	Tip          *string    `json:"tip"`
	Status       string     `json:"status"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

// ItineraryWithTree is the nested shape for GET /itineraries/:id and PUT.
type ItineraryWithTree struct {
	Itinerary
	Days []*ItineraryDayWithActivities `json:"days"`
}

type ItineraryDayWithActivities struct {
	ItineraryDay
	Activities []*ItineraryActivity `json:"activities"`
}

// Page is the paginated list envelope data.
type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// UploadResult for POST /uploads.
type UploadResult struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// SyncStatus for GET /sync/status.
type SyncStatus struct {
	Configured   bool              `json:"configured"`
	LastSyncedAt *time.Time        `json:"last_synced_at"`
	LocalCounts  map[string]int64  `json:"local_counts"`
	Message      *string           `json:"message,omitempty"`
}

// ImportResult for POST /sync/import.
type ImportResult struct {
	Imported map[string]int `json:"imported"`
	Errors   []string       `json:"errors"`
}

// ChangeItem is one row in a publish diff group.
type ChangeItem struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Change string `json:"change"`
}

type DiffGroup struct {
	Created []ChangeItem `json:"created"`
	Updated []ChangeItem `json:"updated"`
	Deleted []ChangeItem `json:"deleted"`
}

// PublishDiff for GET /publish/diff.
type PublishDiff struct {
	Destinations        DiffGroup `json:"destinations"`
	Attractions         DiffGroup `json:"attractions"`
	Itineraries         DiffGroup `json:"itineraries"`
	ItineraryDays       DiffGroup `json:"itinerary_days"`
	ItineraryActivities DiffGroup `json:"itinerary_activities"`
}

// PublishResult for POST /publish.
type PublishResult struct {
	OK      bool           `json:"ok"`
	Applied map[string]int `json:"applied"`
	Errors  []string       `json:"errors"`
}
