package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"checkut-cms-server/internal/model"
)

// Integration tests exercise CRUD + tree reconcile against a real PostgreSQL.
// They require TEST_CMS_DB_DSN; otherwise they skip. Run with:
//
//	TEST_CMS_DB_DSN=postgres://... go test ./internal/repository/ -run Integration
func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_CMS_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_CMS_DB_DSN not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestIntegration_DestinationCRUD(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	repo := NewDestinationRepo(pool)

	created, err := repo.Create(ctx, &model.Destination{Title: "Integration Dest", Status: model.StatusDraft})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("create must generate an id")
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Integration Dest" {
		t.Fatalf("title = %q", got.Title)
	}

	if _, err := repo.SetStatus(ctx, created.ID, model.StatusArchived); err != nil {
		t.Fatal(err)
	}
	if err := repo.SoftDelete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, created.ID); err != ErrNotFound {
		t.Fatalf("soft-deleted row should be hidden, got err=%v", err)
	}
}

func TestIntegration_ItineraryTreeReconcile(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	destRepo := NewDestinationRepo(pool)
	itRepo := NewItineraryRepo(pool)

	dest, err := destRepo.Create(ctx, &model.Destination{Title: "IT Dest", Status: model.StatusDraft})
	if err != nil {
		t.Fatal(err)
	}

	created, err := itRepo.CreateTree(ctx, &model.ItineraryWithTree{
		Itinerary: model.Itinerary{DestinationID: dest.ID, Title: "Tree 1", Status: model.StatusDraft},
		Days: []*model.ItineraryDayWithActivities{
			{ItineraryDay: model.ItineraryDay{Title: sp("Day 1")}, Activities: []*model.ItineraryActivity{{Title: "A1"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	dayID := created.Days[0].ID
	if created.Days[0].DayNumber != 1 {
		t.Fatalf("day_number = %d want 1", created.Days[0].DayNumber)
	}
	if *created.TotalDays != "1" || *created.ActivitiesCount != "1" {
		t.Fatalf("counters = %s/%s", *created.TotalDays, *created.ActivitiesCount)
	}

	incoming := &model.ItineraryWithTree{
		Itinerary: model.Itinerary{ID: created.ID, DestinationID: dest.ID, Title: "Tree 1 updated"},
		Days: []*model.ItineraryDayWithActivities{
			{ItineraryDay: model.ItineraryDay{ID: dayID, Title: sp("Day 1 edited")}, Activities: []*model.ItineraryActivity{{ID: created.Days[0].Activities[0].ID, Title: "A1 edited"}}},
			{ItineraryDay: model.ItineraryDay{Title: sp("Day 2")}, Activities: []*model.ItineraryActivity{{Title: "B1"}}},
		},
	}
	current, err := itRepo.GetWithTree(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	plan := ReconcileTree(current, incoming)
	if err := itRepo.ApplyTreePlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	if err := itRepo.UpdateScalars(ctx, &incoming.Itinerary); err != nil {
		t.Fatal(err)
	}

	tree, err := itRepo.GetWithTree(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Days) != 2 {
		t.Fatalf("days = %d want 2", len(tree.Days))
	}
	if tree.Days[1].DayNumber != 2 {
		t.Fatalf("second day_number = %d want 2", tree.Days[1].DayNumber)
	}
	if tree.Days[0].Activities[0].Title != "A1 edited" {
		t.Fatalf("activity title = %q", tree.Days[0].Activities[0].Title)
	}

	if err := itRepo.SoftDelete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if err := destRepo.SoftDelete(ctx, dest.ID); err != nil {
		t.Fatal(err)
	}
}

func sp(s string) *string { return &s }
