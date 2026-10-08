package service

import (
	"testing"

	"github.com/asynccnu/ccnubox-be/be-library/crawler"
)

func newSeatCandidate(roomID, seatID string) seatCandidate {
	return seatCandidate{RoomID: roomID, Seat: &crawler.Seat{ID: seatID, Label: "A区 " + seatID}}
}

func TestPickRandomSeat(t *testing.T) {
	t.Run("empty candidates", func(t *testing.T) {
		if _, ok := pickRandomSeat(nil, nil); ok {
			t.Fatal("expected no candidate for empty input")
		}
	})

	t.Run("excluded seat is skipped", func(t *testing.T) {
		candidates := []seatCandidate{newSeatCandidate("room-1", "s1"), newSeatCandidate("room-1", "s2")}
		for i := 0; i < 20; i++ {
			picked, ok := pickRandomSeat(candidates, []string{"s2"})
			if !ok {
				t.Fatal("expected a candidate")
			}
			if picked.Seat.ID != "s1" {
				t.Fatalf("picked seat %s, want s1", picked.Seat.ID)
			}
		}
	})

	t.Run("all excluded falls back to full candidates", func(t *testing.T) {
		candidates := []seatCandidate{newSeatCandidate("room-1", "s1"), newSeatCandidate("room-2", "s2")}
		seen := make(map[string]struct{})
		for i := 0; i < 50; i++ {
			picked, ok := pickRandomSeat(candidates, []string{"s1", "s2"})
			if !ok {
				t.Fatal("expected a candidate")
			}
			if picked.Seat.ID != "s1" && picked.Seat.ID != "s2" {
				t.Fatalf("picked unexpected seat %s", picked.Seat.ID)
			}
			seen[picked.Seat.ID] = struct{}{}
		}
		if len(seen) == 0 {
			t.Fatal("no seat picked")
		}
	})

	t.Run("picked seat always belongs to candidates", func(t *testing.T) {
		candidates := []seatCandidate{
			newSeatCandidate("room-1", "s1"),
			newSeatCandidate("room-1", "s2"),
			newSeatCandidate("room-2", "s3"),
		}
		valid := map[string]string{"s1": "room-1", "s2": "room-1", "s3": "room-2"}
		for i := 0; i < 50; i++ {
			picked, ok := pickRandomSeat(candidates, nil)
			if !ok {
				t.Fatal("expected a candidate")
			}
			roomID, exists := valid[picked.Seat.ID]
			if !exists {
				t.Fatalf("picked seat %s not in candidates", picked.Seat.ID)
			}
			if picked.RoomID != roomID {
				t.Fatalf("seat %s belongs to room %s, got %s", picked.Seat.ID, roomID, picked.RoomID)
			}
		}
	})
}

func TestValidateSeatDate(t *testing.T) {
	if err := validateSeatDate("2026-09-10"); err != nil {
		t.Fatalf("valid date rejected: %v", err)
	}
	for _, invalid := range []string{"", "2026/09/10", "tomorrow", "2026-13-01"} {
		if err := validateSeatDate(invalid); err == nil {
			t.Fatalf("invalid date %q accepted", invalid)
		}
	}
}
