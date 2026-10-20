package feed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/asynccnu/ccnubox-be/bff/service/activity"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/redis/go-redis/v9"
)

var ErrInvalidStrategy = errors.New("invalid white list strategy")
var ErrEmptyWhiteList = errors.New("the strategy selected no users")

type WhiteListOptions struct {
	WindowDays    int
	MinActiveDays int
	ScriptCommand []string
}
type WhiteListService struct {
	redis     redis.Cmdable
	feed      feedv1.FeedServiceClient
	options   WhiteListOptions
	now       func() time.Time
	selectors map[string]func(context.Context) ([]string, error)
}

func NewWhiteListService(r redis.Cmdable, f feedv1.FeedServiceClient, options WhiteListOptions) *WhiteListService {
	if options.WindowDays == 0 {
		options.WindowDays = 7
	}
	if options.MinActiveDays == 0 {
		options.MinActiveDays = 5
	}
	s := &WhiteListService{redis: r, feed: f, options: options, now: time.Now}
	s.selectors = map[string]func(context.Context) ([]string, error){"high_activity": s.highActivity, "script": s.fromScript}
	return s
}

// HTTP selects a backend strategy; it never supplies recipients or commands.
func (s *WhiteListService) Set(ctx context.Context, strategy string) (*feedv1.SetOfficialWhiteListResp, error) {
	req := &feedv1.SetOfficialWhiteListReq{Strategy: strategy, All: strategy == "all", ExpiresAt: s.now().Add(7 * 24 * time.Hour).Unix()}
	if !req.All {
		selector, ok := s.selectors[strategy]
		if !ok {
			return nil, ErrInvalidStrategy
		}
		ids, err := selector(ctx)
		if err != nil {
			return nil, err
		}
		ids, err = normalizeUsers(ids)
		if err != nil {
			return nil, err
		}
		req.StudentIds = ids
	}
	return s.feed.SetOfficialWhiteList(ctx, req)
}
func normalizeUsers(ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, ErrEmptyWhiteList
	}
	if len(ids) > 10000 {
		return nil, errors.New("strategy exceeds 10000 users")
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" || strings.TrimSpace(id) != id || !utf8.ValidString(id) || utf8.RuneCountInString(id) > 255 || strings.ContainsAny(id, "\x00\r\n\t ") {
			return nil, errors.New("strategy returned invalid student ID")
		}
		if !seen[id] {
			out = append(out, id)
			seen[id] = true
		}
	}
	sort.Strings(out)
	return out, nil
}
func (s *WhiteListService) highActivity(ctx context.Context) ([]string, error) {
	days, min := s.options.WindowDays, s.options.MinActiveDays
	if days < 1 || days > 30 || min < 1 || min > days {
		return nil, errors.New("invalid backend activity configuration")
	}
	return s.activeUsers(ctx, activity.Day(s.now()).AddDate(0, 0, -days), days, min)
}

type limitedOutput struct{ data bytes.Buffer }

func (b *limitedOutput) Write(p []byte) (int, error) {
	if b.data.Len()+len(p) > 1<<20 {
		return 0, errors.New("script output exceeds 1 MiB")
	}
	return b.data.Write(p)
}
func (s *WhiteListService) fromScript(ctx context.Context) ([]string, error) {
	command := s.options.ScriptCommand
	if len(command) == 0 || command[0] == "" {
		return nil, errors.New("white list script is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// Executable and arguments come only from trusted server configuration.
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.WaitDelay = time.Second
	var out limitedOutput
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("white list script failed: %w", err)
	}
	var ids []string
	d := json.NewDecoder(bytes.NewReader(out.data.Bytes()))
	if err := d.Decode(&ids); err != nil {
		return nil, errors.New("script must output a JSON string array")
	}
	if d.Decode(new(any)) != io.EOF {
		return nil, errors.New("script output contains trailing data")
	}
	return ids, nil
}
func (s *WhiteListService) activeUsers(ctx context.Context, start time.Time, days, minDays int) ([]string, error) {
	if err := activity.CheckWindow(ctx, s.redis, start, days); err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for i := 0; i < days; i++ {
		key := activity.Key(start.AddDate(0, 0, i))
		// Bound memory while reading daily sets and the union of all candidates.
		n, err := s.redis.SCard(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, activity.MissingDay(start, days, start.AddDate(0, 0, i))
		}
		if n > 100001 {
			return nil, errors.New("单日活跃候选人数超过 100000")
		}
		ids, err := s.redis.SMembers(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, activity.MissingDay(start, days, start.AddDate(0, 0, i))
		}
		for _, id := range ids {
			if id != "" {
				counts[id]++
			}
		}
		if len(counts) > 100000 {
			return nil, errors.New("活跃候选人数超过 100000")
		}
	}
	users := make([]string, 0)
	for id, n := range counts {
		if n >= minDays {
			users = append(users, id)
		}
	}
	if len(users) > 10000 {
		return nil, errors.New("筛选结果超过 10000 人，请缩小范围")
	}
	return users, nil
}
