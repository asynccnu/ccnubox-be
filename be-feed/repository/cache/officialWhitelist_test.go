package cache

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

func TestOfficialWhiteListAndScheduledRecipientsRoundTrip(t *testing.T) {
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	c := NewRedisFeedEventCache(r)
	ctx := context.Background()
	list := domain.OfficialWhiteList{Strategy: "script", StudentIDs: []string{"a", "b"}, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	if err := c.SetOfficialWhiteList(ctx, list); err != nil {
		t.Fatal(err)
	}
	got, err := c.GetOfficialWhiteList(ctx)
	if err != nil || got.Strategy != "script" || len(got.StudentIDs) != 2 {
		t.Fatalf("list=%+v error=%v", got, err)
	}
	mr.FastForward(8 * 24 * time.Hour)
	if _, err := c.GetOfficialWhiteList(ctx); err != nil {
		t.Fatal("expired marker disappeared")
	}
	msg := MuxiOfficialMSG{MuxiMSGId: "scheduled", Title: "survey", Targeted: true, StudentIDs: []string{"a", "b"}}
	if err := c.SetMuxiFeeds(ctx, msg, time.Now().Add(-time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	plans, err := c.GetMuxiToBePublicFeeds(ctx, true)
	if err != nil || len(plans) != 1 || !plans[0].Targeted || len(plans[0].StudentIDs) != 2 {
		t.Fatalf("plans=%+v error=%v", plans, err)
	}
}
