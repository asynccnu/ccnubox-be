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
	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

func (d *cachedGradeDAO) BatchInsertOrUpdate(context.Context, []model.Grade, bool) (dao.GradeSyncResult, error) {
	return dao.GradeSyncResult{}, nil
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
				`{"code":0,"msg":"success","data":[{"cj0708id":"grade-id","xs0101id":"student-id","jx0404id":"class-id","xqmc":"2025-2026-1","zcj":80}]}`,
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
func (d *detailGradeDAO) BatchInsertOrUpdate(_ context.Context, grades []model.Grade, _ bool) (dao.GradeSyncResult, error) {
	for _, grade := range grades {
		d.writes = append(d.writes, grade)
		for i := range d.grades {
			if d.grades[i].JxbId == grade.JxbId {
				d.grades[i] = grade
			}
		}
	}
	return dao.GradeSyncResult{Changed: grades}, nil
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

type gradeSyncDAO struct {
	dao.GradeDAO
	result     dao.GradeSyncResult
	final      []model.Grade
	synced     []model.Grade
	syncCalls  int
	findCalls  int
	cacheEmpty bool
}

func (d *gradeSyncDAO) BatchInsertOrUpdate(_ context.Context, grades []model.Grade, detail bool) (dao.GradeSyncResult, error) {
	if detail {
		return dao.GradeSyncResult{}, errors.New("unexpected detail sync")
	}
	d.syncCalls++
	d.synced = append([]model.Grade(nil), grades...)
	return d.result, nil
}

func (d *gradeSyncDAO) FindGrades(context.Context, string, int64, int64) ([]model.Grade, error) {
	d.findCalls++
	if d.cacheEmpty && d.findCalls == 1 {
		return nil, nil
	}
	return d.final, nil
}

type gradeCookieClient struct{ userv1.UserServiceClient }

func (gradeCookieClient) GetCookie(context.Context, *userv1.GetCookieRequest, ...grpc.CallOption) (*userv1.GetCookieResponse, error) {
	return &userv1.GetCookieResponse{Cookie: "test-cookie"}, nil
}

type gradeProxyClient struct{ body string }

func (c gradeProxyClient) NewProxyClient(...proxy.Option) *proxy.HttpClient {
	return &proxy.HttpClient{Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(c.body)), Request: req}, nil
	})}}
}

type failingDetailProducer struct {
	calls int
}

func (p *failingDetailProducer) SendMessage(context.Context, string, domain.NeedDetailGrade) error {
	p.calls++
	return errors.New("detail queue unavailable")
}
func (*failingDetailProducer) Close() error { return nil }

func TestGradeRefreshKeepsNotificationCandidatesSeparateFromQueries(t *testing.T) {
	const student = "2023212345"
	metadata := model.Grade{StudentId: student, JxbId: "metadata", Kcmc: "改名课程", Cj: 80, Xf: 2, Kcxzmc: "必修", ChangeVersion: 7}
	notification := model.Grade{StudentId: student, JxbId: "old-term-new-score", Kcmc: "旧学期新成绩", Xnm: 2020, Xqm: 1, Cj: 90, Xf: 3, Kcxzmc: "必修", ChangeVersion: 1, RegularGradePercent: RegularGradePercentMSG}
	unchanged := model.Grade{StudentId: student, JxbId: "unchanged", Kcmc: "未变课程", Cj: 70, Xf: 1, Kcxzmc: "必修", ChangeVersion: 2}
	for _, name := range []string{"notification candidates", "metadata only", "full grades", "full credit scores"} {
		t.Run(name, func(t *testing.T) {
			d := &gradeSyncDAO{
				result: dao.GradeSyncResult{Changed: []model.Grade{metadata, notification}, Notify: []model.Grade{notification}},
				final:  []model.Grade{metadata, notification, unchanged},
			}
			producer := &failingDetailProducer{}
			s := &gradeService{
				gradeDAO: d, l: zapx.NewZapLogger(zap.NewNop()), userClient: gradeCookieClient{}, producer: producer,
				proxyClient: gradeProxyClient{body: `{"code":0,"data":[{"xs0101id":"2023212345","jx0404id":"class","xqmc":"2020-2021-1","zcj":0}]}`},
			}
			switch name {
			case "notification candidates":
				grades, err := s.GetUpdateScore(context.Background(), student)
				if err != nil || len(grades) != 1 || grades[0].JxbId != notification.JxbId || grades[0].Xnm != 2020 {
					t.Fatalf("GetUpdateScore() = %+v, %v", grades, err)
				}
			case "metadata only":
				d.result.Notify = nil
				grades, err := s.GetUpdateScore(context.Background(), student)
				if err != nil || len(grades) != 0 {
					t.Fatalf("metadata returned as notification: %+v, %v", grades, err)
				}
			case "full grades":
				grades, err := s.GetGradeByTerm(context.Background(), &domain.GetGradeByTermReq{StudentID: student, Refresh: true})
				if err != nil || len(grades) != 3 {
					t.Fatalf("GetGradeByTerm() = %+v, %v", grades, err)
				}
			case "full credit scores":
				d.cacheEmpty = true
				scores, err := s.GetGradeScore(context.Background(), student)
				if err != nil || len(scores) != 1 || len(scores[0].GradeScoreList) != 3 {
					t.Fatalf("GetGradeScore() = %+v, %v", scores, err)
				}
			}
			if d.syncCalls != 1 || len(d.synced) != 1 || d.synced[0].Cj != 0 || producer.calls != 1 {
				t.Fatalf("sync calls=%d grades=%+v producer calls=%d", d.syncCalls, d.synced, producer.calls)
			}
		})
	}
}

func TestInvalidNumericTotalScoreNeverReachesGradeDAO(t *testing.T) {
	for _, item := range []string{`{"zcj":"优秀"}`, `{"zcj":1e40}`} {
		d := &gradeSyncDAO{}
		s := &gradeService{
			gradeDAO: d, l: zapx.NewZapLogger(zap.NewNop()), userClient: gradeCookieClient{},
			proxyClient: gradeProxyClient{body: `{"code":0,"data":[{"zcj":80},` + item + `]}`},
		}
		if grades, err := s.GetUpdateScore(context.Background(), "2023212345"); err == nil || len(grades) != 0 || d.syncCalls != 0 {
			t.Fatalf("invalid snapshot reached DAO: grades=%+v err=%v sync calls=%d", grades, err, d.syncCalls)
		}
	}
}

func TestGradeRefreshRejectsMissingOrAmbiguousScores(t *testing.T) {
	for _, fields := range []string{"", `,"zcj":null`, `,"zcj":0,"zcjstr":"缓考"`, `,"zcj":90,"zcjstr":"优秀"`} {
		d := &gradeSyncDAO{}
		s := &gradeService{
			gradeDAO: d, l: zapx.NewZapLogger(zap.NewNop()), userClient: gradeCookieClient{},
			proxyClient: gradeProxyClient{body: `{"code":0,"data":[{"xs0101id":"2023212345","jx0404id":"class"` + fields + `}]}`},
		}
		if _, err := s.GetUpdateScore(context.Background(), "2023212345"); err == nil || d.syncCalls != 0 {
			t.Fatalf("fields=%s err=%v sync calls=%d", fields, err, d.syncCalls)
		}
	}
}

func TestConvertGraduateGradePreservesExistingScoreFallback(t *testing.T) {
	// 保留既有转换失败回退为零分的行为，不因单条文本成绩拒绝整批数据。
	for _, score := range []string{"", "优秀", "未发布", "1e40"} {
		grades := ConvertGraduateGrade([]crawler.GraduatePoints{{Cj: "80"}, {Cj: score}})
		if len(grades) != 2 || grades[0].Cj != 80 || grades[1].Cj != 0 {
			t.Fatalf("score %q converted to grades=%+v", score, grades)
		}
	}
	grades := ConvertGraduateGrade([]crawler.GraduatePoints{{Cj: "0", Xnm: "2020", Xqm: "3"}, {Cj: "89.25", Xnm: "2021", Xqm: "12"}})
	if len(grades) != 2 || grades[0].Cj != 0 || grades[0].Xqm != 1 || grades[1].Cj != 89.25 || grades[1].Xqm != 2 {
		t.Fatalf("valid graduate scores: grades=%+v", grades)
	}
}
