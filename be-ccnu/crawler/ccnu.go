package crawler

import (
	"context"
	"crypto/rsa"
	"net/http"
	"time"

	"github.com/asynccnu/ccnubox-be/be-ccnu/service"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/tool"
)

// CCNUCrawler 负责 CCNU 外部系统的完整技术流程。
// 它只保存配置；每次完整流程都会创建独立的 HTTP client 和 CookieJar，避免不同用户共享登录态。
type CCNUCrawler struct {
	proxyClient proxy.Client
	timeout     time.Duration
	secret      string
}

func NewCCNUCrawler(proxyClient proxy.Client, timeout time.Duration, secret string) *CCNUCrawler {
	return &CCNUCrawler{
		proxyClient: proxyClient,
		timeout:     timeout,
		secret:      secret,
	}
}

func (c *CCNUCrawler) newPassport() *Passport {
	return &Passport{
		Client: newCrawlerClient(c.proxyClient, c.timeout),
	}
}

func (c *CCNUCrawler) newUnderGrad(client *http.Client) *UnderGrad {
	return &UnderGrad{
		Client: client,
	}
}

func (c *CCNUCrawler) newPostGraduate() *PostGraduate {
	return &PostGraduate{
		client: newCrawlerClient(c.proxyClient, c.timeout),
	}
}

func (c *CCNUCrawler) newLibrary(client *http.Client) *Library {
	return &Library{
		Client: client,
		Secret: c.secret,
	}
}

// 给 Service 调用的，检验登陆是否成功
func (c *CCNUCrawler) LoginUnderGraduate(ctx context.Context, studentID, password string) (bool, error) {
	_, ok, err := c.loginUnderGraduate(ctx, studentID, password)
	return ok, err
}

// 给 crawler 编排用的，拿取 cas client
func (c *CCNUCrawler) loginUnderGraduate(ctx context.Context, studentID, password string) (*http.Client, bool, error) {
	passport := c.newPassport()
	ok, err := passport.LoginPassport(ctx, studentID, password)
	if err != nil {
		return nil, ok, classifyUnderGraduateLoginError(err)
	}
	return passport.Client, ok, nil
}

func classifyUnderGraduateLoginError(err error) error {
	switch {
	case errorx.Is(err, INCorrectPASSWORD):
		return service.NewCrawlerError(
			service.CrawlerInvalidCredential,
			errorx.Errorf("loginUnderGrad passport error: %w", err),
		)
	case tool.IsCCNUAccountInitializationRequired(err):
		return service.NewCrawlerError(
			service.CrawlerAccountInitializationRequired,
			errorx.Errorf("loginUnderGrad account initialization required: %w", err),
		)
	default:
		return service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("loginUnderGrad internal error: %w", err),
		)
	}
}

func (c *CCNUCrawler) GetUnderGraduateCookie(ctx context.Context, studentID, password string, tpe ...string) (string, error) {
	client, ok, err := c.loginUnderGraduate(ctx, studentID, password)
	if err != nil {
		return "", errorx.Errorf("getUnderGradCookie loginUnderGrad error: %w", err)
	}
	if !ok {
		return "", service.NewCrawlerError(
			service.CrawlerInvalidCredential,
			errorx.New("getUnderGradCookie login failed"),
		)
	}

	underGrad := c.newUnderGrad(client)
	_, err = tool.Retry(func() (string, error) {
		if err := underGrad.LoginUnderGradSystem(ctx); err != nil {
			return "", err
		}
		return "", nil
	})
	if err != nil {
		return "", service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("getUnderGradCookie LoginUnderGradSystem error: %w", err),
		)
	}

	cookie, err := underGrad.GetCookieFromUnderGradSystem()
	if err != nil {
		return "", service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("getUnderGradCookie GetCookieFromUnderGradSystem error: %w", err),
		)
	}

	return cookie, nil
}

func (c *CCNUCrawler) LoginPostGraduate(ctx context.Context, studentID, password string) (bool, error) {
	postGraduate := c.newPostGraduate()

	pubKey, err := tool.Retry(func() (*rsa.PublicKey, error) {
		return postGraduate.FetchPublicKey(ctx)
	})
	if err != nil {
		return false, service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("loginGrad FetchPublicKey error: %w", err),
		)
	}

	incorrectPassword := false
	_, err = tool.Retry(func() (string, error) {
		err := postGraduate.LoginPostgraduateSystem(ctx, studentID, password, pubKey)
		if errorx.Is(err, INCorrectPASSWORD) {
			incorrectPassword = true
			return "", nil
		}
		return "", err
	})
	if incorrectPassword {
		return false, service.NewCrawlerError(
			service.CrawlerInvalidCredential,
			errorx.New("loginGrad incorrect password"),
		)
	}
	if err != nil {
		return false, service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("loginGrad LoginPostgraduateSystem error: %w", err),
		)
	}

	return true, nil
}

func (c *CCNUCrawler) GetPostGraduateCookie(ctx context.Context, studentID, password string) (string, error) {
	postGraduate := c.newPostGraduate()

	pubKey, err := tool.Retry(func() (*rsa.PublicKey, error) {
		return postGraduate.FetchPublicKey(ctx)
	})
	if err != nil {
		return "", service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("getGradCookie FetchPublicKey error: %w", err),
		)
	}

	cookie, err := postGraduate.GetCookie(ctx, studentID, password, pubKey)
	if err != nil {
		return "", service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("getGradCookie GetCookie error: %w", err),
		)
	}
	return cookie, nil
}

func (c *CCNUCrawler) GetLibrarySeatToken(ctx context.Context, studentID, password string) (string, error) {
	library, err := c.newAuthenticatedLibrary(ctx, studentID, password)
	if err != nil {
		return "", err
	}

	token, err := library.GetSeatAuthTokenFromLibrary(ctx)
	if err != nil {
		return "", service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("GetLibrarySeatToken GetRawTokenFromLibrarySystem error: %w", err),
		)
	}
	return token, nil
}

func (c *CCNUCrawler) GetLibraryDiscussionToken(ctx context.Context, studentID, password string) (string, error) {
	library, err := c.newAuthenticatedLibrary(ctx, studentID, password)
	if err != nil {
		return "", err
	}

	token, err := library.GetDiscussionAuthTokenFromLibrary(ctx)
	if err != nil {
		return "", service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("GetLibraryDiscussionToken GetRawTokenFromLibrarySystem error: %w", err),
		)
	}
	return token, nil
}

func (c *CCNUCrawler) newAuthenticatedLibrary(ctx context.Context, studentID, password string) (*Library, error) {
	client, ok, err := c.loginUnderGraduate(ctx, studentID, password)
	if err != nil {
		return nil, errorx.Errorf("GetLibraryToken loginUnderGrad error: %w", err)
	}
	if !ok {
		return nil, service.NewCrawlerError(
			service.CrawlerInvalidCredential,
			errorx.New("GetLibraryToken login failed"),
		)
	}

	library := c.newLibrary(client)
	if err := library.LoginLibrary(ctx); err != nil {
		return nil, service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("GetLibraryToken LoginLibrary error: %w", err),
		)
	}
	return library, nil
}

func (c *CCNUCrawler) CheckLibrarySeatToken(ctx context.Context, token string) (bool, error) {
	library := c.newLibrary(
		newCrawlerClient(c.proxyClient, c.timeout, proxy.WithoutProxy()),
	)

	ok, err := library.CheckLibrarySeatToken(ctx, token)
	if err != nil {
		return false, service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("CheckLibrarySeatToken library crawler check token err: %w", err),
		)
	}
	return ok, nil
}

func (c *CCNUCrawler) CheckLibraryDiscussionToken(ctx context.Context, token string) (bool, error) {
	library := c.newLibrary(
		newCrawlerClient(c.proxyClient, c.timeout, proxy.WithoutProxy()),
	)

	ok, err := library.CheckLibraryDiscussionToken(ctx, token)
	if err != nil {
		return false, service.NewCrawlerError(
			service.CrawlerUpstreamFailure,
			errorx.Errorf("CheckLibraryDiscussionToken library crawler check token err: %w", err),
		)
	}
	return ok, nil
}

var _ service.Crawler = (*CCNUCrawler)(nil)
