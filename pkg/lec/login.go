package lec

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	nativeoidc "github.com/devtron-labs/authenticator/oidc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

// LoginClient 复用已有 OIDC/OAuth2 依赖，额外校验 PKCE、nonce 和工作室成员资格。
type LoginClient struct {
	resolver   *IdentityResolver
	store      *sessions.CookieStore
	verifyUser func(jwt.MapClaims) bool
	mu         sync.Mutex
	provider   *gooidc.Provider
}

func NewLoginClient(resolver *IdentityResolver, verifyUser func(jwt.MapClaims) bool) *LoginClient {
	hash := sha256.Sum256([]byte("lec-oidc-sign\x00" + resolver.Config.ClientSecret))
	block := sha256.Sum256([]byte("lec-oidc-encrypt\x00" + resolver.Config.ClientSecret))
	store := sessions.NewCookieStore(hash[:], block[:])
	store.Options = &sessions.Options{Path: "/orchestrator/auth/callback", MaxAge: 300, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode}
	return &LoginClient{resolver: resolver, store: store, verifyUser: verifyUser}
}

func (l *LoginClient) configuration(request *http.Request) (*oauth2.Config, *gooidc.Provider, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.provider == nil {
		provider, err := gooidc.NewProvider(gooidc.ClientContext(request.Context(), l.resolver.httpClient), l.resolver.Config.Issuer)
		if err != nil {
			return nil, nil, err
		}
		l.provider = provider
	}
	config := &oauth2.Config{ClientID: l.resolver.Config.ClientID, ClientSecret: l.resolver.Config.ClientSecret,
		RedirectURL: l.resolver.Config.Origin + "/orchestrator/auth/callback", Endpoint: l.provider.Endpoint(), Scopes: []string{"openid", "profile"}}
	return config, l.provider, nil
}

func randomState() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

// ReturnURL 仅接受本平台的控制台地址，拒绝外部站点及协议相对地址。
func (l *LoginClient) ReturnURL(value string) string {
	fallback := l.resolver.Config.Origin + "/dashboard/application-management/devtron-app/list"
	u, err := url.Parse(value)
	origin, _ := url.Parse(l.resolver.Config.Origin)
	if err != nil || u.User != nil || u.Opaque != "" || (u.Host != "" && (u.Scheme != origin.Scheme || u.Host != origin.Host)) || (u.Scheme != "" && u.Scheme != origin.Scheme) {
		return fallback
	}
	if strings.HasPrefix(u.Path, "/orchestrator/dashboard") {
		u.Path = strings.TrimPrefix(u.Path, "/orchestrator")
	}
	u.Path, u.RawPath = path.Clean(u.Path), ""
	if u.Path != "/dashboard" && !strings.HasPrefix(u.Path, "/dashboard/") {
		return fallback
	}
	u.Scheme, u.Host = origin.Scheme, origin.Host
	return u.String()
}

func (l *LoginClient) HandleLogin(w http.ResponseWriter, r *http.Request) {
	config, _, err := l.configuration(r)
	if err != nil {
		http.Error(w, "暂时无法连接 LecSSO，请稍后重试", http.StatusServiceUnavailable)
		return
	}
	state, err := randomState()
	if err != nil {
		http.Error(w, "无法创建登录会话", http.StatusInternalServerError)
		return
	}
	nonce, err := randomState()
	if err != nil {
		http.Error(w, "无法创建登录会话", http.StatusInternalServerError)
		return
	}
	verifier := oauth2.GenerateVerifier()
	session, _ := l.store.New(r, "lec-oidc-state")
	session.Values["state"], session.Values["nonce"], session.Values["verifier"] = state, nonce, verifier
	session.Values["return"] = l.ReturnURL(r.URL.Query().Get("return_url"))
	if err = session.Save(r, w); err != nil {
		http.Error(w, "无法保存登录会话", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, config.AuthCodeURL(state, gooidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

func (l *LoginClient) HandleCallback(w http.ResponseWriter, r *http.Request) {
	session, err := l.store.Get(r, "lec-oidc-state")
	state, _ := session.Values["state"].(string)
	nonce, _ := session.Values["nonce"].(string)
	verifier, _ := session.Values["verifier"].(string)
	returnURL, _ := session.Values["return"].(string)
	if err != nil || session.IsNew || state == "" || verifier == "" || nonce == "" || subtle.ConstantTimeCompare([]byte(state), []byte(r.URL.Query().Get("state"))) != 1 || r.URL.Query().Get("code") == "" || r.URL.Query().Get("error") != "" {
		http.Error(w, "登录会话已失效，请重新登录", http.StatusBadRequest)
		return
	}
	session.Options.MaxAge = -1
	if err = session.Save(r, w); err != nil {
		http.Error(w, "无法结束登录会话", http.StatusInternalServerError)
		return
	}
	config, provider, err := l.configuration(r)
	if err != nil {
		http.Error(w, "暂时无法连接 LecSSO", http.StatusServiceUnavailable)
		return
	}
	ctx := gooidc.ClientContext(r.Context(), l.resolver.httpClient)
	token, err := config.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		http.Error(w, "LecSSO 授权失败，请重新登录", http.StatusUnauthorized)
		return
	}
	rawToken, ok := token.Extra("id_token").(string)
	if !ok || len(rawToken) > 32768 {
		http.Error(w, "LecSSO 身份凭据无效", http.StatusUnauthorized)
		return
	}
	idToken, err := provider.Verifier(&gooidc.Config{ClientID: config.ClientID}).Verify(ctx, rawToken)
	if err != nil {
		http.Error(w, "LecSSO 身份校验失败", http.StatusUnauthorized)
		return
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(nonce)) != 1 {
		http.Error(w, "登录会话校验失败，请重新登录", http.StatusUnauthorized)
		return
	}
	var claims jwt.MapClaims
	if err = idToken.Claims(&claims); err != nil {
		http.Error(w, "LecSSO 身份凭据无效", http.StatusUnauthorized)
		return
	}
	if _, err = l.resolver.ResolveVerifiedClaims(r.Context(), claims); err != nil || !l.verifyUser(claims) {
		if errors.Is(err, ErrNotMember) {
			http.Error(w, ErrNotMember.Error(), http.StatusForbidden)
		} else {
			http.Error(w, "当前账号未获准访问发布平台，请联系管理员", http.StatusForbidden)
		}
		return
	}
	// 保持 Devtron 前端读取 argocd.token 的既有协议；临时授权状态使用单独的 HttpOnly 加密 Cookie。
	cookies, err := nativeoidc.MakeCookieMetadata(nativeoidc.AuthCookieName, rawToken, "path=/", "SameSite=lax", "Secure")
	if err != nil {
		http.Error(w, "无法保存登录凭据", http.StatusInternalServerError)
		return
	}
	for _, cookie := range cookies {
		w.Header().Add("Set-Cookie", cookie)
	}
	http.Redirect(w, r, l.ReturnURL(returnURL), http.StatusSeeOther)
}
