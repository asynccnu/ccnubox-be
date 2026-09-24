package service

import (
	"context"
	"errors"
	"testing"

	usercrypto "github.com/asynccnu/ccnubox-be/be-user/pkg/crypto"
	"github.com/asynccnu/ccnubox-be/be-user/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-user/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-user/repository/model"
	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

func (*deleteUserCache) GetCookie(context.Context, string) (string, error) {
	return "", nil
}

func (*deleteUserCache) SetCookie(context.Context, string, string) error {
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

func (*missingCookieCache) GetCookie(context.Context, string) (string, error) {
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
		_, err := s.GetCookie(context.Background(), "2024000000")
		if err == nil || userv1.IsIncorrectPasswordError(status.Convert(err).Err()) != tc.passwordError {
			t.Fatalf("err=%v want password error=%v", err, tc.passwordError)
		}
	}
}
