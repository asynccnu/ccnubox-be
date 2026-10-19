package service

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/cache"
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
	if err := s.SetOfficialWhiteList(ctx, domain.OfficialWhiteList{Strategy: "script", StudentIDs: []string{"a", "b"}, ExpiresAt: time.Now().Add(time.Hour).Unix()}); err != nil {
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
	if !converted[0].Targeted || converted[0].StudentIDs[0] != "a" {
		t.Fatal("cache conversion lost recipients")
	}
	c.list.ExpiresAt = time.Now().Add(-time.Hour).Unix()
	if err := s.PublicMuxiOfficialMSG(ctx, msg); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expired list: %v", err)
	}
}
func TestWhiteListValidationAndExplicitAll(t *testing.T) {
	c := &officialCacheStub{}
	s := NewMuxiOfficialMSGService(nil, c, nil)
	ctx := context.Background()
	expires := time.Now().Add(time.Hour).Unix()
	for _, list := range []domain.OfficialWhiteList{
		{Strategy: "script", ExpiresAt: expires},
		{Strategy: "script", ExpiresAt: expires, StudentIDs: []string{"a", "a"}},
		{Strategy: "script", ExpiresAt: expires, StudentIDs: []string{""}},
		{Strategy: "script", ExpiresAt: expires, All: true},
		{Strategy: "all", ExpiresAt: expires, All: true, StudentIDs: []string{"a"}},
	} {
		if err := s.SetOfficialWhiteList(ctx, list); err == nil {
			t.Fatalf("invalid list accepted: %+v", list)
		}
	}
	if err := s.SetOfficialWhiteList(ctx, domain.OfficialWhiteList{Strategy: "all", All: true, ExpiresAt: expires}); err != nil {
		t.Fatal(err)
	}
	if err := s.PublicMuxiOfficialMSG(ctx, &domain.MuxiOfficialMSG{Title: "all"}); err != nil || len(c.stored) != 1 || c.stored[0].Targeted {
		t.Fatalf("broadcast=%+v err=%v", c.stored, err)
	}
}
