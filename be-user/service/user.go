package service

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/asynccnu/ccnubox-be/be-user/pkg/crypto"
	"github.com/asynccnu/ccnubox-be/be-user/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-user/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-user/repository/model"
	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"golang.org/x/sync/singleflight"
)

// 定义基于业务错误码的错误构造函数
var (
	SAVE_USER_ERROR      = errorx.FormatErrorFunc(userv1.ErrorSaveUserError("保存用户失败"))
	DEFAULT_DAO_ERROR    = errorx.FormatErrorFunc(userv1.ErrorDefaultDaoError("数据库异常"))
	USER_NOT_FOUND_ERROR = errorx.FormatErrorFunc(userv1.ErrorUserNotFoundError("无法找到该用户"))
	CCNU_GETCOOKIE_ERROR = errorx.FormatErrorFunc(userv1.ErrorCcnuGetcookieError("获取Cookie失败"))
	ENCRYPT_ERROR        = errorx.FormatErrorFunc(userv1.ErrorEncryptError("Password加密失败"))
	DECRYPT_ERROR        = errorx.FormatErrorFunc(userv1.ErrorDecryptError("Password解密失败"))
	InCorrectPassword    = errorx.FormatErrorFunc(userv1.ErrorIncorrectPasswordError("账号密码错误"))
)

type UserService interface {
	Save(ctx context.Context, studentId string, password string) error
	GetCookie(ctx context.Context, studentId string, tpe ...string) (string, error)
	GetLibraryCookie(ctx context.Context, studentId string) (string, error)
	Check(ctx context.Context, studentId string, password string) (bool, error)
}

type userService struct {
	dao          dao.UserDAO
	cryptoClient *crypto.Crypto
	cache        cache.UserCache
	ccnu         ccnuv1.CCNUServiceClient
	sfGroup      singleflight.Group
	l            logger.Logger
	pClient      proxyv1.ProxyClient
}

func NewUserService(dao dao.UserDAO, cache cache.UserCache, cryptoClient *crypto.Crypto, ccnu ccnuv1.CCNUServiceClient, l logger.Logger,
	pClient proxyv1.ProxyClient) UserService {
	return &userService{dao: dao, cache: cache, cryptoClient: cryptoClient, ccnu: ccnu, l: l, pClient: pClient}
}

func (s *userService) Save(ctx context.Context, studentId string, password string) error {
	// 密码加密
	encryptedPwd, err := s.cryptoClient.Encrypt(password)
	if err != nil {
		return ENCRYPT_ERROR(errorx.Errorf("service: encrypt failed, err: %w", err))
	}

	user, err := s.dao.FindByStudentId(ctx, studentId)
	switch {
	case err == nil:
		if user.Password != encryptedPwd {
			user.Password = encryptedPwd
		}
		return nil
	case errors.Is(err, dao.UserNotFound):
		user = &model.User{
			StudentId: studentId,
			Password:  encryptedPwd,
		}
	default:
		return DEFAULT_DAO_ERROR(errorx.Errorf("service: find user failed, sid: %s, err: %w", studentId, err))
	}

	if err = s.dao.Save(ctx, user); err != nil {
		return SAVE_USER_ERROR(errorx.Errorf("service: dao save failed, sid: %s, err: %w", studentId, err))
	}
	return nil
}

func (s *userService) Check(ctx context.Context, studentId string, password string) (bool, error) {
	tlog := s.l.WithContext(ctx)

	// 优先尝试从官方教务验证
	_, err := tool.Retry(func() (*ccnuv1.LoginCCNUResponse, error) {
		return s.ccnu.LoginCCNU(ctx, &ccnuv1.LoginCCNURequest{StudentId: studentId, Password: password})
	})

	switch {
	case err == nil:
		return true, nil
	case ccnuv1.IsInvalidSidOrPwd(err):
		return false, InCorrectPassword(errorx.New("sid or password incorrect in ccnu system"))
	}

	tlog.Warn("ccnu login failed, fallback to local check", logger.Error(err))

	// 降级逻辑：检查本地数据库加密密码
	encryptedPwd, err := s.cryptoClient.Encrypt(password)
	if err != nil {
		return false, ENCRYPT_ERROR(errorx.Errorf("service: fallback encrypt failed, err: %w", err))
	}

	user, err := s.dao.FindByStudentId(ctx, studentId)
	if err != nil {
		if errors.Is(err, dao.UserNotFound) {
			return false, USER_NOT_FOUND_ERROR(errorx.Errorf("user %s not found locally", studentId))
		}
		return false, DEFAULT_DAO_ERROR(errorx.Errorf("local dao query failed, err: %w", err))
	}

	if user.Password == encryptedPwd {
		return true, nil
	}
	return false, InCorrectPassword(errorx.New("password does not match local record"))
}

func (s *userService) GetCookie(ctx context.Context, studentId string, tpe ...string) (string, error) {
	tlog := s.l.WithContext(ctx)
	// 使用 Singleflight 防止热点学号瞬间击穿缓存请求教务
	result, err, _ := s.sfGroup.Do(studentId, func() (interface{}, error) {
		cookie, err := s.cache.GetCookie(ctx, studentId)
		// 缓存命中且校验有效，直接返回
		if err == nil && s.checkCookie(ctx, cookie) {
			return cookie, nil
		}

		if err != nil && !errors.Is(err, cache.ErrKeyNotFound) {
			tlog.Warn("cache error", logger.Error(err))
		}

		// 缓存失效或 Cookie 过期，获取新 Cookie
		newCookie, err := s.getNewCookie(ctx, studentId, tpe...)
		if err != nil {
			return "", err
		}

		// 异步回填缓存，不阻塞主流程
		go func(sid, cky string) {
			// 使用 Background 防止主请求 ctx 取消导致写入失败
			if err := s.cache.SetCookie(context.Background(), sid, cky); err != nil {
				s.l.Error("async fill cache failed", logger.String("sid", sid), logger.Error(err))
			}
		}(studentId, newCookie)

		return newCookie, nil
	})

	if err != nil {
		return "", CCNU_GETCOOKIE_ERROR(err)
	}
	return result.(string), nil
}

func (s *userService) getNewCookie(ctx context.Context, studentId string, tpe ...string) (string, error) {
	user, err := s.dao.FindByStudentId(ctx, studentId)
	if err != nil {
		return "", USER_NOT_FOUND_ERROR(errorx.Errorf("sid: %s, err: %w", studentId, err))
	}

	decryptPassword, err := s.cryptoClient.Decrypt(user.Password)
	if err != nil {
		return "", DECRYPT_ERROR(errorx.Errorf("sid: %s, err: %w", studentId, err))
	}

	resp, err := tool.Retry(func() (*ccnuv1.GetXKCookieResponse, error) {
		req := &ccnuv1.GetXKCookieRequest{StudentId: user.StudentId, Password: decryptPassword}
		if len(tpe) > 0 {
			req.Type = tpe[0]
		}
		return s.ccnu.GetXKCookie(ctx, req)
	})

	if err != nil {
		return "", errorx.Errorf("rpc GetXKCookie failed, sid: %s, err: %w", studentId, err)
	}
	return resp.Cookie, nil
}

func (s *userService) GetLibraryCookie(ctx context.Context, studentId string) (string, error) {
	key := "lib:" + studentId
	result, err, _ := s.sfGroup.Do(key, func() (interface{}, error) {
		cookie, err := s.cache.GetLibraryCookie(ctx, studentId)
		if err == nil && s.checkLibraryCookie(ctx, cookie) {
			return cookie, nil
		}

		newCookie, err := s.getNewLibraryCookie(ctx, studentId)
		if err != nil {
			return "", err
		}

		go func(sid, cky string) {
			if err := s.cache.SetLibraryCookie(context.Background(), sid, cky); err != nil {
				s.l.Error("async fill library cache failed", logger.Error(err))
			}
		}(studentId, newCookie)

		return newCookie, nil
	})

	if err != nil {
		return "", CCNU_GETCOOKIE_ERROR(err)
	}
	return result.(string), nil
}

func (s *userService) getNewLibraryCookie(ctx context.Context, studentId string) (string, error) {
	user, err := s.dao.FindByStudentId(ctx, studentId)
	if err != nil {
		return "", USER_NOT_FOUND_ERROR(err)
	}

	decryptPassword, err := s.cryptoClient.Decrypt(user.Password)
	if err != nil {
		return "", DECRYPT_ERROR(err)
	}

	resp, err := tool.Retry(func() (*ccnuv1.GetLibraryCookieResponse, error) {
		return s.ccnu.GetLibraryCookie(ctx, &ccnuv1.GetLibraryCookieRequest{
			StudentId: user.StudentId,
			Password:  decryptPassword,
		})
	})

	if err != nil {
		return "", errorx.Errorf("rpc GetLibraryCookie failed, err: %w", err)
	}
	return resp.Cookie, nil
}

// TODO 目前只有新版本教务系统的本科生院部分做了如下的检验，同时应该放到ccnu服务里面而不是在这里
// TODO 未来整个cookie系统应该逐步重构保证整个的完整性和高可用性，例如对cookie获取方式进行抽象，提供GetCookie方法并通过参数+策略工厂的方式实现获取cookie高度解耦，对于具体决定是爬取哪个系统应当由上游决定，下游无感化，只作为函数调用。
// 辅助检测函数：试探性请求教务系统，验证 Cookie 存活
func (s *userService) checkCookie(ctx context.Context, cookie string) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://bkzhjw.ccnu.edu.cn/jsxsd/framework/xsMainV.htmlx", nil)
	req.Header.Set("Cookie", cookie)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36 Edg/144.0.0.0")

	proxyAddr, _ := s.getProxyAddr(ctx)
	client := s.newClient(ctx, proxyAddr)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (s *userService) checkLibraryCookie(ctx context.Context, cookie string) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://kjyy.ccnu.edu.cn/", nil)
	req.Header.Set("Cookie", cookie)

	proxyAddr, _ := s.getProxyAddr(ctx)
	client := s.newClient(ctx, proxyAddr)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (s *userService) newClient(ctx context.Context, proxyAddr string) *http.Client {
	cli := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	if proxyAddr != "" {
		if proxy, err := url.Parse(proxyAddr); err == nil {
			cli.Transport = &http.Transport{Proxy: http.ProxyURL(proxy)}
		}
	}
	return cli
}

func (s *userService) getProxyAddr(ctx context.Context) (string, error) {
	res, err := s.pClient.GetProxyAddr(ctx, &proxyv1.GetProxyAddrRequest{})
	if err != nil {
		return "", errorx.Errorf("proxy rpc error: %w", err)
	}
	return res.Addr, nil
}
