package repository

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"checkut-cms-server/internal/model"
)

// newUUID generates a random uuid v4 string.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]), hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]), hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]))
}

// PrepareTree normalizes an incoming tree for creation: assigns ids to rows with
// empty id, renumbers day_number 1-based, and recomputes counters.
func PrepareTree(in *model.ItineraryWithTree) (*model.ItineraryWithTree, error) {
	out := *in
	totalDays := 0
	activitiesCount := 0
	for i, day := range in.Days {
		if day.ID == "" {
			day.ID = newUUID()
		}
		day.DayNumber = int32(i + 1)
		if day.Status == "" {
			day.Status = model.StatusDraft
		}
		totalDays++
		for _, act := range day.Activities {
			if act.ID == "" {
				act.ID = newUUID()
			}
			if act.Status == "" {
				act.Status = model.StatusDraft
			}
			act.DayID = day.ID
			activitiesCount++
		}
	}
	out.TotalDays = intToPtr(totalDays)
	out.ActivitiesCount = intToPtr(activitiesCount)
	out.CitiesCount = FormatCitiesCount(in.CitiesCount)
	return &out, nil
}

// TreePlan is the set of operations to apply for a whole-tree PUT.
type TreePlan struct {
	ItineraryID string

	Days                []*model.ItineraryDay
	DaysToInsert        []*model.ItineraryDay
	DaysToUpdate        []*model.ItineraryDay
	DaysToDelete        []string
	ActivitiesToInsert  []*model.ItineraryActivity
	ActivitiesToUpdate  []*model.ItineraryActivity
	ActivitiesToDelete  []string
	TotalDays           *string
	CitiesCount         *string
	ActivitiesCount     *string
}

// ReconcileTree computes the plan to transform current into incoming.
// Rows with id=='' are new (assigned ids); rows not present and existing are
// soft-deleted. day_number is renumbered 1-based by array order and the itinerary
// counters are recomputed. A previously soft-deleted row reappearing by id is
// restored (deleted_at cleared).
func ReconcileTree(current, incoming *model.ItineraryWithTree) *TreePlan {
	plan := &TreePlan{ItineraryID: current.ID}

	currentDays := map[string]*model.ItineraryDay{}
	currentActs := map[string]*model.ItineraryActivity{}
	currentDayIDs := map[string]string{} // activityID -> dayID (current)
	for _, d := range current.Days {
		currentDays[d.ID] = &d.ItineraryDay
		for _, a := range d.Activities {
			currentActs[a.ID] = a
			currentDayIDs[a.ID] = d.ID
		}
	}

	seenDays := map[string]bool{}
	seenActs := map[string]bool{}
	totalDays := 0
	activitiesCount := 0

	for i, day := range incoming.Days {
		day.DayNumber = int32(i + 1)
		if day.Status == "" {
			day.Status = model.StatusDraft
		}
		totalDays++
		if day.ID == "" {
			day.ID = newUUID()
			plan.DaysToInsert = append(plan.DaysToInsert, &day.ItineraryDay)
		} else {
			seenDays[day.ID] = true
			if _, exists := currentDays[day.ID]; exists {
				plan.DaysToUpdate = append(plan.DaysToUpdate, &day.ItineraryDay)
			} else {
				// was soft-deleted previously -> restore via insert update; treat as insert
				plan.DaysToInsert = append(plan.DaysToInsert, &day.ItineraryDay)
			}
		}
		plan.Days = append(plan.Days, &day.ItineraryDay)

		for _, act := range day.Activities {
			if act.Status == "" {
				act.Status = model.StatusDraft
			}
			act.DayID = day.ID
			activitiesCount++
			if act.ID == "" {
				act.ID = newUUID()
				plan.ActivitiesToInsert = append(plan.ActivitiesToInsert, act)
			} else {
				seenActs[act.ID] = true
				if _, exists := currentActs[act.ID]; exists {
					plan.ActivitiesToUpdate = append(plan.ActivitiesToUpdate, act)
				} else {
					plan.ActivitiesToInsert = append(plan.ActivitiesToInsert, act)
				}
			}
		}
	}

	// Days existing in current but missing from incoming -> soft delete.
	for id := range currentDays {
		if !seenDays[id] {
			plan.DaysToDelete = append(plan.DaysToDelete, id)
		}
	}
	// Activities existing in current but missing from incoming -> soft delete.
	for id := range currentActs {
		if !seenActs[id] {
			plan.ActivitiesToDelete = append(plan.ActivitiesToDelete, id)
		}
	}

	plan.TotalDays = intToPtr(totalDays)
	plan.ActivitiesCount = intToPtr(activitiesCount)
	plan.CitiesCount = FormatCitiesCount(incoming.CitiesCount)
	return plan
}

func FormatCitiesCount(s *string) *string {
	if s == nil {
		return nil
	}
	str := strings.TrimSpace(*s)
	if str == "" {
		return nil
	}
	var digits strings.Builder
	for _, r := range str {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	if digits.Len() > 0 {
		res := digits.String()
		return &res
	}
	return &str
}

func intToPtr(n int) *string {
	s := fmt.Sprintf("%d", n)
	return &s
}
