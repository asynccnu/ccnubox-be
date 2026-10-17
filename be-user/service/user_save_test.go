package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	usercrypto "github.com/asynccnu/ccnubox-be/be-user/pkg/crypto"
	"github.com/asynccnu/ccnubox-be/be-user/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-user/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-user/repository/model"
	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type saveUserDAO struct {
	user        *model.User
	saveCalls   int
	deleteCalls int
	findErr     error
	deleteErr   error
	events      *[]string
}

func (d *saveUserDAO) FindByStudentId(context.Context, string) (*model.User, error) {
	if d.findErr != nil {
		return nil, d.findErr
	}
	if d.user == nil {
		return nil, dao.UserNotFound
	}
	copyOfUser := *d.user
	return &copyOfUser, nil
}

func (d *saveUserDAO) Save(_ context.Context, user *model.User) error {
	d.saveCalls++
	copyOfUser := *user
	d.user = &copyOfUser
	return nil
}

func (d *saveUserDAO) Delete(context.Context, string) error {
	d.deleteCalls++
	if d.events != nil {
		*d.events = append(*d.events, "dao")
	}
	if d.deleteErr != nil {
		return d.deleteErr
	}
	d.user = nil
	return nil
}

type deleteUserCache struct {
	deleteCalls int
	err         error
	events      *[]string
}

func (*deleteUserCache) GetCookie(context.Context, string, tool.XKSystem) (string, error) {
	return "", nil
}

func (*deleteUserCache) SetCookie(context.Context, string, string, tool.XKSystem) error {
	return nil
}

func (*deleteUserCache) GetLibraryToken(context.Context, string, string) (string, error) {
	return "", nil
}

func (*deleteUserCache) SetLibraryToken(context.Context, string, string, string) error {
	return nil
}

func (c *deleteUserCache) DeleteUserData(context.Context, string) error {
	c.deleteCalls++
	if c.events != nil {
		*c.events = append(*c.events, "cache")
	}
	return c.err
}

func TestSaveSkipsUnchangedPassword(t *testing.T) {
	cryptoClient, err := usercrypto.NewCrypto("muxiStudioSecret")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cryptoClient.Encrypt("password")
	if err != nil {
		t.Fatal(err)
	}
	userDAO := &saveUserDAO{user: &model.User{StudentId: "2024000000", Password: encrypted}}
	service := &userService{dao: userDAO, cryptoClient: cryptoClient}

	if err := service.Save(context.Background(), "2024000000", "password"); err != nil {
		t.Fatal(err)
	}
	if userDAO.saveCalls != 0 {
		t.Fatalf("Save() wrote unchanged password %d times", userDAO.saveCalls)
	}
}

func TestDeleteVerifiesPasswordAndRemovesUserData(t *testing.T) {
	cryptoClient, err := usercrypto.NewCrypto("muxiStudioSecret")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cryptoClient.Encrypt("password")
	if err != nil {
		t.Fatal(err)
	}
	userDAO := &saveUserDAO{user: &model.User{StudentId: "2024000000", Password: encrypted}}
	events := []string{}
	userDAO.events = &events
	userCache := &deleteUserCache{events: &events}
	service := &userService{dao: userDAO, cache: userCache, cryptoClient: cryptoClient}

	if err := service.Delete(context.Background(), "2024000000", "wrong"); err == nil {
		t.Fatal("Delete() accepted wrong password")
	}
	if userDAO.deleteCalls != 0 {
		t.Fatal("Delete() removed user after wrong password")
	}

	if err := service.Delete(context.Background(), "2024000000", "password"); err != nil {
		t.Fatal(err)
	}
	if userDAO.deleteCalls != 1 {
		t.Fatalf("Delete() DAO calls = %d, want 1", userDAO.deleteCalls)
	}
	if userCache.deleteCalls != 1 {
		t.Fatalf("Delete() cache calls = %d, want 1", userCache.deleteCalls)
	}
	if len(events) != 2 || events[0] != "cache" || events[1] != "dao" {
		t.Fatalf("Delete() operation order = %v, want [cache dao]", events)
	}
}

func TestDeleteClassifiesFindErrors(t *testing.T) {
	cryptoClient, err := usercrypto.NewCrypto("muxiStudioSecret")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		err  error
		want func(error) bool
	}{
		{name: "user not found", err: dao.UserNotFound, want: userv1.IsUserNotFoundError},
		{name: "database failure", err: errors.New("database unavailable"), want: userv1.IsDefaultDaoError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &userService{
				dao:          &saveUserDAO{findErr: tt.err},
				cryptoClient: cryptoClient,
			}
			if err := service.Delete(context.Background(), "2024000000", "password"); !tt.want(err) {
				t.Fatalf("Delete() error = %v, unexpected classification", err)
			}
		})
	}
}

func TestDeleteKeepsUserWhenCacheCleanupFails(t *testing.T) {
	cryptoClient, err := usercrypto.NewCrypto("muxiStudioSecret")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cryptoClient.Encrypt("password")
	if err != nil {
		t.Fatal(err)
	}
	userDAO := &saveUserDAO{user: &model.User{StudentId: "2024000000", Password: encrypted}}
	userCache := &deleteUserCache{err: errors.New("redis unavailable")}
	service := &userService{dao: userDAO, cache: userCache, cryptoClient: cryptoClient}

	err = service.Delete(context.Background(), "2024000000", "password")
	if !userv1.IsDefaultDaoError(err) {
		t.Fatalf("Delete() error = %v, want default DAO error", err)
	}
	if userDAO.deleteCalls != 0 {
		t.Fatalf("Delete() called DAO %d times after cache failure", userDAO.deleteCalls)
	}
	if userDAO.user == nil {
		t.Fatal("Delete() removed user after cache cleanup failure")
	}
}

type missingCookieCache struct{ deleteUserCache }

func (*missingCookieCache) GetCookie(context.Context, string, tool.XKSystem) (string, error) {
	return "", cache.ErrKeyNotFound
}

type failedCookieClient struct {
	ccnuv1.CCNUServiceClient
	err error
}

func (c failedCookieClient) GetXKCookie(context.Context, *ccnuv1.GetXKCookieRequest, ...grpc.CallOption) (*ccnuv1.GetXKCookieResponse, error) {
	return nil, c.err
}

func TestGetCookiePreservesPasswordClassification(t *testing.T) {
	cryptoClient, err := usercrypto.NewCrypto("muxiStudioSecret")
	if err != nil {
		t.Fatal(err)
	}
	password, err := cryptoClient.Encrypt("password")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		err           error
		passwordError bool
	}{
		{ccnuv1.ErrorInvalidSidOrPwd("账号密码错误"), true},
		{ccnuv1.ErrorCcnuserverError("服务不可用"), false},
	} {
		s := &userService{
			dao:   &saveUserDAO{user: &model.User{StudentId: "2024000000", Password: password}},
			cache: &missingCookieCache{}, cryptoClient: cryptoClient,
			ccnu: failedCookieClient{err: status.Convert(tc.err).Err()}, l: zapx.NewZapLogger(zap.NewNop()),
		}
		_, err := s.GetCookie(context.Background(), "2024000000", tool.XKPostgraduate)
		if err == nil || userv1.IsIncorrectPasswordError(status.Convert(err).Err()) != tc.passwordError {
			t.Fatalf("err=%v want password error=%v", err, tc.passwordError)
		}
	}
}

type cookieCacheStub struct {
	deleteUserCache
	get func(context.Context, string, tool.XKSystem) (string, error)
	set func(context.Context, string, string, tool.XKSystem) error
}

func (c *cookieCacheStub) GetCookie(ctx context.Context, sid string, system tool.XKSystem) (string, error) {
	return c.get(ctx, sid, system)
}
func (c *cookieCacheStub) SetCookie(ctx context.Context, sid, cookie string, system tool.XKSystem) error {
	return c.set(ctx, sid, cookie, system)
}

type cookieRPCStub struct {
	ccnuv1.CCNUServiceClient
	get   func(context.Context, *ccnuv1.GetXKCookieRequest) (*ccnuv1.GetXKCookieResponse, error)
	check func(context.Context, *ccnuv1.CheckXKCookieRequest) (*ccnuv1.CheckXKCookieResponse, error)
}

func (c cookieRPCStub) GetXKCookie(ctx context.Context, req *ccnuv1.GetXKCookieRequest, _ ...grpc.CallOption) (*ccnuv1.GetXKCookieResponse, error) {
	return c.get(ctx, req)
}
func (c cookieRPCStub) CheckXKCookie(ctx context.Context, req *ccnuv1.CheckXKCookieRequest, _ ...grpc.CallOption) (*ccnuv1.CheckXKCookieResponse, error) {
	return c.check(ctx, req)
}

func newCookieTestService(t *testing.T, sid string) *userService {
	t.Helper()
	cryptoClient, err := usercrypto.NewCrypto("muxiStudioSecret")
	if err != nil {
		t.Fatal(err)
	}
	password, err := cryptoClient.Encrypt("test-password")
	if err != nil {
		t.Fatal(err)
	}
	return &userService{dao: &saveUserDAO{user: &model.User{StudentId: sid, Password: password}}, cryptoClient: cryptoClient, l: zapx.NewZapLogger(zap.NewNop())}
}

func TestCookieCacheAndValidationFlow(t *testing.T) {
	for _, system := range []tool.XKSystem{tool.XKUndergraduate, tool.XKPostgraduate} {
		sid := "2024200000"
		if system == tool.XKPostgraduate {
			sid = "2024100000"
		}
		for _, tc := range []struct {
			name, cookie                string
			readErr, checkErr, writeErr error
			valid, wantErr, refresh     bool
		}{
			{name: "miss", readErr: cache.ErrKeyNotFound, refresh: true},
			{name: "valid", cookie: "cached-cookie", valid: true},
			{name: "expired", cookie: "cached-cookie", refresh: true},
			{name: "dependency error", cookie: "cached-cookie", checkErr: ccnuv1.ErrorCcnuserverError("unavailable"), wantErr: true},
			{name: "timeout", cookie: "cached-cookie", checkErr: context.DeadlineExceeded, wantErr: true},
			{name: "canceled", cookie: "cached-cookie", checkErr: context.Canceled, wantErr: true},
			{name: "redis read failure", readErr: errors.New("redis unavailable"), refresh: true},
			{name: "redis write failure", readErr: cache.ErrKeyNotFound, writeErr: errors.New("redis unavailable"), refresh: true},
		} {
			t.Run(string(system)+"/"+tc.name, func(t *testing.T) {
				s := newCookieTestService(t, sid)
				getCalls, checkCalls, writeCalls := 0, 0, 0
				s.cache = &cookieCacheStub{
					get: func(_ context.Context, gotSID string, gotSystem tool.XKSystem) (string, error) {
						if gotSID != sid || gotSystem != system {
							t.Error("wrong cache identity")
						}
						return tc.cookie, tc.readErr
					},
					set: func(ctx context.Context, gotSID, cookie string, gotSystem tool.XKSystem) error {
						writeCalls++
						deadline, ok := ctx.Deadline()
						if !ok || time.Until(deadline) > time.Second || ctx.Err() != nil {
							t.Error("cache write context must be bounded and live")
						}
						if gotSID != sid || gotSystem != system || cookie != "fresh-cookie" {
							t.Error("wrong cache write")
						}
						return tc.writeErr
					},
				}
				s.ccnu = cookieRPCStub{
					get: func(_ context.Context, req *ccnuv1.GetXKCookieRequest) (*ccnuv1.GetXKCookieResponse, error) {
						getCalls++
						if req.GetType() != string(system) || req.GetStudentId() != sid || req.GetPassword() != "test-password" {
							t.Error("wrong login contract")
						}
						return &ccnuv1.GetXKCookieResponse{Cookie: "fresh-cookie"}, nil
					},
					check: func(_ context.Context, req *ccnuv1.CheckXKCookieRequest) (*ccnuv1.CheckXKCookieResponse, error) {
						checkCalls++
						if req.GetType() != string(system) || req.GetCookie() != tc.cookie {
							t.Error("wrong check contract")
						}
						return &ccnuv1.CheckXKCookieResponse{Valid: tc.valid}, tc.checkErr
					},
				}
				cookie, err := s.GetCookie(context.Background(), sid, system)
				if (err != nil) != tc.wantErr || userv1.IsIncorrectPasswordError(err) {
					t.Fatalf("cookie=%q err=%v", cookie, err)
				}
				wantCalls := 0
				if tc.refresh {
					wantCalls = 1
				}
				if getCalls != wantCalls || writeCalls != wantCalls {
					t.Fatalf("login=%d write=%d want=%d", getCalls, writeCalls, wantCalls)
				}
				if (checkCalls == 1) != (tc.readErr == nil) {
					t.Fatalf("checkCalls=%d", checkCalls)
				}
				if err == nil {
					want := tc.cookie
					if tc.refresh {
						want = "fresh-cookie"
					}
					if cookie != want {
						t.Fatalf("cookie=%q want=%q", cookie, want)
					}
				}
			})
		}
	}
}

func TestCookiePermanentErrorsAreNotRetried(t *testing.T) {
	for _, rpcErr := range []error{
		ccnuv1.ErrorInvalidSidOrPwd("incorrect"),
		ccnuv1.ErrorInvalidXkSystem("unsupported"),
		ccnuv1.ErrorCcnuserverError(tool.CCNUAccountInitializationRequiredMarker),
	} {
		s := newCookieTestService(t, "2024200000")
		s.cache = &missingCookieCache{}
		calls := 0
		s.ccnu = cookieRPCStub{get: func(context.Context, *ccnuv1.GetXKCookieRequest) (*ccnuv1.GetXKCookieResponse, error) {
			calls++
			return nil, status.Convert(rpcErr).Err()
		}}
		_, err := s.GetCookie(context.Background(), "2024200000", tool.XKUndergraduate)
		if calls != 1 || userv1.IsIncorrectPasswordError(err) != ccnuv1.IsInvalidSidOrPwd(rpcErr) ||
			ccnuv1.IsInvalidXkSystem(err) != ccnuv1.IsInvalidXkSystem(rpcErr) ||
			tool.IsCCNUAccountInitializationRequired(err) != tool.IsCCNUAccountInitializationRequired(rpcErr) {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	}
}

func TestCookieTransientErrorsKeepBoundedRetries(t *testing.T) {
	unavailable := status.Error(codes.Unavailable, "temporarily unavailable")
	incorrect := status.Convert(ccnuv1.ErrorInvalidSidOrPwd("incorrect")).Err()
	for _, tc := range []struct {
		name     string
		failures []error
		wantErr  error
	}{
		{name: "recover after temporary failure", failures: []error{unavailable}},
		{name: "stop after permanent failure", failures: []error{unavailable, incorrect}, wantErr: incorrect},
		{name: "at most three attempts", failures: []error{unavailable, unavailable, unavailable}, wantErr: unavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newCookieTestService(t, "2024200000")
			calls := 0
			s.ccnu = cookieRPCStub{get: func(context.Context, *ccnuv1.GetXKCookieRequest) (*ccnuv1.GetXKCookieResponse, error) {
				calls++
				if calls <= len(tc.failures) {
					return nil, tc.failures[calls-1]
				}
				return &ccnuv1.GetXKCookieResponse{Cookie: "fresh-cookie"}, nil
			}}
			cookie, err := s.getNewCookie(context.Background(), "2024200000", tool.XKUndergraduate)
			wantCalls := len(tc.failures)
			if tc.wantErr == nil {
				wantCalls++
				if cookie != "fresh-cookie" {
					t.Fatalf("cookie=%q", cookie)
				}
			}
			if calls != wantCalls || !errors.Is(err, tc.wantErr) {
				t.Fatalf("calls=%d want=%d err=%v", calls, wantCalls, err)
			}
		})
	}
}

func TestCookieCanceledLoginIsNotRetried(t *testing.T) {
	for _, canceledBeforeCall := range []bool{false, true} {
		s := newCookieTestService(t, "2024200000")
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		s.ccnu = cookieRPCStub{get: func(context.Context, *ccnuv1.GetXKCookieRequest) (*ccnuv1.GetXKCookieResponse, error) {
			calls++
			cancel()
			return nil, status.Error(codes.Canceled, "request canceled")
		}}
		wantCalls := 1
		if canceledBeforeCall {
			cancel()
			wantCalls = 0
		}
		_, err := s.getNewCookie(ctx, "2024200000", tool.XKUndergraduate)
		cancel()
		if calls != wantCalls || !errors.Is(err, context.Canceled) {
			t.Fatalf("canceledBeforeCall=%v calls=%d err=%v", canceledBeforeCall, calls, err)
		}
	}
}

func TestCookieConcurrentRefreshAndSynchronousCacheFill(t *testing.T) {
	s := newCookieTestService(t, "2024200000")
	var mu sync.Mutex
	cached := ""
	writeStarted, finishWrite := make(chan struct{}), make(chan struct{})
	var loginCalls atomic.Int32
	s.cache = &cookieCacheStub{
		get: func(context.Context, string, tool.XKSystem) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			if cached == "" {
				return "", cache.ErrKeyNotFound
			}
			return cached, nil
		},
		set: func(ctx context.Context, _ string, cookie string, _ tool.XKSystem) error {
			close(writeStarted)
			select {
			case <-finishWrite:
			case <-ctx.Done():
				return ctx.Err()
			}
			mu.Lock()
			cached = cookie
			mu.Unlock()
			return nil
		},
	}
	s.ccnu = cookieRPCStub{
		get: func(context.Context, *ccnuv1.GetXKCookieRequest) (*ccnuv1.GetXKCookieResponse, error) {
			loginCalls.Add(1)
			return &ccnuv1.GetXKCookieResponse{Cookie: "fresh-cookie"}, nil
		},
		check: func(context.Context, *ccnuv1.CheckXKCookieRequest) (*ccnuv1.CheckXKCookieResponse, error) {
			return &ccnuv1.CheckXKCookieResponse{Valid: true}, nil
		},
	}
	results := make(chan error, 32)
	for i := 0; i < 32; i++ {
		go func() {
			cookie, err := s.GetCookie(context.Background(), "2024200000", tool.XKUndergraduate)
			if err == nil && cookie != "fresh-cookie" {
				err = errors.New("wrong shared cookie")
			}
			results <- err
		}()
	}
	<-writeStarted
	received := 0
	select {
	case <-results:
		received++
		t.Error("request returned before cache fill finished")
	default:
	}
	// 不匹配系统的请求必须立即报错，不能共享正在刷新的本科 Cookie。
	if _, err := s.GetCookie(context.Background(), "2024200000", tool.XKPostgraduate); !ccnuv1.IsInvalidXkSystem(err) {
		t.Errorf("mismatch err=%v", err)
	}
	close(finishWrite)
	for i := received; i < 32; i++ {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
	if loginCalls.Load() != 1 {
		t.Fatalf("loginCalls=%d", loginCalls.Load())
	}
}

func TestCookieCacheFillSurvivesRequestCancellation(t *testing.T) {
	s := newCookieTestService(t, "2024200000")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wrote := false
	s.cache = &cookieCacheStub{
		get: func(context.Context, string, tool.XKSystem) (string, error) { return "", cache.ErrKeyNotFound },
		set: func(cacheCtx context.Context, _, _ string, _ tool.XKSystem) error {
			deadline, ok := cacheCtx.Deadline()
			if ctx.Err() == nil || cacheCtx.Err() != nil || !ok || time.Until(deadline) > time.Second {
				t.Error("cache write did not use an independent bounded context")
			}
			wrote = true
			return nil
		},
	}
	s.ccnu = cookieRPCStub{get: func(context.Context, *ccnuv1.GetXKCookieRequest) (*ccnuv1.GetXKCookieResponse, error) {
		cancel()
		return &ccnuv1.GetXKCookieResponse{Cookie: "fresh-cookie"}, nil
	}}
	cookie, err := s.GetCookie(ctx, "2024200000", tool.XKUndergraduate)
	if err != nil || cookie != "fresh-cookie" || !wrote {
		t.Fatalf("cookie=%q err=%v wrote=%v", cookie, err, wrote)
	}
}
