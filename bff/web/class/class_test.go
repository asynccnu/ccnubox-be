package class

import (
	"context"
	"errors"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	b_errorx "github.com/asynccnu/ccnubox-be/bff/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	cs "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classService/v1"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func TestMapGetClassListError(t *testing.T) {
	tests := []struct {
		name       string
		origin     error
		wantCode   int
		wantStatus int
	}{
		{
			name:       "account initialization required",
			origin:     classlistv1.ErrorCCNULoginError(tool.CCNUAccountInitializationRequiredMarker),
			wantCode:   errs.CCNU_ACCOUNT_INITIALIZATION_REQUIRED_ERROR_CODE,
			wantStatus: 409,
		},
		{
			name:       "unexpected class error",
			origin:     errors.New("database unavailable"),
			wantCode:   errs.GET_CLASS_LIST_ERROR_CODE,
			wantStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := mapGetClassListError(tt.origin)
			var got *b_errorx.CustomError
			if !errors.As(wrapped, &got) {
				t.Fatal("wrapped error does not contain a CustomError")
			}
			if got.Code != tt.wantCode {
				t.Errorf("code = %d, want %d", got.Code, tt.wantCode)
			}
			if got.HttpCode != tt.wantStatus {
				t.Errorf("HTTP status = %d, want %d", got.HttpCode, tt.wantStatus)
			}
		})
	}
}

func TestWrapClassMutationError(t *testing.T) {
	tests := []struct {
		name       string
		origin     error
		wantCode   int
		wantStatus int
	}{
		{name: "schedule conflict", origin: classlistv1.ErrorErrClassScheduleConflict("conflict"), wantCode: errs.CLASS_SCHEDULE_CONFLICT_ERROR_CODE, wantStatus: 409},
		{name: "already exists", origin: classlistv1.ErrorClassisexist("exists"), wantCode: errs.CLASS_ALREADY_EXISTS_ERROR_CODE, wantStatus: 409},
		{name: "invalid parameter", origin: classlistv1.ErrorParamErr("invalid"), wantCode: errs.BAD_ENTITY_ERROR_CODE, wantStatus: 422},
		{name: "unexpected service error", origin: errors.New("database unavailable"), wantCode: errs.ADD_CLASS_ERROR_CODE, wantStatus: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := wrapClassMutationError(tt.origin, errs.ADD_CLASS_ERROR)
			var got *b_errorx.CustomError
			if !errors.As(wrapped, &got) {
				t.Fatal("wrapped error does not contain a CustomError")
			}
			if got.Code != tt.wantCode {
				t.Errorf("code = %d, want %d", got.Code, tt.wantCode)
			}
			if got.HttpCode != tt.wantStatus {
				t.Errorf("HTTP status = %d, want %d", got.HttpCode, tt.wantStatus)
			}
		})
	}
}

func TestClassHandler_ConvertWeek(t *testing.T) {
	type args struct {
		weeks []int
	}
	tests := []struct {
		name string
		c    *ClassHandler
		args args
		want int64
	}{
		{
			name: "Single week",
			c:    &ClassHandler{},
			args: args{weeks: []int{1}},
			want: 1,
		},
		{
			name: "Multiple weeks",
			c:    &ClassHandler{},
			args: args{weeks: []int{1, 2, 3}},
			want: 7,
		},
		{
			name: "Non-consecutive weeks",
			c:    &ClassHandler{},
			args: args{weeks: []int{1, 3, 5}},
			want: 21,
		},
		{
			name: "Weeks out of range",
			c:    &ClassHandler{},
			args: args{weeks: []int{0, 31}},
			want: 0,
		},
		{
			name: "Mixed valid and invalid weeks",
			c:    &ClassHandler{},
			args: args{weeks: []int{1, 0, 3, 31}},
			want: 5,
		},
		{
			name: "Empty weeks",
			c:    &ClassHandler{},
			args: args{weeks: []int{}},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertWeekFromArrayToInt(tt.args.weeks); got != tt.want {
				t.Errorf("ClassHandler.convertWeekFromArrayToInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertWeekFromIntToArray(t *testing.T) {
	type args struct {
		weeks int64
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		{
			name: "Single week",
			args: args{weeks: 1},
			want: []int{1},
		},
		{
			name: "Multiple weeks",
			args: args{weeks: 7},
			want: []int{1, 2, 3},
		},
		{
			name: "Non-consecutive weeks",
			args: args{weeks: 21},
			want: []int{1, 3, 5},
		},
		{
			name: "Weeks out of range",
			args: args{weeks: 0},
			want: nil,
		},
		{
			name: "Mixed valid and invalid weeks",
			args: args{weeks: 5},
			want: []int{1, 3},
		},
		{
			name: "Empty weeks",
			args: args{weeks: 0},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertWeekFromIntToArray(tt.args.weeks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertWeekFromIntToArray() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeClassServiceClient struct {
	searchReply    *cs.SearchReply
	searchErr      error
	studiedReply   *cs.GetClassToBeStudiedReply
	studiedErr     error
	lastStudiedReq *cs.GetClassToBeStudiedRequest
}

func (f *fakeClassServiceClient) SearchClass(ctx context.Context, in *cs.SearchRequest, opts ...grpc.CallOption) (*cs.SearchReply, error) {
	return f.searchReply, f.searchErr
}

func (f *fakeClassServiceClient) AddClass(ctx context.Context, in *cs.AddClassRequest, opts ...grpc.CallOption) (*cs.AddClassReply, error) {
	return nil, nil
}

func (f *fakeClassServiceClient) GetClassToBeStudied(ctx context.Context, in *cs.GetClassToBeStudiedRequest, opts ...grpc.CallOption) (*cs.GetClassToBeStudiedReply, error) {
	f.lastStudiedReq = in
	return f.studiedReply, f.studiedErr
}

func newTestGinContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	return ctx
}

func assertCustomErrorCode(t *testing.T, err error, wantCode, wantStatus int) {
	t.Helper()
	var got *b_errorx.CustomError
	if !errors.As(err, &got) {
		t.Fatalf("error %v does not contain a CustomError", err)
	}
	if got.Code != wantCode {
		t.Errorf("code = %d, want %d", got.Code, wantCode)
	}
	if got.HttpCode != wantStatus {
		t.Errorf("HTTP status = %d, want %d", got.HttpCode, wantStatus)
	}
}

func TestClassHandler_SearchClass(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fake := &fakeClassServiceClient{searchReply: &cs.SearchReply{ClassInfos: []*cs.ClassInfo{
			{
				Id: "c1", Day: 1, Teacher: "张老师", Where: "n101", ClassWhen: "1-2",
				WeekDuration: "1-2周", Classname: "高等数学", Credit: 4, Weeks: 3,
				Semester: "1", Year: "2025",
			},
		}}}
		h := &ClassHandler{ClassServiceClient: fake}

		resp, err := h.SearchClass(newTestGinContext(), SearchRequest{
			SearchKeyWords: "数学", Year: "2025", Semester: "1", Page: 1, PageSize: 20,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, ok := resp.Data.(SearchClassResp)
		if !ok {
			t.Fatalf("data type = %T, want SearchClassResp", resp.Data)
		}
		if len(data.ClassInfos) != 1 {
			t.Fatalf("classInfos length = %d, want 1", len(data.ClassInfos))
		}
		got := data.ClassInfos[0]
		if got.ID != "c1" || got.Classname != "高等数学" || got.Teacher != "张老师" {
			t.Errorf("unexpected class info: %+v", got)
		}
		if want := []int{1, 2}; !reflect.DeepEqual(got.Weeks, want) {
			t.Errorf("weeks = %v, want %v", got.Weeks, want)
		}
	})

	t.Run("invalid page", func(t *testing.T) {
		h := &ClassHandler{ClassServiceClient: &fakeClassServiceClient{}}
		_, err := h.SearchClass(newTestGinContext(), SearchRequest{
			SearchKeyWords: "数学", Year: "2025", Semester: "1", Page: 0, PageSize: 20,
		})
		assertCustomErrorCode(t, err, errs.INVALID_PARAM_VALUE_ERROR_CODE, 400)
	})

	t.Run("client error", func(t *testing.T) {
		h := &ClassHandler{ClassServiceClient: &fakeClassServiceClient{searchErr: errors.New("es unavailable")}}
		_, err := h.SearchClass(newTestGinContext(), SearchRequest{
			SearchKeyWords: "数学", Year: "2025", Semester: "1", Page: 1, PageSize: 20,
		})
		assertCustomErrorCode(t, err, errs.SEARCH_CLASS_ERROR_CODE, 500)
	})
}

func TestClassHandler_GetToBeStudiedClass(t *testing.T) {
	studiedReply := &cs.GetClassToBeStudiedReply{
		IdentityDevelop: []*cs.GetClassToBeStudiedReply_ClassToBeStudiedInfo{
			{Id: "b2", Name: "课程B", Status: "未修读", Property: "个性发展", Credit: "2", Studiable: "2025-2026-1"},
		},
		SpecificSkill: []*cs.GetClassToBeStudiedReply_ClassToBeStudiedInfo{
			{Id: "b1", Name: "课程A", Status: "修读中", Property: "专业主干", Credit: "3", Studiable: "2025-2026-1"},
			{Id: "a1", Name: "课程C", Status: "已修读", Property: "专业主干", Credit: "4", Studiable: "2024-2025-2"},
		},
		CommonEducate: []*cs.GetClassToBeStudiedReply_ClassToBeStudiedInfo{
			{Id: "c1", Name: "课程D", Status: "未修读", Property: "通识教育", Credit: "1", Studiable: "2025-2026-1"},
		},
	}

	t.Run("success and sorted", func(t *testing.T) {
		h := &ClassHandler{ClassServiceClient: &fakeClassServiceClient{studiedReply: studiedReply}}
		resp, err := h.GetToBeStudiedClass(newTestGinContext(), ijwt.UserClaims{StudentId: "2025211366"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, ok := resp.Data.(GetToBeStudiedClassResp)
		if !ok {
			t.Fatalf("data type = %T, want GetToBeStudiedClassResp", resp.Data)
		}
		if len(data.IdentityDevelop) != 1 || len(data.SpecificSkill) != 2 || len(data.CommonEducate) != 1 {
			t.Fatalf("unexpected lengths: identity=%d specific=%d common=%d",
				len(data.IdentityDevelop), len(data.SpecificSkill), len(data.CommonEducate))
		}
		if data.SpecificSkill[0].ID != "a1" || data.SpecificSkill[1].ID != "b1" {
			t.Errorf("specific skill not sorted by id: %+v", data.SpecificSkill)
		}
		if data.CommonEducate[0].Name != "课程D" {
			t.Errorf("unexpected common educate class: %+v", data.CommonEducate[0])
		}
	})

	t.Run("client error", func(t *testing.T) {
		h := &ClassHandler{ClassServiceClient: &fakeClassServiceClient{studiedErr: errors.New("db unavailable")}}
		_, err := h.GetToBeStudiedClass(newTestGinContext(), ijwt.UserClaims{StudentId: "2025211366"})
		assertCustomErrorCode(t, err, errs.GET_TO_BE_STUDIED_CLASS_ERROR_CODE, 500)
	})
}

func TestClassHandler_GetToBeStudiedClassByStatus(t *testing.T) {
	t.Run("status and student id passed through", func(t *testing.T) {
		fake := &fakeClassServiceClient{studiedReply: &cs.GetClassToBeStudiedReply{}}
		h := &ClassHandler{ClassServiceClient: fake}
		_, err := h.GetToBeStudiedClassByStatus(newTestGinContext(),
			GetToBeStudiedClassReq{Status: "未修读"}, ijwt.UserClaims{StudentId: "2025211366"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.lastStudiedReq.GetStatus() != "未修读" {
			t.Errorf("status = %q, want %q", fake.lastStudiedReq.GetStatus(), "未修读")
		}
		if fake.lastStudiedReq.GetStuId() != "2025211366" {
			t.Errorf("stu_id = %q, want %q", fake.lastStudiedReq.GetStuId(), "2025211366")
		}
	})

	t.Run("client error", func(t *testing.T) {
		h := &ClassHandler{ClassServiceClient: &fakeClassServiceClient{studiedErr: errors.New("db unavailable")}}
		_, err := h.GetToBeStudiedClassByStatus(newTestGinContext(),
			GetToBeStudiedClassReq{Status: "已修读"}, ijwt.UserClaims{StudentId: "2025211366"})
		assertCustomErrorCode(t, err, errs.GET_TO_BE_STUDIED_CLASS_ERROR_CODE, 500)
	})
}
