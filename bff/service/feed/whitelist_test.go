package feed

import (
	"context"
	"errors"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/asynccnu/ccnubox-be/bff/service/activity"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"os"
	"strings"
	"testing"
	"time"
)

type whiteListFeedStub struct {
	feedv1.FeedServiceClient
	req *feedv1.SetOfficialWhiteListReq
}

func (f *whiteListFeedStub) SetOfficialWhiteList(_ context.Context, r *feedv1.SetOfficialWhiteListReq, _ ...grpc.CallOption) (*feedv1.SetOfficialWhiteListResp, error) {
	f.req = r
	return &feedv1.SetOfficialWhiteListResp{Strategy: r.Strategy, UserCount: int32(len(r.StudentIds)), ExpiresAt: r.ExpiresAt}, nil
}
func TestHighActivityStrategyUsesBackendRules(t *testing.T) {
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, activity.China)
	mr.SetTime(now)
	ctx := context.Background()
	f := &whiteListFeedStub{}
	s := NewWhiteListService(r, f, WhiteListOptions{})
	s.now = func() time.Time { return now }
	if _, err := s.Set(ctx, "high_activity"); !errors.Is(err, activity.ErrNotReady) {
		t.Fatalf("missing history: %v", err)
	}
	if err := activity.EnsureDay(ctx, r, now.AddDate(0, 0, -8)); err != nil {
		t.Fatal(err)
	}
	for day := 1; day <= 7; day++ {
		for range 3 {
			if err := activity.Record(ctx, r, "daily", now.AddDate(0, 0, -day)); err != nil {
				t.Fatal(err)
			}
		}
		if day > 1 {
			if err := activity.Record(ctx, r, "six_days", now.AddDate(0, 0, -day)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := activity.Record(ctx, r, "today", now); err != nil {
		t.Fatal(err)
	}
	result, err := s.Set(ctx, "high_activity")
	if err != nil || result.UserCount != 1 || f.req.StudentIds[0] != "daily" || f.req.All {
		t.Fatalf("result=%+v request=%+v err=%v", result, f.req, err)
	}
	s.options.MinActiveDays = 6
	result, err = s.Set(ctx, "high_activity")
	if err != nil || result.UserCount != 2 {
		t.Fatalf("configured threshold: %+v %v", result, err)
	}
	if _, err = s.Set(ctx, "manual"); !errors.Is(err, ErrInvalidStrategy) {
		t.Fatalf("manual must not accept recipients: %v", err)
	}
	if _, err = s.Set(ctx, "all"); err != nil || !f.req.All || len(f.req.StudentIds) != 0 {
		t.Fatalf("explicit broadcast: %+v %v", f.req, err)
	}
}
func TestWhiteListScriptHelper(t *testing.T) {
	for i, arg := range os.Args {
		if arg == "--whitelist-helper" {
			fmt.Print(os.Args[i+1])
			os.Exit(0)
		}
	}
}
func TestScriptStrategyResolvesBackendRecipients(t *testing.T) {
	f := &whiteListFeedStub{}
	command := []string{os.Args[0], "-test.run=TestWhiteListScriptHelper", "--", "--whitelist-helper", `["b","a","b"]`}
	s := NewWhiteListService(nil, f, WhiteListOptions{ScriptCommand: command})
	result, err := s.Set(context.Background(), "script")
	if err != nil || result.UserCount != 2 || strings.Join(f.req.StudentIds, ",") != "a,b" {
		t.Fatalf("script selection: %+v %v", f.req, err)
	}
	for _, output := range []string{`[]`, `null`, `{}`, `[" a"]`, `["a"] {}`} {
		s.options.ScriptCommand = append(append([]string(nil), command[:len(command)-1]...), output)
		f.req = nil
		if _, err := s.Set(context.Background(), "script"); err == nil || f.req != nil {
			t.Fatalf("invalid script output accepted: %s", output)
		}
	}
	var limited limitedOutput
	if _, err := limited.Write([]byte(strings.Repeat("a", (1<<20)+1))); err == nil {
		t.Fatal("unbounded output")
	}
	s.options.ScriptCommand = nil
	if _, err := s.Set(context.Background(), "script"); err == nil {
		t.Fatal("unconfigured script accepted")
	}
}
