package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
	"time"
)

type officialCacheStub struct {
	cache.FeedEventCache
	list    domain.OfficialWhiteList
	readErr error
	stored  []cache.MuxiOfficialMSG
	at      int64
}

func (c *officialCacheStub) GetOfficialWhiteList(context.Context) (domain.OfficialWhiteList, error) {
	return c.list, c.readErr
}
func (c *officialCacheStub) SetOfficialWhiteList(_ context.Context, l domain.OfficialWhiteList) error {
	c.list = l
	return nil
}
func (c *officialCacheStub) GetUniqueKey() string { return "scheduled-test" }
func (c *officialCacheStub) SetMuxiFeeds(_ context.Context, m cache.MuxiOfficialMSG, at int64) error {
	c.stored = append(c.stored, m)
	c.at = at
	return nil
}
func TestOfficialPublishRequiresValidWhiteListAndFreezesRecipients(t *testing.T) {
	c := &officialCacheStub{readErr: redis.Nil}
	s := NewMuxiOfficialMSGService(nil, c, nil)
	ctx := context.Background()
	msg := &domain.MuxiOfficialMSG{Title: "survey", Content: "content", PublicTime: time.Now().Add(time.Hour).Unix()}
	if err := s.PublicMuxiOfficialMSG(ctx, msg); status.Code(err) != codes.FailedPrecondition || len(c.stored) != 0 {
		t.Fatalf("missing list: %v", err)
	}
	c.readErr = nil
	if _, err := s.SetOfficialWhiteList(ctx, domain.OfficialWhiteList{Strategy: "script", StudentIDs: []string{"a", "b"}, ExpiresAt: time.Now().Add(time.Hour).Unix()}); err != nil {
		t.Fatal(err)
	}
	if err := s.PublicMuxiOfficialMSG(ctx, msg); err != nil {
		t.Fatal(err)
	}
	if len(c.stored) != 1 || !c.stored[0].Targeted || len(c.stored[0].StudentIDs) != 2 || c.at != msg.PublicTime {
		t.Fatalf("plan=%+v", c.stored)
	}
	c.list.StudentIDs[0] = "changed"
	if c.stored[0].StudentIDs[0] != "a" {
		t.Fatal("changing current list altered scheduled recipients")
	}
	converted := convMuxiMessageFromCacheToDomain(c.stored)
	if !converted[0].Targeted || converted[0].WhiteListStrategy != "script" || converted[0].StudentIDs[0] != "a" {
		t.Fatal("cache conversion lost recipients")
	}
	c.list.ExpiresAt = time.Now().Add(-time.Hour).Unix()
	if err := s.PublicMuxiOfficialMSG(ctx, msg); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expired list: %v", err)
	}
}
func TestWhiteListValidationAndExplicitAll(t *testing.T) {
	c := &officialCacheStub{}
	id := fmt.Sprintf("%d00001", time.Now().Year()-1)
	s := NewMuxiOfficialMSGService(nil, c, &officialUsersStub{batches: [][]string{{id}}})
	ctx := context.Background()
	expires := time.Now().Add(time.Hour).Unix()
	for _, list := range []domain.OfficialWhiteList{
		{Strategy: "script", ExpiresAt: expires},
		{Strategy: "script", ExpiresAt: expires, StudentIDs: []string{"a", "a"}},
		{Strategy: "script", ExpiresAt: expires, StudentIDs: []string{""}},
		{Strategy: "script", ExpiresAt: expires, All: true},
		{Strategy: "all", ExpiresAt: expires, All: true, StudentIDs: []string{"a"}},
	} {
		if _, err := s.SetOfficialWhiteList(ctx, list); err == nil {
			t.Fatalf("invalid list accepted: %+v", list)
		}
	}
	if _, err := s.SetOfficialWhiteList(ctx, domain.OfficialWhiteList{Strategy: "all", All: true, ExpiresAt: expires}); err != nil {
		t.Fatal(err)
	}
	if err := s.PublicMuxiOfficialMSG(ctx, &domain.MuxiOfficialMSG{Title: "all"}); err != nil || len(c.stored) != 1 || !c.stored[0].Targeted || len(c.stored[0].StudentIDs) != 1 {
		t.Fatalf("broadcast=%+v err=%v", c.stored, err)
	}
}

type officialUsersStub struct {
	dao.FeedUserConfigDAO
	batches [][]string
	err     error
	cursors []int64
}

func (d *officialUsersStub) GetStudentIdsByCursor(_ context.Context, last int64, _ int) ([]string, int64, error) {
	d.cursors = append(d.cursors, last)
	if d.err != nil {
		return nil, 0, d.err
	}
	if last >= int64(len(d.batches)) {
		return nil, 0, nil
	}
	return d.batches[last], last + 1, nil
}
func TestOfficialWhiteListAllowsBoundedClockSkew(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		expires int64
		valid   bool
	}{
		{"nominal", now.Add(7 * 24 * time.Hour).Unix(), true},
		{"bff_ahead_one_second", now.Add(7*24*time.Hour + time.Second).Unix(), true},
		{"bff_ahead_one_minute", now.Add(7*24*time.Hour + time.Minute).Unix(), true},
		{"beyond_tolerance", now.Add(7*24*time.Hour + time.Minute + time.Second).Unix(), false},
		{"expired", now.Add(-time.Second).Unix(), false},
		{"expires_now", now.Unix(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOfficialWhiteList(domain.OfficialWhiteList{Strategy: "script", StudentIDs: []string{"a"}, ExpiresAt: tc.expires}, now)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}
func TestAllStrategyPaginatesAndFreezesOnlyCurrentStudents(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	users := &officialUsersStub{batches: [][]string{{"20190001", "invalid"}, {"20220001", "20230001", "20260001", "20270001", "20230001"}}}
	c := &officialCacheStub{}
	s := NewMuxiOfficialMSGService(nil, c, users).(*muxiOfficialMSGService)
	s.now = func() time.Time { return now }
	n, err := s.SetOfficialWhiteList(context.Background(), domain.OfficialWhiteList{Strategy: "all", All: true, ExpiresAt: now.Add(time.Hour).Unix()})
	if err != nil || n != 2 || c.list.All || len(c.list.StudentIDs) != 2 || c.list.StudentIDs[0] != "20230001" || c.list.StudentIDs[1] != "20260001" || len(users.cursors) != 3 {
		t.Fatalf("n=%d list=%+v cursors=%v err=%v", n, c.list, users.cursors, err)
	}
	if err := s.PublicMuxiOfficialMSG(context.Background(), &domain.MuxiOfficialMSG{Title: "all"}); err != nil {
		t.Fatal(err)
	}
	if !c.stored[0].Targeted || c.stored[0].WhiteListStrategy != "all" || len(c.stored[0].StudentIDs) != 2 {
		t.Fatalf("plan=%+v", c.stored)
	}
}
func TestAllStrategyFailurePreservesExistingList(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	original := domain.OfficialWhiteList{Strategy: "script", StudentIDs: []string{"old"}, ExpiresAt: now.Add(time.Hour).Unix()}
	for _, users := range []*officialUsersStub{{err: errors.New("DB unavailable")}, {batches: [][]string{{"19990001", "invalid"}}}} {
		c := &officialCacheStub{list: original}
		s := NewMuxiOfficialMSGService(nil, c, users)
		if _, err := s.SetOfficialWhiteList(ctx, domain.OfficialWhiteList{Strategy: "all", All: true, ExpiresAt: now.Add(time.Hour).Unix()}); err == nil {
			t.Fatal("expected all selection failure")
		}
		if c.list.Strategy != "script" || c.list.StudentIDs[0] != "old" {
			t.Fatal("failed selection overwrote previous list")
		}
	}
}
func TestAllStrategyIsNotLimitedToTenThousandUsers(t *testing.T) {
	now := time.Now()
	users := &officialUsersStub{}
	for i := 0; i < 10001; i++ {
		if i%500 == 0 {
			users.batches = append(users.batches, nil)
		}
		id := fmt.Sprintf("%d%08d", now.Year()-1, i)
		page := len(users.batches) - 1
		users.batches[page] = append(users.batches[page], id)
	}
	c := &officialCacheStub{}
	s := NewMuxiOfficialMSGService(nil, c, users)
	n, err := s.SetOfficialWhiteList(context.Background(), domain.OfficialWhiteList{Strategy: "all", All: true, ExpiresAt: now.Add(time.Hour).Unix()})
	if err != nil || n != 10001 || len(c.list.StudentIDs) != 10001 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
func TestAllStrategyRechecksGraduationForDelayedAndLegacyMessages(t *testing.T) {
	zone := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, zone)
	for _, legacy := range []bool{false, true} {
		c := &officialCacheStub{list: domain.OfficialWhiteList{Strategy: "all", All: legacy, StudentIDs: []string{"20220001", "20230001"}, ExpiresAt: now.Add(time.Hour).Unix()}}
		if legacy {
			c.list.StudentIDs = nil
		}
		s := NewMuxiOfficialMSGService(nil, c, &officialUsersStub{batches: [][]string{{"20220001", "20230001"}}}).(*muxiOfficialMSGService)
		s.now = func() time.Time { return now }
		msg := &domain.MuxiOfficialMSG{Title: "survey", PublicTime: time.Date(2026, 6, 30, 12, 0, 0, 0, zone).Unix()}
		if err := s.PublicMuxiOfficialMSG(context.Background(), msg); err != nil {
			t.Fatal(err)
		}
		if !c.stored[0].Targeted || len(c.stored[0].StudentIDs) != 1 || c.stored[0].StudentIDs[0] != "20230001" {
			t.Fatalf("legacy=%v plan=%+v", legacy, c.stored)
		}
	}
}
