/*
 * Copyright (c) 2020-2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package authentication

import (
	"errors"
	"net/http"
	"strings"

	"github.com/devtron-labs/authenticator/client"
	authMiddleware "github.com/devtron-labs/authenticator/middleware"
	"github.com/devtron-labs/authenticator/oidc"
	"github.com/devtron-labs/devtron/pkg/auth/user"
	"github.com/devtron-labs/devtron/pkg/lec"
	"go.uber.org/zap"
)

type SSOClient interface {
	HandleLogin(http.ResponseWriter, *http.Request)
	HandleCallback(http.ResponseWriter, *http.Request)
}

type UserAuthOidcHelper interface {
	GetClientApp() SSOClient
	GetDexProxy() func(writer http.ResponseWriter, request *http.Request)
	UpdateInMemoryDataOnSsoAddUpdate(ssoUrl string) error
}

const (
	Orchestrator = "/orchestrator"
	Dashboard    = "dashboard"
)

type UserAuthOidcHelperImpl struct {
	logger                       *zap.SugaredLogger
	selfRegistrationRolesService user.UserSelfRegistrationService
	dexProxy                     func(writer http.ResponseWriter, request *http.Request)
	clientApp                    SSOClient
	dexConfig                    *client.DexConfig
	settings                     *oidc.Settings
	sessionManager               *authMiddleware.SessionManager
}

func NewUserAuthOidcHelperImpl(logger *zap.SugaredLogger, selfRegistrationRolesService user.UserSelfRegistrationService, dexConfig *client.DexConfig,
	settings *oidc.Settings, sessionManager *authMiddleware.SessionManager) (*UserAuthOidcHelperImpl, error) {
	impl := &UserAuthOidcHelperImpl{
		logger:                       logger,
		settings:                     settings,
		dexConfig:                    dexConfig,
		sessionManager:               sessionManager,
		selfRegistrationRolesService: selfRegistrationRolesService,
	}
	identity, err := lec.DefaultIdentity()
	if err != nil {
		return nil, err
	}
	if identity != nil {
		// 本地管理员与 API Token 仍使用原签名密钥；SSO Token 改为验证 LecSSO issuer。
		next := *settings
		next.URL = identity.Config.Origin + Orchestrator
		next.OIDCConfig.Issuer = identity.Config.Issuer
		next.OIDCConfig.ClientID = identity.Config.ClientID
		next.OIDCConfig.ClientSecret = identity.Config.ClientSecret
		next.OIDCConfig.RequestedScopes = []string{"openid", "profile"}
		impl.settings = &next
		sessionManager.UpdateSettings(&next, &client.DexConfig{DexServerAddress: identity.Config.Issuer})
		impl.clientApp = lec.NewLoginClient(identity, selfRegistrationRolesService.CheckAndCreateUserIfConfigured)
		impl.dexProxy = http.NotFound
		return impl, nil
	}
	oidcClient, dexProxy, err := client.GetOidcClient(dexConfig, selfRegistrationRolesService.CheckAndCreateUserIfConfigured, impl.sanitiseRedirectUrl)
	if err != nil {
		logger.Errorw("error in getting oidc client", "err", err)
		return nil, err
	}
	impl.dexProxy = dexProxy
	impl.clientApp = oidcClient
	return impl, nil
}

// SanitiseRedirectUrl replaces initial "/orchestrator" from url
func (impl UserAuthOidcHelperImpl) sanitiseRedirectUrl(redirectUrl string) string {
	if strings.Contains(redirectUrl, Dashboard) {
		redirectUrl = strings.ReplaceAll(redirectUrl, Orchestrator, "")
	}
	return redirectUrl
}

func (impl UserAuthOidcHelperImpl) GetClientApp() SSOClient {
	return impl.clientApp
}

func (impl UserAuthOidcHelperImpl) GetDexProxy() func(writer http.ResponseWriter, request *http.Request) {
	return impl.dexProxy
}

func (impl UserAuthOidcHelperImpl) UpdateInMemoryDataOnSsoAddUpdate(ssoUrl string) error {
	if identity, err := lec.DefaultIdentity(); err != nil || identity != nil {
		return errors.New("LecSSO 配置由平台管理员统一维护")
	}

	impl.logger.Infow("updating in memory data on sso update", "ssoUrl", ssoUrl)

	// set url in dexConfig
	impl.dexConfig.Url = ssoUrl
	proxyUrl, err := impl.dexConfig.GetDexProxyUrl()
	if err != nil {
		impl.logger.Errorw("error in getting proxy url from ssoUrl", "err", err, "ssoUrl", ssoUrl)
		return err
	}

	// update url in oidc settings
	impl.settings.URL = ssoUrl
	impl.settings.OIDCConfig.Issuer = proxyUrl

	// update session manager oidc settings
	impl.sessionManager.UpdateSettings(impl.settings, impl.dexConfig)

	// get oidc client
	oidcClient, _, err := client.GetOidcClient(impl.dexConfig, impl.selfRegistrationRolesService.CheckAndCreateUserIfConfigured, impl.sanitiseRedirectUrl)
	if err != nil {
		impl.logger.Errorw("error in getting oidc client", "err", err, "ssoUrl", ssoUrl)
		return err
	}

	// update client app config
	impl.clientApp.(*oidc.ClientApp).UpdateConfig(oidcClient)
	return nil
}
