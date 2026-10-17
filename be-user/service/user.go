package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-user/pkg/crypto"
	"github.com/asynccnu/ccnubox-be/be-user/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-user/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-user/repository/model"
	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// 定义基于业务错误码的错误构造函数
var (
	SAVE_USER_ERROR      = errorx.FormatErrorFunc(userv1.ErrorSaveUserError("保存用户失败"))
	DEFAULT_DAO_ERROR    = errorx.FormatErrorFunc(userv1.ErrorDefaultDaoError("数据库异常"))
	USER_NOT_FOUND_ERROR = errorx.FormatErrorFunc(userv1.ErrorUserNotFoundError("无法找到该用户"))
	CCNU_GETCOOKIE_ERROR = tool.FormatCCNUErrorFunc(userv1.ErrorCcnuGetcookieError("获取Cookie失败"), userv1.ErrorCcnuGetcookieError(tool.CCNUAccountInitializationRequiredMarker))
	ENCRYPT_ERROR        = errorx.FormatErrorFunc(userv1.ErrorEncryptError("Password加密失败"))
	DECRYPT_ERROR        = errorx.FormatErrorFunc(userv1.ErrorDecryptError("Password解密失败"))
	InCorrectPassword    = errorx.FormatErrorFunc(userv1.ErrorIncorrectPasswordError("账号密码错误"))
)

type UserService interface {
	Save(ctx context.Context, studentId string, password string) error
	Delete(ctx context.Context, studentId string, password string) error
	GetCookie(ctx context.Context, studentId string, system tool.XKSystem) (string, error)
	GetLibrarySeatToken(ctx context.Context, studentId string) (string, error)
	GetLibraryDiscussionToken(ctx context.Context, studentId string) (string, error)
	Check(ctx context.Context, studentId string, password string) (bool, error)
}

type userService struct {
	dao          dao.UserDAO
	cryptoClient *crypto.Crypto
	cache        cache.UserCache
	ccnu         ccnuv1.CCNUServiceClient
	sfGroup      singleflight.Group
	l            logger.Logger
}

func NewUserService(dao dao.UserDAO, cache cache.UserCache, cryptoClient *crypto.Crypto, ccnu ccnuv1.CCNUServiceClient, l logger.Logger) UserService {
	return &userService{dao: dao, cache: cache, cryptoClient: cryptoClient, ccnu: ccnu, l: l}
}

func (s *userService) Delete(ctx context.Context, studentId string, password string) error {
	user, err := s.dao.FindByStudentId(ctx, studentId)
	if err != nil {
		if errors.Is(err, dao.UserNotFound) {
			return USER_NOT_FOUND_ERROR(errorx.Errorf("service: user not found for deletion, sid: %s, err: %w", studentId, err))
		}
		return DEFAULT_DAO_ERROR(errorx.Errorf("service: find user for deletion failed, sid: %s, err: %w", studentId, err))
	}
	storedPassword, err := s.cryptoClient.Decrypt(user.Password)
	if err != nil {
		return DECRYPT_ERROR(errorx.Errorf("service: decrypt password for deletion failed, sid: %s, err: %w", studentId, err))
	}
	if subtle.ConstantTimeCompare([]byte(storedPassword), []byte(password)) != 1 {
		return InCorrectPassword(errorx.New("service: delete user password mismatch"))
	}
	if err := s.cache.DeleteUserData(ctx, studentId); err != nil {
		return DEFAULT_DAO_ERROR(errorx.Errorf("service: clear user credentials before deletion failed, sid: %s, err: %w", studentId, err))
	}
	if err := s.dao.Delete(ctx, studentId); err != nil {
		return DEFAULT_DAO_ERROR(errorx.Errorf("service: delete user failed, sid: %s, err: %w", studentId, err))
	}
	return nil
}

func (s *userService) Save(ctx context.Context, studentId string, password string) error {
	user, err := s.dao.FindByStudentId(ctx, studentId)
	switch {
	case err == nil:
		storedPassword, decryptErr := s.cryptoClient.Decrypt(user.Password)
		if decryptErr != nil {
			return DECRYPT_ERROR(errorx.Errorf("service: decrypt stored password failed, sid: %s, err: %w", studentId, decryptErr))
		}
		if storedPassword == password && !s.cryptoClient.NeedsMigration(user.Password) {
			return nil
		}
	case errors.Is(err, dao.UserNotFound):
		user = &model.User{
			StudentId: studentId,
		}
	default:
		return DEFAULT_DAO_ERROR(errorx.Errorf("service: find user failed, sid: %s, err: %w", studentId, err))
	}

	encryptedPwd, err := s.cryptoClient.Encrypt(password)
	if err != nil {
		return ENCRYPT_ERROR(errorx.Errorf("service: encrypt failed, err: %w", err))
	}
	user.Password = encryptedPwd

	if err = s.dao.Save(ctx, user); err != nil {
		return SAVE_USER_ERROR(errorx.Errorf("service: dao save failed, sid: %s, err: %w", studentId, err))
	}
	return nil
}

func (s *userService) Check(ctx context.Context, studentId string, password string) (bool, error) {
	tlog := s.l.WithContext(ctx)

	// 优先尝试从官方教务验证
	var accountInitializationErr error
	_, err := tool.Retry(func() (*ccnuv1.LoginCCNUResponse, error) {
		resp, loginErr := s.ccnu.LoginCCNU(ctx, &ccnuv1.LoginCCNURequest{StudentId: studentId, Password: password})
		if tool.IsCCNUAccountInitializationRequired(loginErr) {
			accountInitializationErr = loginErr
			return nil, nil
		}
		return resp, loginErr
	})
	if accountInitializationErr != nil {
		return false, errorx.Errorf("service: ccnu account initialization required: %w", accountInitializationErr)
	}

	switch {
	case err == nil:
		return true, nil
	case ccnuv1.IsInvalidSidOrPwd(err):
		return false, InCorrectPassword(errorx.New("sid or password incorrect in ccnu system"))
	}

	tlog.Warn("ccnu login failed, fallback to local check", logger.Error(err))

	// 降级逻辑：检查本地数据库加密密码
	user, err := s.dao.FindByStudentId(ctx, studentId)
	if err != nil {
		if errors.Is(err, dao.UserNotFound) {
			return false, USER_NOT_FOUND_ERROR(errorx.Errorf("user %s not found locally", studentId))
		}
		return false, DEFAULT_DAO_ERROR(errorx.Errorf("local dao query failed, err: %w", err))
	}

	decryptedPwd, err := s.cryptoClient.Decrypt(user.Password)
	if err != nil {
		return false, DEFAULT_DAO_ERROR(errorx.Errorf("service: decrypt failed, err: %w", err))
	}

	if decryptedPwd == password {
		return true, nil
	}
	tlog.Error("database check failed, login failed")
	return false, InCorrectPassword(errorx.New("password does not match local record"))
}

func (s *userService) GetCookie(ctx context.Context, studentId string, system tool.XKSystem) (string, error) {
	if err := tool.ValidateXKSystem(studentId, system); err != nil {
		return "", err
	}
	tlog := s.l.WithContext(ctx)
	// 系统类型在兼容入口归一化，缓存与请求合并使用同一份明确类型。
	key := fmt.Sprintf("xk:%s:%s", system, studentId)
	result, err, _ := s.sfGroup.Do(key, func() (interface{}, error) {
		cookie, err := s.cache.GetCookie(ctx, studentId, system)
		if err == nil {
			resp, checkErr := s.ccnu.CheckXKCookie(ctx, &ccnuv1.CheckXKCookieRequest{Cookie: cookie, Type: string(system)})
			if checkErr != nil {
				// 暂时不可验证不代表过期，保留缓存并由调用方重试，避免无意义登录。
				return "", checkErr
			}
			if resp == nil {
				return "", errorx.New("rpc CheckXKCookie returned empty response")
			}
			if resp.GetValid() {
				return cookie, nil
			}
		} else if !errors.Is(err, cache.ErrKeyNotFound) {
			tlog.Warn("cookie cache read failed", logger.String("system", string(system)), logger.Error(err))
		}

		newCookie, err := s.getNewCookie(ctx, studentId, system)
		if err != nil {
			return "", err
		}
		// 在请求合并结束前回填，避免异步窗口导致重复登录；脱离请求取消但写入时间有界。
		cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		if err := s.cache.SetCookie(cacheCtx, studentId, newCookie, system); err != nil {
			tlog.Warn("cookie cache write failed", logger.String("system", string(system)), logger.Error(err))
		}
		return newCookie, nil
	})

	if err != nil {
		if ccnuv1.IsInvalidSidOrPwd(err) {
			return "", InCorrectPassword(err)
		}
		if ccnuv1.IsInvalidXkSystem(err) || userv1.IsUserNotFoundError(err) ||
			userv1.IsDefaultDaoError(err) || userv1.IsDecryptError(err) {
			return "", err
		}
		return "", CCNU_GETCOOKIE_ERROR(err)
	}
	return result.(string), nil
}

func (s *userService) getNewCookie(ctx context.Context, studentId string, system tool.XKSystem) (string, error) {
	user, err := s.dao.FindByStudentId(ctx, studentId)
	if err != nil {
		if errors.Is(err, dao.UserNotFound) {
			return "", USER_NOT_FOUND_ERROR(errorx.Errorf("sid: %s, err: %w", studentId, err))
		}
		return "", DEFAULT_DAO_ERROR(errorx.Errorf("sid: %s, err: %w", studentId, err))
	}

	decryptPassword, err := s.cryptoClient.Decrypt(user.Password)
	if err != nil {
		return "", DECRYPT_ERROR(errorx.Errorf("sid: %s, err: %w", studentId, err))
	}

	// 保留暂时故障的有限重试，永久性账号错误和请求取消则立即停止。
	var terminalErr error
	resp, err := tool.Retry(func() (*ccnuv1.GetXKCookieResponse, error) {
		if ctx.Err() != nil {
			terminalErr = ctx.Err()
			return nil, nil
		}
		resp, err := s.ccnu.GetXKCookie(ctx, &ccnuv1.GetXKCookieRequest{
			StudentId: user.StudentId, Password: decryptPassword, Type: string(system),
		})
		if err != nil && (ccnuv1.IsInvalidSidOrPwd(err) || ccnuv1.IsInvalidXkSystem(err) ||
			tool.IsCCNUAccountInitializationRequired(err) || ctx.Err() != nil) {
			terminalErr = err
			if ctx.Err() != nil {
				terminalErr = ctx.Err()
			}
			return nil, nil
		}
		return resp, err
	})
	if terminalErr != nil {
		err = terminalErr
	}
	if err != nil {
		return "", errorx.Errorf("rpc GetXKCookie failed, system: %s, err: %w", system, err)
	}
	if resp == nil || resp.GetCookie() == "" {
		return "", errorx.New("rpc GetXKCookie returned empty cookie")
	}
	return resp.GetCookie(), nil
}

func (s *userService) GetLibrarySeatToken(ctx context.Context, studentId string) (string, error) {
	return s.GetLibraryToken(ctx, studentId, ccnuv1.LIBRARY_TYPE_LIBRARY_SEAT)
}

func (s *userService) GetLibraryDiscussionToken(ctx context.Context, studentId string) (string, error) {
	return s.GetLibraryToken(ctx, studentId, ccnuv1.LIBRARY_TYPE_LIBRARY_DISCUSSION)
}

func (s *userService) GetLibraryToken(ctx context.Context, studentId string, serviceType ccnuv1.LIBRARY_TYPE) (string, error) {
	key := fmt.Sprintf("lib:%s:%s", serviceType, studentId)
	result, err, _ := s.sfGroup.Do(key, func() (interface{}, error) {
		token, err := s.cache.GetLibraryToken(ctx, studentId, serviceType.String())

		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
		if token != "" {
			ok, err := s.checkLibraryToken(ctx, token, serviceType)
			if err != nil {
				return nil, err
			}

			if ok {
				return token, nil
			}
		}

		newToken, err := s.getNewLibraryToken(ctx, studentId, serviceType)
		if err != nil {
			return "", err
		}

		go func(sid, cky string) {
			if err := s.cache.SetLibraryToken(context.Background(), sid, cky, serviceType.String()); err != nil {
				s.l.Error("async fill library cache failed", logger.Error(err))
			}
		}(studentId, newToken)

		return newToken, nil

	})

	if err != nil {
		return "", CCNU_GETCOOKIE_ERROR(err)
	}
	return result.(string), nil
}

func (s *userService) getNewLibraryToken(ctx context.Context, studentId string, serviceType ccnuv1.LIBRARY_TYPE) (string, error) {
	user, err := s.dao.FindByStudentId(ctx, studentId)
	if err != nil {
		return "", USER_NOT_FOUND_ERROR(err)
	}

	decryptPassword, err := s.cryptoClient.Decrypt(user.Password)
	if err != nil {
		return "", DECRYPT_ERROR(err)
	}

	resp, err := tool.Retry(func() (*ccnuv1.GetLibraryTokenResponse, error) {
		return s.ccnu.GetLibraryToken(ctx, &ccnuv1.GetLibraryTokenRequest{
			StudentId: user.StudentId,
			Password:  decryptPassword,
			Type:      serviceType,
		})
	})
	if err != nil {
		return "", errorx.Errorf("rpc GetLibraryCookie failed, err: %w", err)
	}
	return resp.GetToken(), nil
}

func (s *userService) checkLibraryToken(ctx context.Context, token string, serviceType ccnuv1.LIBRARY_TYPE) (bool, error) {
	req := &ccnuv1.CheckLibraryTokenRequest{
		Token: token,
		Type:  serviceType,
	}

	res, err := s.ccnu.CheckLibraryToken(ctx, req)
	if err != nil {
		return false, err
	}
	return res.GetValid(), nil
}
