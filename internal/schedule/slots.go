package schedule

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
)

// slotHorizonDays bounds how far ahead NextOpenSlot searches for an open slot.
const slotHorizonDays = 14

// CreateSlot adds a recurring posting slot to a channel (workspace-checked).
func (s *Service) CreateSlot(ctx context.Context, workspaceID, channelID uuid.UUID, dayOfWeek int, timeOfDay, timezone string) (Slot, error) {
	if _, err := s.channels.PlatformFor(ctx, workspaceID, channelID); err != nil {
		return Slot{}, err
	}
	if dayOfWeek < 0 || dayOfWeek > 6 {
		return Slot{}, apperr.Validation("invalid_day", "day_of_week must be 0..6")
	}
	if _, _, err := parseHHMM(timeOfDay); err != nil {
		return Slot{}, apperr.Validation("invalid_time", "time_of_day must be HH:MM")
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return Slot{}, apperr.Validation("invalid_timezone", "timezone must be a valid IANA name")
	}

	// #nosec G115 -- dayOfWeek is bounds-checked to 0..6 above.
	row, err := s.pool.Queries().CreateScheduleSlot(ctx, sqlc.CreateScheduleSlotParams{
		ChannelID: channelID, DayOfWeek: int16(dayOfWeek), TimeOfDay: timeOfDay, Timezone: timezone,
	})
	if err != nil {
		return Slot{}, apperr.Internal(err)
	}
	return toSlot(row), nil
}

// ListSlots returns a channel's posting slots (workspace-checked).
func (s *Service) ListSlots(ctx context.Context, workspaceID, channelID uuid.UUID) ([]Slot, error) {
	if _, err := s.channels.PlatformFor(ctx, workspaceID, channelID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Queries().ListSlotsForChannel(ctx, channelID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	slots := make([]Slot, len(rows))
	for i, r := range rows {
		slots[i] = toSlot(r)
	}
	return slots, nil
}

// DeleteSlot removes a posting slot from a channel (workspace-checked).
func (s *Service) DeleteSlot(ctx context.Context, workspaceID, channelID, slotID uuid.UUID) error {
	if _, err := s.channels.PlatformFor(ctx, workspaceID, channelID); err != nil {
		return err
	}
	if err := s.pool.Queries().DeleteScheduleSlot(ctx, slotID); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// NextOpenSlot returns the earliest slot time strictly after `after` that is not
// already occupied by a scheduled job on the channel. Slot times are computed in
// each slot's own timezone (so DST is handled by the tz database) and returned
// in UTC.
func (s *Service) NextOpenSlot(ctx context.Context, channelID uuid.UUID, after time.Time) (time.Time, error) {
	slots, err := s.pool.Queries().ListSlotsForChannel(ctx, channelID)
	if err != nil {
		return time.Time{}, apperr.Internal(err)
	}
	if len(slots) == 0 {
		return time.Time{}, apperr.Validation("no_slots", "channel has no posting slots; schedule a specific time instead")
	}

	occupied, err := s.occupiedSlots(ctx, channelID, after)
	if err != nil {
		return time.Time{}, err
	}

	candidates := slotCandidates(slots, after)
	for _, c := range candidates {
		if _, taken := occupied[c.Truncate(time.Minute).UTC()]; !taken {
			return c.UTC(), nil
		}
	}
	return time.Time{}, apperr.Validation("no_open_slot", "no open slot within the scheduling horizon")
}

// occupiedSlots returns the set of already-scheduled run times (minute-truncated, UTC).
func (s *Service) occupiedSlots(ctx context.Context, channelID uuid.UUID, after time.Time) (map[time.Time]struct{}, error) {
	rows, err := s.pool.Queries().ListScheduledRunAtForChannel(ctx, sqlc.ListScheduledRunAtForChannelParams{
		ChannelID: channelID, RunAt: tsFromTime(after.UTC()),
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	set := make(map[time.Time]struct{}, len(rows))
	for _, r := range rows {
		set[r.Time.Truncate(time.Minute).UTC()] = struct{}{}
	}
	return set, nil
}

// slotCandidates expands the recurring slots into concrete times after `after`,
// within the horizon, sorted ascending.
func slotCandidates(slots []sqlc.ScheduleSlot, after time.Time) []time.Time {
	var out []time.Time
	for _, slot := range slots {
		loc, err := time.LoadLocation(slot.Timezone)
		if err != nil {
			continue
		}
		hh, mm, err := parseHHMM(slot.TimeOfDay)
		if err != nil {
			continue
		}
		base := after.In(loc)
		// Generate EVERY occurrence within the horizon (not just the first), so the
		// caller can skip occupied times and re-queue to the same recurring slot on
		// a later week.
		for d := 0; d <= slotHorizonDays; d++ {
			day := base.AddDate(0, 0, d)
			if int(day.Weekday()) != int(slot.DayOfWeek) {
				continue
			}
			cand := time.Date(day.Year(), day.Month(), day.Day(), hh, mm, 0, 0, loc)
			if cand.After(after) {
				out = append(out, cand)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

// parseHHMM parses a "HH:MM" 24-hour time.
func parseHHMM(s string) (hour, minute int, err error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time %q", s)
	}
	hour, err = strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("invalid hour in %q", s)
	}
	minute, err = strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid minute in %q", s)
	}
	return hour, minute, nil
}

func toSlot(r sqlc.ScheduleSlot) Slot {
	return Slot{
		ID: r.ID, ChannelID: r.ChannelID, DayOfWeek: int(r.DayOfWeek),
		TimeOfDay: r.TimeOfDay, Timezone: r.Timezone, CreatedAt: r.CreatedAt.Time,
	}
}
