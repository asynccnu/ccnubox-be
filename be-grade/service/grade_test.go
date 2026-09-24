package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/crawler"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"go.uber.org/zap"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestReserveGradeRefreshThrottlesByStudent(t *testing.T) {
	service := &gradeService{
		nextRefresh:     make(map[string]time.Time),
		refreshInterval: time.Minute,
	}
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)

	if !service.reserveGradeRefresh("student-a", now) {
		t.Fatal("first refresh was not reserved")
	}
	if service.reserveGradeRefresh("student-a", now.Add(30*time.Second)) {
		t.Fatal("refresh inside interval was reserved")
	}
	if !service.reserveGradeRefresh("student-b", now.Add(30*time.Second)) {
		t.Fatal("different student's refresh was throttled")
	}
	if !service.reserveGradeRefresh("student-a", now.Add(time.Minute)) {
		t.Fatal("refresh after interval was not reserved")
	}
}

type cachedGradeDAO struct {
	findCalls int
}

func (d *cachedGradeDAO) FirstOrCreate(context.Context, *model.Grade) error {
	return nil
}

func (d *cachedGradeDAO) FindGrades(context.Context, string, int64, int64) ([]model.Grade, error) {
	d.findCalls++
	return []model.Grade{{StudentId: "cached"}}, nil
}

func (d *cachedGradeDAO) BatchInsertOrUpdate(context.Context, []model.Grade, bool) ([]model.Grade, error) {
	return nil, nil
}

func (d *cachedGradeDAO) GetDistinctGradeType(context.Context, string) ([]string, error) {
	return nil, nil
}

func TestGetGradeByTermRefreshDoesNotHideFetchErrorWithCache(t *testing.T) {
	dao := &cachedGradeDAO{}
	service := &gradeService{
		gradeDAO: dao,
		l:        zapx.NewZapLogger(zap.NewNop()),
	}

	grades, err := service.GetGradeByTerm(context.Background(), &domain.GetGradeByTermReq{
		StudentID: "invalid-student-id",
		Refresh:   true,
	})

	if err == nil {
		t.Fatal("GetGradeByTerm() error = nil, want fetch error")
	}
	if !gradev1.IsGetGradeError(err) {
		t.Fatalf("GetGradeByTerm() error = %v, want GetGradeError", err)
	}
	if grades != nil {
		t.Fatalf("GetGradeByTerm() grades = %#v, want nil", grades)
	}
	if dao.findCalls != 0 {
		t.Fatalf("fallback FindGrades() calls = %d, want 0", dao.findCalls)
	}
}

func TestAggregateGradePreservesDetailRecordID(t *testing.T) {
	grades := aggregateGrade([]crawler.Grade{{
		CJ0708ID: "grade-record-id",
		XS0101ID: "student-id",
		JX0404ID: "class-id",
	}}, map[string]crawler.Score{})

	if len(grades) != 1 {
		t.Fatalf("aggregateGrade() count = %d, want 1", len(grades))
	}
	if grades[0].KcId != "grade-record-id" {
		t.Fatalf("aggregateGrade() KcId = %q, want %q", grades[0].KcId, "grade-record-id")
	}
}

func TestUndergraduateStudentFetchesOnlyGradeList(t *testing.T) {
	requestCount := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"code":0,"msg":"success","data":[{"cj0708id":"grade-id","xs0101id":"student-id","jx0404id":"class-id","xqmc":"2025-2026-1"}]}`,
			)),
			Request: req,
		}, nil
	})}
	ug, err := crawler.NewUnderGrad(client)
	if err != nil {
		t.Fatalf("NewUnderGrad() error = %v", err)
	}

	grades, err := (&UndergraduateStudent{ug: ug}).GetGrades(context.Background(), "unused-cookie", 0, 0, 300)
	if err != nil {
		t.Fatalf("GetGrades() error = %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("HTTP request count = %d, want 1 list request", requestCount)
	}
	if len(grades) != 1 || grades[0].KcId != "grade-id" {
		t.Fatalf("GetGrades() grades = %#v", grades)
	}
	if grades[0].RegularGradePercent != RegularGradePercentMSG || grades[0].FinalGradePercent != FinalGradePercentMAG {
		t.Fatalf("detail placeholders not set: %#v", grades[0])
	}
}

type detailGradeDAO struct {
	dao.GradeDAO
	grades []model.Grade
	writes []model.Grade
}

func (d *detailGradeDAO) FindGrades(context.Context, string, int64, int64) ([]model.Grade, error) {
	return append([]model.Grade(nil), d.grades...), nil
}
func (d *detailGradeDAO) BatchInsertOrUpdate(_ context.Context, grades []model.Grade, _ bool) ([]model.Grade, error) {
	for _, grade := range grades {
		d.writes = append(d.writes, grade)
		for i := range d.grades {
			if d.grades[i].JxbId == grade.JxbId {
				d.grades[i] = grade
			}
		}
	}
	return grades, nil
}

func TestUpdateDetailScorePropagatesPartialFailureAndResumes(t *testing.T) {
	d := &detailGradeDAO{grades: []model.Grade{
		{StudentId: "s", JxbId: "a", Cj: 90, ChangeVersion: 5, RegularGradePercent: RegularGradePercentMSG, FinalGradePercent: FinalGradePercentMAG},
		{StudentId: "s", JxbId: "b", Cj: 80, ChangeVersion: 2, RegularGradePercent: RegularGradePercentMSG, FinalGradePercent: FinalGradePercentMAG},
	}}
	s := &gradeService{gradeDAO: d, l: zapx.NewZapLogger(zap.NewNop())}
	need := domain.NeedDetailGrade{StudentID: "s", Grades: []model.Grade{{JxbId: "a", Cj: 50, ChangeVersion: 1}, {JxbId: "b"}}}
	failure := errors.New("detail request failed")
	err := s.updateDetailScore(context.Background(), need, func(_ context.Context, grade model.Grade) (crawler.Score, error) {
		if grade.JxbId == "a" {
			if grade.Cj != 90 || grade.ChangeVersion != 5 {
				t.Fatalf("used stale message snapshot: %+v", grade)
			}
			return crawler.Score{Cjxm3: 90, Cjxm1: 90, Cjxm3bl: "30", Cjxm1bl: "70"}, nil
		}
		return crawler.Score{}, failure
	})
	if !errors.Is(err, failure) || len(d.writes) != 1 {
		t.Fatalf("err=%v writes=%v", err, d.writes)
	}
	calls := 0
	err = s.updateDetailScore(context.Background(), need, func(_ context.Context, grade model.Grade) (crawler.Score, error) {
		calls++
		if grade.JxbId != "b" {
			t.Fatal("retried already completed detail")
		}
		return crawler.Score{Cjxm3: 80, Cjxm1: 80, Cjxm3bl: "30", Cjxm1bl: "70"}, nil
	})
	if err != nil || calls != 1 || len(d.writes) != 2 {
		t.Fatalf("err=%v calls=%d writes=%v", err, calls, d.writes)
	}
}

func TestDetailFailuresAreClassifiedWithoutDroppingTransientCourses(t *testing.T) {
	for _, tc := range []struct {
		name      string
		err       error
		permanent bool
	}{
		{"password", userv1.ErrorIncorrectPasswordError("账号密码错误"), true},
		{"initialization", tool.ErrCCNUAccountInitializationRequired, true},
		{"parse", crawler.ErrDetailParse, true},
		{"expired after refresh", crawler.ErrCookieTimeout, true},
		{"network", errors.New("connection reset"), false},
		{"timeout", context.DeadlineExceeded, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &detailGradeDAO{grades: []model.Grade{{StudentId: "s", JxbId: "a", RegularGradePercent: RegularGradePercentMSG}}}
			s := &gradeService{gradeDAO: d, l: zapx.NewZapLogger(zap.NewNop())}
			need := domain.NeedDetailGrade{StudentID: "s", Grades: d.grades}
			err := s.updateDetailScore(context.Background(), need, func(context.Context, model.Grade) (crawler.Score, error) { return crawler.Score{}, tc.err })
			if !errors.Is(err, tc.err) || saramax.IsPermanent(err) != tc.permanent {
				t.Fatalf("err=%v permanent=%v", err, saramax.IsPermanent(err))
			}
		})
	}
	d := &detailGradeDAO{grades: []model.Grade{
		{StudentId: "s", JxbId: "a", RegularGradePercent: RegularGradePercentMSG},
		{StudentId: "s", JxbId: "b", RegularGradePercent: RegularGradePercentMSG},
	}}
	s := &gradeService{gradeDAO: d, l: zapx.NewZapLogger(zap.NewNop())}
	err := s.updateDetailScore(context.Background(), domain.NeedDetailGrade{StudentID: "s", Grades: d.grades}, func(_ context.Context, g model.Grade) (crawler.Score, error) {
		if g.JxbId == "a" {
			return crawler.Score{}, crawler.ErrDetailParse
		}
		return crawler.Score{}, context.DeadlineExceeded
	})
	if saramax.IsPermanent(err) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("mixed course failures must remain retryable: %v", err)
	}
}
