package authorization

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jitsucom/jitsu/configurator/middleware"

	"github.com/carlmjohnson/requests"
	"github.com/jitsucom/jitsu/configurator/handlers"
	"github.com/jitsucom/jitsu/server/random"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type BoxyHQ struct {
	Config *SSOConfig
}

func (p *BoxyHQ) Name() string {
	return BoxyHQName
}

func (p *BoxyHQ) AccessTokenTTL() time.Duration {
	return time.Duration(p.Config.AccessTokenTTLSeconds) * time.Second
}

func (p *BoxyHQ) GetSSOSession(ctx context.Context, code string) (*handlers.SSOSession, error) {
	conf := &clientcredentials.Config{
		ClientID:     "dummy",
		ClientSecret: "dummy",
		EndpointParams: url.Values{
			"tenant":     {p.Config.Tenant},
			"product":    {p.Config.Product},
			"grant_type": {"authorization_code"},
			"code":       {code},
		},
		TokenURL:  p.Config.Host + "/api/oauth/token",
		AuthStyle: oauth2.AuthStyleInParams,
	}

	token, err := conf.Token(ctx)
	if err != nil {
		return nil, middleware.ReadableError{
			Description: "Failed to get BoxyHQ SSO token",
			Cause:       err,
		}
	}

	var info boxyHQUserInfo
	if err := requests.URL(p.Config.Host+"/api/oauth/userinfo").
		Header("authorization", "Bearer "+token.AccessToken).
		CheckStatus(http.StatusOK).
		ToJSON(&info).
		Fetch(ctx); err != nil {
		return nil, middleware.ReadableError{
			Description: "Failed to load user info from BoxyHQ SSO",
			Cause:       err,
		}
	}

	return &handlers.SSOSession{
		UserID:      info.ID,
		Email:       info.Email,
		AccessToken: token.AccessToken,
	}, nil
}

func (p *BoxyHQ) AuthLink() string {
	return fmt.Sprintf(
		"%s?response_type=code&provider=saml&client_id=dummy&tenant=%s&product=%s&state=%s",
		p.Config.Host+"/api/oauth/authorize",
		p.Config.Tenant,
		p.Config.Product,
		random.String(10),
	)
}

type boxyHQUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}
