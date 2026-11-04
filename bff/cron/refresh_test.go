package cron

import (
	"context"
	"errors"
	"testing"
	"time"

	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	"google.golang.org/grpc"
)

type refreshGradeClient struct {
	gradev1.GradeServiceClient
	grades []*gradev1.Grade
	err    error
}

func (c refreshGradeClient) GetUpdateScore(context.Context, *gradev1.GetUpdateScoreReq, ...grpc.CallOption) (*gradev1.GetUpdateScoreResp, error) {
	return &gradev1.GetUpdateScoreResp{Grades: c.grades}, c.err
}

type refreshFeedClient struct {
	feedv1.FeedServiceClient
	publish func(context.Context, *feedv1.PublicFeedEventReq) (*feedv1.PublicFeedEventResp, error)
}

func (c refreshFeedClient) PublicFeedEvent(ctx context.Context, req *feedv1.PublicFeedEventReq, _ ...grpc.CallOption) (*feedv1.PublicFeedEventResp, error) {
	return c.publish(ctx, req)
}

type refreshClassClient struct {
	classlistv1.ClasserClient
	find func(context.Context, *classlistv1.GetStuIdByJxbIdRequest) (*classlistv1.GetStuIdByJxbIdResponse, error)
}

func (c refreshClassClient) GetStuIdByJxbId(ctx context.Context, req *classlistv1.GetStuIdByJxbIdRequest, _ ...grpc.CallOption) (*classlistv1.GetStuIdByJxbIdResponse, error) {
	return c.find(ctx, req)
}

type refreshCounterClient struct {
	counterv1.CounterServiceClient
	boost func(context.Context, *counterv1.BoostScoresReq) (*counterv1.BoostScoresResp, error)
}

func (c refreshCounterClient) BoostScores(ctx context.Context, req *counterv1.BoostScoresReq, _ ...grpc.CallOption) (*counterv1.BoostScoresResp, error) {
	return c.boost(ctx, req)
}

func TestGradeRefreshPublishesBeforeOptionalBoost(t *testing.T) {
	for _, name := range []string{"class query failure", "empty classmates", "nil classmates response", "boost failure", "success"} {
		t.Run(name, func(t *testing.T) {
			grades := []*gradev1.Grade{
				{JxbId: "class", Kcmc: "旧学期成绩", Xnm: 2020, Xqm: 1, Kcxzmc: "专业课", Cj: 0, ChangeVersion: 8},
				{JxbId: "class", Kcmc: "另一门成绩", Xnm: 2025, Xqm: 2, Kcxzmc: "选修课", Cj: 90, ChangeVersion: 2},
			}
			published, queried, boosted := 0, 0, 0
			h := &TieredHandler{gradeClient: refreshGradeClient{grades: grades}, l: nopLogger{}}
			h.feedClient = refreshFeedClient{publish: func(_ context.Context, req *feedv1.PublicFeedEventReq) (*feedv1.PublicFeedEventResp, error) {
				if queried != 0 {
					t.Fatal("class query happened before all personal notifications")
				}
				g := grades[published]
				if req.StudentId != "student" || req.IsAll || req.Event.Type != feedv1.FeedEventType_GRADE || req.Event.DedupeKey != gradeDedupeKey("student", g.JxbId, g.Cj, g.ChangeVersion) {
					t.Fatalf("unexpected event: %+v", req)
				}
				wantURL := getGradeEventUrl(formatSemester(g.Xnm, g.Xqm), g.Kcxzmc)
				if req.Event.Url != wantURL || req.Event.ExtendFields["url"] != wantURL {
					t.Fatalf("event did not use its course term and nature: %+v", req.Event)
				}
				published++
				if published == 1 {
					return &feedv1.PublicFeedEventResp{Status: feedv1.PublishStatus_ACCEPTED}, nil
				}
				return &feedv1.PublicFeedEventResp{Status: feedv1.PublishStatus_DUPLICATE}, nil
			}}
			h.classList = refreshClassClient{find: func(ctx context.Context, req *classlistv1.GetStuIdByJxbIdRequest) (*classlistv1.GetStuIdByJxbIdResponse, error) {
				queried++
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 2*time.Second || req.JxbId != "class" || published != 2 {
					t.Fatalf("unbounded or premature class query: deadline=%v req=%+v published=%d", deadline, req, published)
				}
				switch name {
				case "class query failure":
					return nil, errors.New("class unavailable")
				case "empty classmates":
					return &classlistv1.GetStuIdByJxbIdResponse{}, nil
				case "nil classmates response":
					return nil, nil
				default:
					return &classlistv1.GetStuIdByJxbIdResponse{StuId: []string{"classmate"}}, nil
				}
			}}
			h.counter = refreshCounterClient{boost: func(ctx context.Context, req *counterv1.BoostScoresReq) (*counterv1.BoostScoresResp, error) {
				boosted++
				if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 2*time.Second || len(req.StudentIds) != 1 {
					t.Fatal("boost missing bounded context or classmates")
				}
				if name == "boost failure" {
					return nil, errors.New("boost unavailable")
				}
				return &counterv1.BoostScoresResp{}, nil
			}}
			if err := h.gradeRefresh(context.Background(), "student"); err != nil {
				t.Fatalf("optional boost failure leaked into result: %v", err)
			}
			wantBoosted := 0
			if name == "boost failure" || name == "success" {
				wantBoosted = 1
			}
			if published != 2 || queried != 1 || boosted != wantBoosted {
				t.Fatalf("published=%d queried=%d boosted=%d", published, queried, boosted)
			}
		})
	}
}

func TestGradeRefreshContinuesPublishingAndJoinsErrors(t *testing.T) {
	firstErr, thirdErr := errors.New("first publish failed"), errors.New("third publish failed")
	published, queried := 0, 0
	h := &TieredHandler{
		gradeClient: refreshGradeClient{grades: []*gradev1.Grade{{JxbId: "class"}, {JxbId: "class"}, {JxbId: "class"}}},
		l:           nopLogger{},
		feedClient: refreshFeedClient{publish: func(context.Context, *feedv1.PublicFeedEventReq) (*feedv1.PublicFeedEventResp, error) {
			published++
			switch published {
			case 1:
				return nil, firstErr
			case 3:
				return nil, thirdErr
			default:
				return &feedv1.PublicFeedEventResp{Status: feedv1.PublishStatus_ACCEPTED}, nil
			}
		}},
		classList: refreshClassClient{find: func(context.Context, *classlistv1.GetStuIdByJxbIdRequest) (*classlistv1.GetStuIdByJxbIdResponse, error) {
			queried++
			return nil, errors.New("class query failed")
		}},
	}
	err := h.gradeRefresh(context.Background(), "student")
	if !errors.Is(err, firstErr) || !errors.Is(err, thirdErr) || published != 3 || queried != 1 {
		t.Fatalf("err=%v published=%d queried=%d", err, published, queried)
	}
}

func TestGradeRefreshSkipsFakeClassIDsButStillNotifies(t *testing.T) {
	published, queried := 0, 0
	h := &TieredHandler{
		gradeClient: refreshGradeClient{grades: []*gradev1.Grade{
			{JxbId: ""}, {JxbId: " "}, {JxbId: "课程20201", Kcmc: "课程", Xnm: 2020, Xqm: 1},
			{JxbId: "real"}, {JxbId: "real"},
		}},
		l: nopLogger{},
		feedClient: refreshFeedClient{publish: func(context.Context, *feedv1.PublicFeedEventReq) (*feedv1.PublicFeedEventResp, error) {
			published++
			return &feedv1.PublicFeedEventResp{}, nil
		}},
		classList: refreshClassClient{find: func(_ context.Context, req *classlistv1.GetStuIdByJxbIdRequest) (*classlistv1.GetStuIdByJxbIdResponse, error) {
			queried++
			if req.JxbId != "real" {
				t.Fatalf("queried fake class: %q", req.JxbId)
			}
			return &classlistv1.GetStuIdByJxbIdResponse{}, nil
		}},
	}
	if err := h.gradeRefresh(context.Background(), "student"); err != nil || published != 5 || queried != 1 {
		t.Fatalf("err=%v published=%d queried=%d", err, published, queried)
	}
}

func TestGradeBoostHonorsParentCancellation(t *testing.T) {
	published, queryFinished := false, false
	h := &TieredHandler{
		gradeClient: refreshGradeClient{grades: []*gradev1.Grade{{JxbId: "class"}}},
		l:           nopLogger{},
		feedClient: refreshFeedClient{publish: func(context.Context, *feedv1.PublicFeedEventReq) (*feedv1.PublicFeedEventResp, error) {
			published = true
			return &feedv1.PublicFeedEventResp{}, nil
		}},
		classList: refreshClassClient{find: func(ctx context.Context, _ *classlistv1.GetStuIdByJxbIdRequest) (*classlistv1.GetStuIdByJxbIdResponse, error) {
			if !published {
				t.Fatal("boost blocked personal notification")
			}
			<-ctx.Done()
			queryFinished = true
			return nil, ctx.Err()
		}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := h.gradeRefresh(ctx, "student"); err != nil || !queryFinished {
		t.Fatalf("err=%v query finished=%v", err, queryFinished)
	}
}

func TestPublishGradeEventRejectsEmptyAndUnexpectedResponses(t *testing.T) {
	for _, resp := range []*feedv1.PublicFeedEventResp{nil, {Status: feedv1.PublishStatus_SUPPRESSED_BY_ALLOW_LIST}} {
		h := &TieredHandler{l: nopLogger{}, feedClient: refreshFeedClient{publish: func(context.Context, *feedv1.PublicFeedEventReq) (*feedv1.PublicFeedEventResp, error) {
			return resp, nil
		}}}
		if err := h.publishGradeEvent(context.Background(), "student", &gradev1.Grade{}); err == nil {
			t.Fatalf("unexpected publish response accepted: %+v", resp)
		}
	}
}
