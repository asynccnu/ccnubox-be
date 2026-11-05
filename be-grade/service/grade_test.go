package service

import (
	"context"
	"errors"
	"fmt"
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

func (d *cachedGradeDAO) FindGrades(context.Context, string, int64, int64) ([]model.Grade, error) {
	d.findCalls++
	return []model.Grade{{StudentId: "cached"}}, nil
}

func (d *cachedGradeDAO) SyncGrades(context.Context, dao.GradeSnapshot) (dao.GradeSyncResult, error) {
	return dao.GradeSyncResult{}, nil
}

func (d *cachedGradeDAO) UpdateGradeDetails(context.Context, []model.Grade) error { return nil }

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
	grades, err := aggregateGrade([]crawler.Grade{{
		CJ0708ID: "grade-record-id",
		XQMC:     "2020-2021-1",
		XS0101ID: "student-id",
		JX0404ID: "class-id",
	}}, map[string]crawler.Score{})

	if err != nil || len(grades) != 1 {
		t.Fatalf("aggregateGrade() count = %d err=%v", len(grades), err)
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
				`{"code":0,"msg":"success","total":1,"data":[{"cj0708id":"grade-id","xs0101id":"student-id","jx0404id":"class-id","xqmc":"2025-2026-1","zcj":80}]}`,
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
func (d *detailGradeDAO) UpdateGradeDetails(_ context.Context, grades []model.Grade) error {
	for _, grade := range grades {
		d.writes = append(d.writes, grade)
		for i := range d.grades {
			if d.grades[i].JxbId == grade.JxbId {
				d.grades[i] = grade
			}
		}
	}
	return nil
}

func TestUpdateDetailScorePropagatesPartialFailureAndResumes(t *testing.T) {
	d := &detailGradeDAO{grades: []model.Grade{
		{StudentId: "s", JxbId: "a", Cj: 90, ChangeVersion: 5, RegularGradePercent: RegularGradePercentMSG, FinalGradePercent: FinalGradePercentMAG},
		{StudentId: "s", JxbId: "b", Cj: 80, ChangeVersion: 2, RegularGradePercent: RegularGradePercentMSG, FinalGradePercent: FinalGradePercentMAG},
	}}
	s := &gradeService{snapshotVerified: true, gradeDAO: d, l: zapx.NewZapLogger(zap.NewNop())}
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
			s := &gradeService{snapshotVerified: true, gradeDAO: d, l: zapx.NewZapLogger(zap.NewNop())}
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
	s := &gradeService{snapshotVerified: true, gradeDAO: d, l: zapx.NewZapLogger(zap.NewNop())}
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

func (d *gradeSyncDAO) SyncGrades(_ context.Context, snapshot dao.GradeSnapshot) (dao.GradeSyncResult, error) {
	if !snapshot.Complete || snapshot.StudentID == "" || snapshot.SourceSystem == "" || snapshot.ObservedAt.IsZero() {
		return dao.GradeSyncResult{}, errors.New("incomplete snapshot")
	}
	grades := snapshot.Grades
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
			s := &gradeService{snapshotVerified: true,
				gradeDAO: d, l: zapx.NewZapLogger(zap.NewNop()), userClient: gradeCookieClient{}, producer: producer,
				proxyClient: gradeProxyClient{body: `{"code":0,"total":1,"data":[{"xs0101id":"2023212345","jx0404id":"class","xqmc":"2020-2021-1","zcj":0}]}`},
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
		s := &gradeService{snapshotVerified: true,
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
		s := &gradeService{snapshotVerified: true,
			gradeDAO: d, l: zapx.NewZapLogger(zap.NewNop()), userClient: gradeCookieClient{},
			proxyClient: gradeProxyClient{body: `{"code":0,"total":1,"data":[{"xs0101id":"2023212345","jx0404id":"class"` + fields + `}]}`},
		}
		if _, err := s.GetUpdateScore(context.Background(), "2023212345"); err == nil || d.syncCalls != 0 {
			t.Fatalf("fields=%s err=%v sync calls=%d", fields, err, d.syncCalls)
		}
	}
}

func TestConvertGraduateGradeRejectsInvalidScores(t *testing.T) {
	for _, score := range []string{"", "优秀", "未发布", "1e40", "NaN", "+Inf"} {
		grades, err := ConvertGraduateGrade([]crawler.GraduatePoints{{Cj: "80", Xnm: "2020", Xqm: "3"}, {Cj: score, Xnm: "2020", Xqm: "3"}})
		if err == nil || len(grades) != 0 {
			t.Fatalf("score %q converted to grades=%+v err=%v", score, grades, err)
		}
	}
	grades, err := ConvertGraduateGrade([]crawler.GraduatePoints{{Cj: "0", Xnm: "2020", Xqm: "3"}, {Cj: "89.25", Xnm: "2021", Xqm: "12"}})
	if err != nil || len(grades) != 2 || grades[0].Cj != 0 || grades[0].Xqm != 1 || grades[1].Cj != 89.25 || grades[1].Xqm != 2 {
		t.Fatalf("valid graduate scores: grades=%+v err=%v", grades, err)
	}
}

func TestActiveGradeQueryPersistsOutboxWithoutBFF(t *testing.T) {
	const student = "2023212345"
	db := newGradeServiceTestDB(t)
	if err := dao.InitTables(db); err != nil {
		t.Fatal(err)
	}
	s := &gradeService{snapshotVerified: true, gradeDAO: dao.NewGradeDAO(db), l: zapx.NewZapLogger(zap.NewNop()), userClient: gradeCookieClient{}, producer: &failingDetailProducer{}}
	for _, score := range []int{80, 90} {
		s.proxyClient = gradeProxyClient{body: fmt.Sprintf(`{"code":0,"total":1,"data":[{"xs0101id":"%s","jx0404id":"class","xqmc":"2020-2021-1","zcj":%d}]}`, student, score)}
		if grades, err := s.GetGradeByTerm(context.Background(), &domain.GetGradeByTermReq{StudentID: student, Refresh: true}); err != nil || len(grades) != 1 || grades[0].Cj != float32(score) {
			t.Fatal(grades, err)
		}
	}
	var count int64
	if err := db.Model(&model.GradeNotificationOutbox{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatal(count, err)
	}
	if grades, err := s.GetUpdateScore(context.Background(), student); err != nil || len(grades) != 0 {
		t.Fatal(grades, err)
	}
	if err := db.Model(&model.GradeNotificationOutbox{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatal(count, err)
	}
}

func TestActiveGradeQueryLazilyActivatesLegacyScore(t *testing.T) {
	const student = "2023212345"
	db := newGradeServiceTestDB(t)
	if err := dao.InitTables(db); err != nil {
		t.Fatal(err)
	}
	legacy := model.Grade{StudentId: student, JxbId: "class", Cj: 80, Jd: calcJd(80), Xnm: 2020, Xqm: 1, Kcbj: "是否辅修字段暂时缺失", ChangeVersion: 19}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	s := &gradeService{snapshotVerified: true, gradeDAO: dao.NewGradeDAO(db), l: zapx.NewZapLogger(zap.NewNop()), userClient: gradeCookieClient{}, producer: &failingDetailProducer{}}
	for i, score := range []int{80, 90, 90} {
		s.proxyClient = gradeProxyClient{body: fmt.Sprintf(`{"code":0,"total":1,"data":[{"xs0101id":"%s","jx0404id":"class","xqmc":"2020-2021-1","zcj":%d}]}`, student, score)}
		got, err := s.GetUpdateScore(context.Background(), student)
		if err != nil {
			t.Fatal(err)
		}
		if i == 1 {
			if len(got) != 1 || got[0].ScoreVersion != 1 || got[0].ChangeVersion != 20 {
				t.Fatal(got)
			}
		} else if len(got) != 0 {
			t.Fatal(got)
		}
		if i == 0 {
			stored, err := s.gradeDAO.FindGrades(context.Background(), student, 0, 0)
			if err != nil || len(stored) != 1 || stored[0].NotificationID != nil || stored[0].ScoreVersion != 0 || stored[0].ChangeVersion != 19 {
				t.Fatal(stored, err)
			}
		}
	}
	var count int64
	if err := db.Model(&model.GradeNotificationOutbox{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatal(count, err)
	}
}

func TestCompleteSnapshotRequiresExplicitProtocolVerification(t *testing.T) {
	d := &gradeSyncDAO{}
	// 默认配置不能因第一页恰好较短或空列表就建立首次基线；也不请求 Cookie。
	s := NewGradeService(d, zapx.NewZapLogger(zap.NewNop()), nil, nil, nil, nil, nil).(*gradeService)
	if _, err := s.fetchGradesWithSingleFlight(context.Background(), "2023212345"); err == nil || !strings.Contains(err.Error(), "snapshotVerified") || d.syncCalls != 0 {
		t.Fatalf("gate bypassed: err=%v calls=%d", err, d.syncCalls)
	}
}
