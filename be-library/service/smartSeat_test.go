package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/crawler"
	"github.com/asynccnu/ccnubox-be/be-library/tool"
	v1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
)

func newTimeline(roomID, seatID string, intervals ...[2]int) seatTimeline {
	items := make([]seatInterval, 0, len(intervals))
	for _, interval := range intervals {
		items = append(items, seatInterval{Start: interval[0], End: interval[1]})
	}
	return seatTimeline{
		RoomID:    roomID,
		SeatID:    seatID,
		SeatLabel: "A区 " + seatID,
		Intervals: items,
	}
}

func assertSlicesContiguous(t *testing.T, slices []seatInterval) {
	t.Helper()
	for i := 1; i < len(slices); i++ {
		if slices[i].Start != slices[i-1].End {
			t.Fatalf("slices not contiguous at %d: %+v", i, slices)
		}
	}
}

func TestSplitSmartSeatWindow(t *testing.T) {
	t.Run("exact 30 minute slices", func(t *testing.T) {
		slices := splitSmartSeatWindow(540, 720)
		if len(slices) != 6 {
			t.Fatalf("got %d slices, want 6", len(slices))
		}
		for i, slice := range slices {
			if slice.End-slice.Start != 30 {
				t.Fatalf("slice %d length %d, want 30", i, slice.End-slice.Start)
			}
		}
		assertSlicesContiguous(t, slices)
		if slices[0].Start != 540 || slices[len(slices)-1].End != 720 {
			t.Fatalf("window not covered: %+v", slices)
		}
	})

	t.Run("remainder goes to the last slice", func(t *testing.T) {
		slices := splitSmartSeatWindow(540, 700)
		if len(slices) != 6 {
			t.Fatalf("got %d slices, want 6", len(slices))
		}
		assertSlicesContiguous(t, slices)
		if slices[0].Start != 540 || slices[len(slices)-1].End != 700 {
			t.Fatalf("window not covered: %+v", slices)
		}
	})

	t.Run("short window gets two slices", func(t *testing.T) {
		slices := splitSmartSeatWindow(540, 600)
		if len(slices) != 2 {
			t.Fatalf("got %d slices, want 2", len(slices))
		}
		assertSlicesContiguous(t, slices)
		if slices[0].Start != 540 || slices[1].End != 600 {
			t.Fatalf("window not covered: %+v", slices)
		}
	})

	t.Run("capped at max slices", func(t *testing.T) {
		slices := splitSmartSeatWindow(480, 780)
		if len(slices) != smartSeatMaxSlices {
			t.Fatalf("got %d slices, want %d", len(slices), smartSeatMaxSlices)
		}
		assertSlicesContiguous(t, slices)
		if slices[0].Start != 480 || slices[len(slices)-1].End != 780 {
			t.Fatalf("window not covered: %+v", slices)
		}
	})

	t.Run("invalid window", func(t *testing.T) {
		if slices := splitSmartSeatWindow(600, 600); slices != nil {
			t.Fatalf("expected nil for empty window, got %+v", slices)
		}
		if slices := splitSmartSeatWindow(700, 600); slices != nil {
			t.Fatalf("expected nil for reversed window, got %+v", slices)
		}
	})
}

func TestParseSeatIntervals(t *testing.T) {
	t.Run("valid intervals are sorted", func(t *testing.T) {
		intervals := parseSeatIntervals([]*crawler.FreeTime{
			{Start: "13:00", End: "14:30"},
			{Start: "09:00", End: "12:00"},
		})
		if len(intervals) != 2 {
			t.Fatalf("got %d intervals, want 2", len(intervals))
		}
		if intervals[0] != (seatInterval{Start: 540, End: 720}) || intervals[1] != (seatInterval{Start: 780, End: 870}) {
			t.Fatalf("unexpected intervals: %+v", intervals)
		}
	})

	t.Run("minute strings are accepted", func(t *testing.T) {
		intervals := parseSeatIntervals([]*crawler.FreeTime{{Start: "540", End: "600"}})
		if len(intervals) != 1 || intervals[0] != (seatInterval{Start: 540, End: 600}) {
			t.Fatalf("unexpected intervals: %+v", intervals)
		}
	})

	t.Run("invalid entries are skipped", func(t *testing.T) {
		intervals := parseSeatIntervals([]*crawler.FreeTime{
			nil,
			{Start: "abc", End: "12:00"},
			{Start: "12:00", End: "09:00"},
			{Start: "09:00", End: "10:00"},
		})
		if len(intervals) != 1 || intervals[0] != (seatInterval{Start: 540, End: 600}) {
			t.Fatalf("unexpected intervals: %+v", intervals)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		if intervals := parseSeatIntervals(nil); len(intervals) != 0 {
			t.Fatalf("expected no intervals, got %+v", intervals)
		}
	})
}

func TestClipIntervals(t *testing.T) {
	intervals := []seatInterval{
		{Start: 480, End: 600},
		{Start: 700, End: 800},
		{Start: 400, End: 500},
	}
	clipped := clipIntervals(intervals, 540, 720)
	want := []seatInterval{{Start: 540, End: 600}, {Start: 700, End: 720}}
	if len(clipped) != len(want) {
		t.Fatalf("got %+v, want %+v", clipped, want)
	}
	for i := range want {
		if clipped[i] != want[i] {
			t.Fatalf("got %+v, want %+v", clipped, want)
		}
	}
}

func TestIntervalsCoverAndPoint(t *testing.T) {
	t.Run("interval covering the whole window", func(t *testing.T) {
		if !intervalsCover([]seatInterval{{Start: 540, End: 720}}, 540, 720) {
			t.Fatal("expected cover")
		}
		if !intervalsCover([]seatInterval{{Start: 500, End: 800}}, 540, 720) {
			t.Fatal("expected cover for wider interval")
		}
		if intervalsCover([]seatInterval{{Start: 540, End: 700}}, 540, 720) {
			t.Fatal("expected no cover for shorter interval")
		}
	})

	t.Run("timeline covers point", func(t *testing.T) {
		item := newTimeline("r1", "a", [2]int{540, 600})
		if !timelineCoversPoint(&item, 540) || !timelineCoversPoint(&item, 599) {
			t.Fatal("expected point coverage inside interval")
		}
		if timelineCoversPoint(&item, 539) || timelineCoversPoint(&item, 600) {
			t.Fatal("expected no coverage outside interval")
		}
	})
}

func TestGreedySmartSeatChain(t *testing.T) {
	t.Run("single longest seat wins", func(t *testing.T) {
		timelines := []seatTimeline{
			newTimeline("r1", "a", [2]int{540, 600}),
			newTimeline("r1", "c", [2]int{540, 700}),
		}
		chain, ok := greedySmartSeatChain(timelines, 540, 700, nil)
		if !ok || len(chain) != 1 || chain[0].SeatID != "c" {
			t.Fatalf("unexpected chain: %+v (ok=%v)", chain, ok)
		}
	})

	t.Run("two segments form a contiguous chain", func(t *testing.T) {
		timelines := []seatTimeline{
			newTimeline("r1", "a", [2]int{540, 600}),
			newTimeline("r2", "b", [2]int{600, 720}),
		}
		chain, ok := greedySmartSeatChain(timelines, 540, 720, nil)
		if !ok || len(chain) != 2 {
			t.Fatalf("unexpected chain: %+v (ok=%v)", chain, ok)
		}
		if chain[0].SeatID != "a" || chain[0].Start != 540 || chain[0].End != 600 {
			t.Fatalf("unexpected first segment: %+v", chain[0])
		}
		if chain[1].SeatID != "b" || chain[1].Start != 600 || chain[1].End != 720 {
			t.Fatalf("unexpected second segment: %+v", chain[1])
		}
	})

	t.Run("forced first seat", func(t *testing.T) {
		timelines := []seatTimeline{
			newTimeline("r1", "a", [2]int{540, 600}),
			newTimeline("r2", "b", [2]int{540, 720}),
		}
		chain, ok := greedySmartSeatChain(timelines, 540, 720, &timelines[0])
		if !ok || len(chain) != 2 || chain[0].SeatID != "a" {
			t.Fatalf("unexpected chain: %+v (ok=%v)", chain, ok)
		}
		if chain[0].End != chain[1].Start {
			t.Fatalf("chain is not contiguous: %+v", chain)
		}
	})

	t.Run("uncoverable window", func(t *testing.T) {
		timelines := []seatTimeline{newTimeline("r1", "a", [2]int{540, 600})}
		if chain, ok := greedySmartSeatChain(timelines, 540, 720, nil); ok {
			t.Fatalf("expected failure, got %+v", chain)
		}
	})

	t.Run("too many segments", func(t *testing.T) {
		var timelines []seatTimeline
		for i := 0; i < 13; i++ {
			timelines = append(timelines, newTimeline("r1", "s", [2]int{540 + i, 541 + i}))
		}
		if chain, ok := greedySmartSeatChain(timelines, 540, 553, nil); ok {
			t.Fatalf("expected failure for too many segments, got %+v", chain)
		}
	})
}

func TestBuildSmartSeatChains(t *testing.T) {
	timelines := []seatTimeline{
		newTimeline("r1", "a", [2]int{540, 720}),
		newTimeline("r1", "b", [2]int{540, 600}),
		newTimeline("r1", "c", [2]int{600, 720}),
	}
	chains := buildSmartSeatChains(timelines, 540, 720)
	if len(chains) != 2 {
		t.Fatalf("got %d chains, want 2: %+v", len(chains), chains)
	}
	if len(chains[0]) != 1 || chains[0][0].SeatID != "a" {
		t.Fatalf("unexpected best chain: %+v", chains[0])
	}
	if len(chains[1]) != 2 || chains[1][0].SeatID != "b" || chains[1][1].SeatID != "a" {
		t.Fatalf("unexpected variant chain: %+v", chains[1])
	}
	if chains[1][0].End != chains[1][1].Start {
		t.Fatalf("variant chain is not contiguous: %+v", chains[1])
	}
}

func TestDedupeAndSortSmartSeatChains(t *testing.T) {
	one := []smartSeatPick{{RoomID: "r1", SeatID: "a", Start: 540, End: 720}}
	two := []smartSeatPick{
		{RoomID: "r1", SeatID: "b", Start: 540, End: 600},
		{RoomID: "r1", SeatID: "c", Start: 600, End: 720},
	}

	t.Run("duplicates removed", func(t *testing.T) {
		chains := dedupeSmartSeatChains([][]smartSeatPick{one, one, two, nil})
		if len(chains) != 2 {
			t.Fatalf("got %d chains, want 2: %+v", len(chains), chains)
		}
	})

	t.Run("sorted by segment count then room rank", func(t *testing.T) {
		x := []smartSeatPick{{RoomID: "r1", SeatID: "x", Start: 540, End: 720}}
		y := []smartSeatPick{{RoomID: "r2", SeatID: "y", Start: 540, End: 720}}
		chains := [][]smartSeatPick{two, x, y}
		sortSmartSeatChains(chains, map[string]int{"r2": 0, "r1": 1})
		if len(chains) != 3 {
			t.Fatalf("got %d chains, want 3", len(chains))
		}
		if chains[0][0].SeatID != "y" {
			t.Fatalf("expected preferred room seat y first, got %+v", chains[0])
		}
		if len(chains[2]) != 2 {
			t.Fatalf("expected multi-segment chain last, got %+v", chains[2])
		}
	})
}

func newSmartSegment(seatID, start, end string) *v1.SmartSeatSegment {
	return &v1.SmartSeatSegment{
		SeatId:    seatID,
		SeatLabel: "A区 " + seatID,
		RoomId:    "room-1",
		Start:     start,
		End:       end,
	}
}

func TestNormalizeSmartSeatSegments(t *testing.T) {
	t.Run("valid unordered segments are sorted", func(t *testing.T) {
		segments, err := normalizeSmartSeatSegments([]*v1.SmartSeatSegment{
			newSmartSegment("s2", "10:00", "11:00"),
			newSmartSegment("s1", "09:00", "10:00"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(segments) != 2 {
			t.Fatalf("got %d segments, want 2", len(segments))
		}
		if segments[0].SeatID != "s1" || segments[0].StartMinute != 540 || segments[0].EndMinute != 600 {
			t.Fatalf("unexpected first segment: %+v", segments[0])
		}
		if segments[1].SeatID != "s2" || segments[1].StartMinute != 600 || segments[1].EndMinute != 660 {
			t.Fatalf("unexpected second segment: %+v", segments[1])
		}
	})

	t.Run("minute strings are accepted", func(t *testing.T) {
		segments, err := normalizeSmartSeatSegments([]*v1.SmartSeatSegment{newSmartSegment("s1", "540", "600")})
		if err != nil || len(segments) != 1 || segments[0].StartMinute != 540 || segments[0].EndMinute != 600 {
			t.Fatalf("unexpected result: %+v (err=%v)", segments, err)
		}
	})

	t.Run("broken chain is rejected", func(t *testing.T) {
		_, err := normalizeSmartSeatSegments([]*v1.SmartSeatSegment{
			newSmartSegment("s1", "09:00", "10:00"),
			newSmartSegment("s2", "10:30", "11:00"),
		})
		if err == nil {
			t.Fatal("expected error for non-continuous chain")
		}
	})

	t.Run("overlapping chain is rejected", func(t *testing.T) {
		_, err := normalizeSmartSeatSegments([]*v1.SmartSeatSegment{
			newSmartSegment("s1", "09:00", "10:00"),
			newSmartSegment("s2", "09:30", "10:30"),
		})
		if err == nil {
			t.Fatal("expected error for overlapping chain")
		}
	})

	t.Run("missing seat id is rejected", func(t *testing.T) {
		if _, err := normalizeSmartSeatSegments([]*v1.SmartSeatSegment{newSmartSegment("", "09:00", "10:00")}); err == nil {
			t.Fatal("expected error for missing seat id")
		}
	})

	t.Run("empty segments rejected", func(t *testing.T) {
		if _, err := normalizeSmartSeatSegments(nil); err == nil {
			t.Fatal("expected error for empty segments")
		}
	})

	t.Run("invalid time is rejected", func(t *testing.T) {
		if _, err := normalizeSmartSeatSegments([]*v1.SmartSeatSegment{newSmartSegment("s1", "abc", "10:00")}); err == nil {
			t.Fatal("expected error for invalid start time")
		}
	})

	t.Run("too many segments rejected", func(t *testing.T) {
		var segments []*v1.SmartSeatSegment
		for i := 0; i <= smartSeatMaxSegments; i++ {
			start := 540 + i*10
			segments = append(segments, newSmartSegment("s1", crawler.FormatMinute(start), crawler.FormatMinute(start+10)))
		}
		if _, err := normalizeSmartSeatSegments(segments); err == nil {
			t.Fatal("expected error for too many segments")
		}
	})
}

func TestSmartSeatGroupTTL(t *testing.T) {
	loc := tool.GetLocation()

	t.Run("ttl until next day plus one hour", func(t *testing.T) {
		now, err := time.ParseInLocation("2006-01-02 15:04", "2026-09-09 12:00", loc)
		if err != nil {
			t.Fatalf("parse now: %v", err)
		}
		ttl := smartSeatGroupTTL("2026-09-10", now)
		if ttl != 37*time.Hour {
			t.Fatalf("got %v, want 37h", ttl)
		}
	})

	t.Run("same day booking", func(t *testing.T) {
		now, err := time.ParseInLocation("2006-01-02 15:04", "2026-09-10 12:00", loc)
		if err != nil {
			t.Fatalf("parse now: %v", err)
		}
		ttl := smartSeatGroupTTL("2026-09-10", now)
		if ttl != 13*time.Hour {
			t.Fatalf("got %v, want 13h", ttl)
		}
	})

	t.Run("invalid date falls back", func(t *testing.T) {
		ttl := smartSeatGroupTTL("not-a-date", time.Now())
		if ttl != 48*time.Hour {
			t.Fatalf("got %v, want 48h", ttl)
		}
	})

	t.Run("expired date is clamped", func(t *testing.T) {
		now, err := time.ParseInLocation("2006-01-02 15:04", "2026-09-12 12:00", loc)
		if err != nil {
			t.Fatalf("parse now: %v", err)
		}
		ttl := smartSeatGroupTTL("2026-09-10", now)
		if ttl != time.Hour {
			t.Fatalf("got %v, want 1h", ttl)
		}
	})
}

func TestNormalizeSmartSeatRoomIDs(t *testing.T) {
	t.Run("dedupe and keep order", func(t *testing.T) {
		got, err := normalizeSmartSeatRoomIDs([]string{"r2", "r1", "r2", " r1 "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 || got[0] != "r2" || got[1] != "r1" {
			t.Fatalf("got %+v, want [r2 r1]", got)
		}
	})

	t.Run("blank entries are dropped", func(t *testing.T) {
		got, err := normalizeSmartSeatRoomIDs([]string{" ", "", "r1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0] != "r1" {
			t.Fatalf("got %+v, want [r1]", got)
		}
	})

	t.Run("empty input rejected", func(t *testing.T) {
		if _, err := normalizeSmartSeatRoomIDs([]string{"", "  "}); err == nil {
			t.Fatal("expected error for empty rooms")
		}
	})

	t.Run("max rooms boundary", func(t *testing.T) {
		rooms := make([]string, 0, smartSeatMaxRooms)
		for i := 0; i < smartSeatMaxRooms; i++ {
			rooms = append(rooms, fmt.Sprintf("r%d", i))
		}
		if _, err := normalizeSmartSeatRoomIDs(rooms); err != nil {
			t.Fatalf("unexpected error at boundary: %v", err)
		}
		rooms = append(rooms, "overflow")
		if _, err := normalizeSmartSeatRoomIDs(rooms); err == nil {
			t.Fatal("expected error when exceeding max rooms")
		}
	})
}
