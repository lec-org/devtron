package lec

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v4"
)

// 使用真实 TLS、JWKS 和签名，验证回调与成员边界，避免只测字符串拼接。
func TestLoginSecurityBoundary(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer, nonce, challenge, mode string
	var tokenCalls, provisionCalls int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oidc/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]interface{}{"issuer": issuer, "authorization_endpoint": issuer + "/auth", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/jwks", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/oidc/jwks":
			json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "test", Algorithm: "RS256", Use: "sig"}}})
		case "/oidc/token":
			r.ParseForm()
			if r.Form.Get("grant_type") == "client_credentials" {
				json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "member-read", "token_type": "Bearer", "expires_in": 3600})
				return
			}
			tokenCalls++
			digest := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if r.Form.Get("code") != "one-use-code" || base64.RawURLEncoding.EncodeToString(digest[:]) != challenge {
				t.Error("PKCE 不匹配")
			}
			claims := jwt.MapClaims{"iss": issuer, "sub": "member-1", "aud": "devtron", "nonce": nonce, "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix()}
			if mode == "nonce" {
				claims["nonce"] = "other-session"
			}
			if mode == "audience" {
				claims["aud"] = "other-app"
			}
			if mode == "issuer" {
				claims["iss"] = "https://other.invalid/oidc"
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "test"
			signingKey := key
			if mode == "signature" {
				signingKey, _ = rsa.GenerateKey(rand.Reader, 2048)
			}
			signed, err := token.SignedString(signingKey)
			if err != nil {
				t.Fatal(err)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "unused", "token_type": "Bearer", "id_token": signed})
		case "/api/users/member-1":
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "member-1", "name": "测试成员", "isSuspended": mode == "suspended"})
		case "/api/users/member-1/organizations":
			org := "lec-studio"
			if mode == "outsider" {
				org = "other"
			}
			json.NewEncoder(w).Encode([]map[string]string{{"id": org}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	issuer = server.URL + "/oidc"
	for _, test := range []struct {
		mode   string
		status int
	}{
		{"valid", 303}, {"state", 400}, {"nonce", 401}, {"audience", 401}, {"issuer", 401}, {"signature", 401}, {"suspended", 403}, {"outsider", 403}, {"disabled", 403},
	} {
		t.Run(test.mode, func(t *testing.T) {
			mode = test.mode
			resolver, err := NewIdentityResolver(IdentityConfig{Issuer: issuer, Origin: "https://deploy.lec.invalid", ClientID: "devtron", ClientSecret: "secret", ManagementEndpoint: server.URL + "/api", ManagementResource: "management", ManagementClientID: "reader", ManagementClientSecret: "reader-secret", OrganizationID: "lec-studio"}, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			login := NewLoginClient(resolver, func(claims jwt.MapClaims) bool { provisionCalls++; return mode != "disabled" })
			start := httptest.NewRecorder()
			login.HandleLogin(start, httptest.NewRequest("GET", "https://deploy.lec.invalid/orchestrator/auth/login?return_url=https://evil.invalid/", nil))
			if start.Code != 302 {
				t.Fatalf("登录入口: %d %s", start.Code, start.Body.String())
			}
			redirect, _ := url.Parse(start.Header().Get("Location"))
			nonce, challenge = redirect.Query().Get("nonce"), redirect.Query().Get("code_challenge")
			if nonce == "" || challenge == "" || redirect.Query().Get("code_challenge_method") != "S256" {
				t.Fatal("缺少 nonce 或 PKCE")
			}
			cookie := start.Result().Cookies()[0]
			if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
				t.Fatal("状态 Cookie 属性错误")
			}
			state := redirect.Query().Get("state")
			if mode == "state" {
				state = "forged"
			}
			callback := httptest.NewRequest("GET", "https://deploy.lec.invalid/orchestrator/auth/callback?code=one-use-code&state="+url.QueryEscape(state), nil)
			callback.AddCookie(cookie)
			before, provisionBefore := tokenCalls, provisionCalls
			result := httptest.NewRecorder()
			login.HandleCallback(result, callback)
			if result.Code != test.status {
				t.Fatalf("回调: %d %s", result.Code, result.Body.String())
			}
			if mode == "state" && tokenCalls != before {
				t.Fatal("非法状态不应交换 Token")
			}
			if test.status == 401 && provisionCalls != provisionBefore {
				t.Fatal("校验失败不应创建账号")
			}
			if test.status == 303 {
				if !strings.HasPrefix(result.Header().Get("Location"), "https://deploy.lec.invalid/dashboard/") {
					t.Fatal("外部重定向未拒绝")
				}
				if !strings.Contains(strings.Join(result.Header().Values("Set-Cookie"), " "), "argocd.token=") {
					t.Fatal("未建立 Devtron 登录态")
				}
				member, err := resolver.ResolveVerifiedClaims(context.Background(), jwt.MapClaims{"iss": issuer, "sub": "member-1", "aud": "devtron", "email": "changed@example.com"})
				if err != nil || member.Account != AccountKey(issuer, "member-1") {
					t.Fatal("身份不应依赖可变邮箱")
				}
			}
		})
	}
}

func TestIdentityConfigurationAndReturnURL(t *testing.T) {
	login := &LoginClient{resolver: &IdentityResolver{Config: IdentityConfig{Origin: "https://deploy.lec.invalid"}}}
	for _, input := range []string{"//evil.invalid/dashboard", "https://evil.invalid/dashboard", "javascript:alert(1)", "/other"} {
		if got := login.ReturnURL(input); got != "https://deploy.lec.invalid/dashboard/application-management/devtron-app/list" {
			t.Fatalf("未拒绝地址 %q: %s", input, got)
		}
	}
	if got := login.ReturnURL("/orchestrator/dashboard/apps?app=1"); got != "https://deploy.lec.invalid/dashboard/apps?app=1" {
		t.Fatal(got)
	}
	if AccountKey("https://sso/oidc", "1") == AccountKey("https://other/oidc", "1") {
		t.Fatal("身份未区分 issuer")
	}
	if _, err := NewIdentityResolver(IdentityConfig{Issuer: "http://sso/oidc"}, http.DefaultClient); err == nil {
		t.Fatal("接受了 HTTP 身份源")
	}
}
