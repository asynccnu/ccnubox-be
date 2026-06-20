package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/asynccnu/ccnubox-be/be-grade/crawler"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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
