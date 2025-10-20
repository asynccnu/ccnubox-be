package service

import (
	"context"
	"errors"
	ccnuv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/ccnu/v1"
	userv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/be-user/pkg/crypto"
	"github.com/asynccnu/ccnubox-be/be-user/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-user/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-user/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-user/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-user/tool"
	"golang.org/x/sync/singleflight"
	"net/http"
)

// 定义错误,这里将kratos的error作为一个重要部分传入
var (
	SAVE_USER_ERROR = func(err error) error {
		return errorx.New(userv1.ErrorSaveUserError("保存用户失败"), "dao", err)
	}

	DEFAULT_DAO_ERROR = func(err error) error {
		return errorx.New(userv1.ErrorDefaultDaoError("数据库异常"), "dao", err)
	}

	USER_NOT_FOUND_ERROR = func(err error) error {
		return errorx.New(userv1.ErrorUserNotFoundError("无法找到该用户"), "dao", err)
	}

	CCNU_GETCOOKIE_ERROR = func(err error) error {
		return errorx.New(userv1.ErrorCcnuGetcookieError("获取Cookie失败"), "ccnu", err)
	}

	ENCRYPT_ERROR = func(err error) error {
		return errorx.New(userv1.ErrorEncryptError("Password加密失败"), "crypt", err)
	}

	DECRYPT_ERROR = func(err error) error {
		return errorx.New(userv1.ErrorDecryptError("Password解密失败"), "crypt", err)
	}
	InCorrectPassword = func(err error) error {
		return errorx.New(userv1.ErrorIncorrectPasswordError("账号密码错误"), "user", err)
	}
)

type UserService interface {
	Save(ctx context.Context, studentId string, password string) error
	GetCookie(ctx context.Context, studentId string) (string, error)
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

func (s *userService) Save(ctx context.Context, studentId string, password string) error {

	//加密
	password, err := s.cryptoClient.Encrypt(password)
	if err != nil {
		return ENCRYPT_ERROR(err)
	}

	//尝试查找用户
	user, err := s.dao.FindByStudentId(ctx, studentId)
	switch {
	case err == nil:

		//检查是否有更新的价值
		if user.Password != password {
			user.Password = password
		} else {
			return nil
		}

	case errors.Is(err, dao.UserNotFound):
		user.StudentId = studentId
		user.Password = password

	default:
		return DEFAULT_DAO_ERROR(err)
	}

	//更新用户
	err = s.dao.Save(ctx, user)
	if err != nil {
		return SAVE_USER_ERROR(err)
	}

	return nil
}

func (s *userService) Check(ctx context.Context, studentId string, password string) (bool, error) {

	_, err := tool.Retry(func() (*ccnuv1.LoginCCNUResponse, error) {
		return s.ccnu.LoginCCNU(ctx, &ccnuv1.LoginCCNURequest{StudentId: studentId, Password: password})
	})

	switch {
	case err == nil:
		return true, nil
	case ccnuv1.IsInvalidSidOrPwd(err):
		return false, InCorrectPassword(errors.New("invalid sid or password"))
	}
	s.l.Warn("尝试从ccnu登录失败!", logger.Error(err))

	//尝试查找用户
	user, err := s.dao.FindByStudentId(ctx, studentId)
	switch err {
	case nil:
		return user.Password == password, nil
	default:
		return false, DEFAULT_DAO_ERROR(err)
	}

}
func (s *userService) GetCookie(ctx context.Context, studentId string) (string, error) {
	key := studentId
	result, err, _ := s.sfGroup.Do(key, func() (interface{}, error) {
		var cookie string
		var newCookie string
		//如果从缓存获取成功就直接返回,否则降级处理
		cookie, err := s.cache.GetCookie(ctx, studentId)
		if err != nil {
			s.l.Info("从缓存获取cookie失败", logger.Error(err))

			//直接获取新的
			newCookie, err = s.getNewCookie(ctx, studentId)
			if err != nil {
				return "", err
			}

		} else {
			//如果是从缓存获取的要验证是否可用
			if !s.checkCookie(cookie) {
				//直接获取新的
				newCookie, err = s.getNewCookie(ctx, studentId)
				if err != nil {
					return "", err
				}
			}

		}

		if newCookie != "" {
			cookie = newCookie
			//异步回填
			go func() {
				err := s.cache.SetCookie(context.Background(), studentId, cookie)
				if err != nil {
					s.l.Error("回填cookie失败", logger.Error(err))
				}
			}()
		}
		return cookie, nil
	})

	if err != nil {
		return "", CCNU_GETCOOKIE_ERROR(err)
	}

	cookie, ok := result.(string)
	if !ok {
		return "", nil
	}

	return cookie, nil
}

func (s *userService) getNewCookie(ctx context.Context, studentId string) (string, error) {
	//失败则重试
	//尝试从数据库获取
	user, err := s.dao.FindByStudentId(ctx, studentId)
	if err != nil {
		return "", USER_NOT_FOUND_ERROR(err)
	}

	//解密
	decryptPassword, err := s.cryptoClient.Decrypt(user.Password)
	if err != nil {
		return "", DECRYPT_ERROR(err)
	}

	resp, err := tool.Retry(func() (*ccnuv1.GetXKCookieResponse, error) {
		return s.ccnu.GetXKCookie(ctx, &ccnuv1.GetXKCookieRequest{
			StudentId: user.StudentId,
			Password:  decryptPassword,
		})
	})

	if err != nil {
		return "", CCNU_GETCOOKIE_ERROR(err)
	}
	return resp.Cookie, nil
}

func (s *userService) checkCookie(cookie string) bool {

	// 试探性请求，保证cookie长时间有效
	req, err := http.NewRequest("GET", "https://xk.ccnu.edu.cn/jwglxt/dlflgl/flzyqr_cxFlzyqrxx.html", nil)
	if err != nil {
		return false
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0")

	// 创建HTTP客户端，禁止自动重定向
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 禁止自动跳转，返回原始响应
		},
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}
