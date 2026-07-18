package biz

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeCache struct {
	data map[string]string
}

func (f fakeCache) Get(_ context.Context, key string) (string, error) {
	val, ok := f.data[key]
	if !ok {
		return "", errors.New("cache miss")
	}
	return val, nil
}

func (f fakeCache) Set(_ context.Context, _ string, _ interface{}, _ time.Duration) error {
	return nil
}

func (f fakeCache) Del(_ context.Context, _ ...string) error {
	return nil
}

func (f fakeCache) SAdd(_ context.Context, _ string, _ ...interface{}) error {
	return nil
}

func (f fakeCache) SMembers(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (f fakeCache) SExpire(_ context.Context, _ string, _ time.Duration) error {
	return nil
}

func TestToSerializableClassroomStatsSortsNaturally(t *testing.T) {
	stats := toSerializableClassroomStats(map[string][]bool{
		"n102":  {true},
		"n101":  {true},
		"1002":  {true},
		"1001":  {true},
		"n1001": {true},
		"n201":  {true},
	})

	got := make([]string, 0, len(stats))
	for _, stat := range stats {
		got = append(got, stat.Classroom)
	}

	want := []string{"1001", "1002", "n101", "n102", "n201", "n1001"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected classroom order: got %v, want %v", got, want)
	}
}

func TestHasUniformAvailability(t *testing.T) {
	if !hasUniformAvailability(map[string][]bool{
		"n101": {true, true},
		"n102": {true, true},
	}) {
		t.Fatal("expected all-true stats to be uniform")
	}

	if hasUniformAvailability(map[string][]bool{
		"n101": {true, false},
		"n102": {true, true},
	}) {
		t.Fatal("expected mixed stats not to be uniform")
	}
}

func TestGetFreeClassRoomFromCacheReturnsErrorOnPartialMiss(t *testing.T) {
	cacheKey := "ccnubox_freeclassroom:2024:2:6:2:1:1"
	f := NewFreeClassroomBiz(nil, nil, nil, nil, fakeCache{
		data: map[string]string{
			cacheKey: `["n101","n102"]`,
		},
	}, nil)

	_, err := f.GetFreeClassRoomFromCache(context.Background(), "2024", "2", 6, 2, 1, []int{1, 2}, "n1")
	if err == nil {
		t.Fatal("expected partial cache miss to return an error")
	}
}
