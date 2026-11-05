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

type refreshFeedClient struct{ feedv1.FeedServiceClient }

func (refreshFeedClient) PublicFeedEvent(context.Context, *feedv1.PublicFeedEventReq, ...grpc.CallOption) (*feedv1.PublicFeedEventResp, error) {
	panic("BFF must not publish grade notifications")
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

func TestGradeRefreshOnlyBoostsAndNeverPublishes(t *testing.T) {
	for _, name := range []string{"class query failure", "empty classmates", "nil response", "boost failure", "success"} {
		t.Run(name, func(t *testing.T) {
			queried, boosted := 0, 0
			h := &TieredHandler{
				gradeClient: refreshGradeClient{grades: []*gradev1.Grade{
					{JxbId: ""}, {JxbId: " "}, {JxbId: "课程20201", Kcmc: "课程", Xnm: 2020, Xqm: 1},
					{JxbId: "real", Cj: 0, Xnm: 2020}, {JxbId: "real", Cj: 90}, nil,
				}},
				feedClient: refreshFeedClient{}, l: nopLogger{},
			}
			h.classList = refreshClassClient{find: func(ctx context.Context, req *classlistv1.GetStuIdByJxbIdRequest) (*classlistv1.GetStuIdByJxbIdResponse, error) {
				queried++
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 2*time.Second || req.JxbId != "real" {
					t.Fatalf("unbounded or fake class query: %v %+v", deadline, req)
				}
				switch name {
				case "class query failure":
					return nil, errors.New("class unavailable")
				case "empty classmates":
					return &classlistv1.GetStuIdByJxbIdResponse{}, nil
				case "nil response":
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
				t.Fatal(err)
			}
			want := 0
			if name == "boost failure" || name == "success" {
				want = 1
			}
			if queried != 1 || boosted != want {
				t.Fatalf("queried=%d boosted=%d", queried, boosted)
			}
		})
	}
}

func TestGradeRefreshReturnsFetchError(t *testing.T) {
	failure := errors.New("grade unavailable")
	h := &TieredHandler{gradeClient: refreshGradeClient{err: failure}, feedClient: refreshFeedClient{}}
	if err := h.gradeRefresh(context.Background(), "student"); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}
