// Package lec 维护 LEC 的身份与组织边界，不从 SSO 角色推导应用发布权限。
package lec

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/devtron-labs/authenticator/middleware"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

var ErrIdentity = errors.New("登录身份无效，请重新使用 LecSSO 登录")
var ErrNotMember = errors.New("只有工作室正式成员可以访问发布平台")

// IdentityConfig 的登录客户端与独立成员查询凭据由部署 Secret 提供。
type IdentityConfig struct {
	Issuer, Origin, ClientID, ClientSecret                     string
	ManagementEndpoint, ManagementResource                     string
	ManagementClientID, ManagementClientSecret, OrganizationID string
}

type Identity struct {
	Subject   string `json:"subject"`
	Account   string `json:"account"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	Suspended bool   `json:"isSuspended"`
}

type memberProfile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Username    string `json:"username"`
	IsSuspended bool   `json:"isSuspended"`
}

// ListMembers 供管理员选择成员，只返回稳定身份键和展示名，不返回邮箱、手机号或凭据。
func (r *IdentityResolver) ListMembers(ctx context.Context) ([]Identity, error) {
	members := make([]Identity, 0)
	for page := 1; page <= 100; page++ {
		var profiles []memberProfile
		if err := r.get(ctx, fmt.Sprintf("/organizations/%s/users?page=%d&page_size=100", url.PathEscape(r.Config.OrganizationID), page), &profiles); err != nil {
			return nil, err
		}
		for _, profile := range profiles {
			if profile.ID == "" || profile.IsSuspended {
				continue
			}
			members = append(members, Identity{Subject: profile.ID, Account: AccountKey(r.Config.Issuer, profile.ID), Name: profile.Name, Username: profile.Username})
		}
		if len(profiles) < 100 {
			return members, nil
		}
	}
	return nil, errors.New("工作室成员数量超过当前查询上限")
}

// ValidateAccounts 在授予权限前重新核对成员，不能把手填邮箱当作 LecSSO 身份。
func ValidateAccounts(ctx context.Context, accounts string) error {
	resolver, err := DefaultIdentity()
	if err != nil || resolver == nil {
		return err
	}
	members, err := resolver.ListMembers(ctx)
	if err != nil {
		return err
	}
	allowed := make(map[string]bool, len(members))
	for _, member := range members {
		allowed[member.Account] = true
	}
	for _, account := range strings.Split(accounts, ",") {
		if !allowed[account] {
			return ErrNotMember
		}
	}
	return nil
}

type cachedIdentity struct {
	identity Identity
	until    time.Time
}
type IdentityResolver struct {
	Config     IdentityConfig
	client     *http.Client
	httpClient *http.Client
	mu         sync.Mutex
	cache      map[string]cachedIdentity
}

var defaultIdentity struct {
	once  sync.Once
	value *IdentityResolver
	err   error
}

// DefaultIdentity 关闭时保留上游行为；启用但缺少配置时拒绝启动或认证。
func DefaultIdentity() (*IdentityResolver, error) {
	defaultIdentity.once.Do(func() {
		if os.Getenv("LEC_SSO_ENABLED") != "true" {
			return
		}
		defaultIdentity.value, defaultIdentity.err = NewIdentityResolver(IdentityConfig{
			Issuer: os.Getenv("LEC_SSO_ISSUER"), Origin: os.Getenv("LEC_PUBLIC_ORIGIN"),
			ClientID: os.Getenv("LEC_SSO_CLIENT_ID"), ClientSecret: os.Getenv("LEC_SSO_CLIENT_SECRET"),
			ManagementEndpoint: os.Getenv("LEC_SSO_MANAGEMENT_ENDPOINT"), ManagementResource: os.Getenv("LEC_SSO_MANAGEMENT_RESOURCE"),
			ManagementClientID: os.Getenv("LEC_SSO_MANAGEMENT_CLIENT_ID"), ManagementClientSecret: os.Getenv("LEC_SSO_MANAGEMENT_CLIENT_SECRET"),
			OrganizationID: os.Getenv("LEC_SSO_ORGANIZATION_ID"),
		}, &http.Client{Timeout: 10 * time.Second})
	})
	return defaultIdentity.value, defaultIdentity.err
}

func NewIdentityResolver(config IdentityConfig, transport *http.Client) (*IdentityResolver, error) {
	for _, endpoint := range []string{config.Issuer, config.Origin, config.ManagementEndpoint} {
		u, err := url.Parse(endpoint)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, errors.New("LecSSO 必须配置受信任的 HTTPS 地址")
		}
	}
	origin, _ := url.Parse(config.Origin)
	if origin.Path != "" && origin.Path != "/" {
		return nil, errors.New("发布平台地址不能包含路径")
	}
	config.Origin = strings.TrimRight(config.Origin, "/")
	if config.ClientID == "" || config.ClientSecret == "" || config.ManagementClientID == "" || config.ManagementClientSecret == "" || config.ManagementResource == "" || config.OrganizationID == "" {
		return nil, errors.New("LecSSO 客户端或成员查询配置不完整")
	}
	form := url.Values{"resource": {config.ManagementResource}}
	auth := clientcredentials.Config{ClientID: config.ManagementClientID, ClientSecret: config.ManagementClientSecret,
		TokenURL: config.Issuer + "/token", EndpointParams: form, Scopes: []string{"all"}}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, transport)
	apiClient := auth.Client(ctx)
	apiClient.Timeout = 10 * time.Second
	return &IdentityResolver{Config: config, client: apiClient, httpClient: transport, cache: make(map[string]cachedIdentity)}, nil
}

// AccountKey 只作上游 email_id 字段的兼容标识，不表示邮箱所有权，也不用于发送邮件。
func AccountKey(issuer, subject string) string {
	return fmt.Sprintf("%x@identity.lec.invalid", sha256.Sum256([]byte(issuer+"\x00"+subject)))
}

// AccountFromVerifiedClaims 在所有鉴权入口统一身份键；原管理员和 API Token 保留上游协议。
func AccountFromVerifiedClaims(claims map[string]interface{}) (string, bool, error) {
	resolver, err := DefaultIdentity()
	if err != nil {
		return "", false, err
	}
	issuer, _ := claims["iss"].(string)
	if resolver != nil && issuer != middleware.SessionManagerClaimsIssuer && issuer != middleware.ApiTokenClaimIssuer && issuer != middleware.LicenseManagerClaimIssuer {
		member, err := resolver.ResolveVerifiedClaims(context.Background(), claims)
		return member.Account, true, err
	}
	email, _ := claims["email"].(string)
	subject, _ := claims["sub"].(string)
	if email == "" && (subject == "admin" || subject == "admin:login") {
		email = "admin"
	}
	return email, false, nil
}

// ResolveVerifiedClaims 只能接收 OIDC 客户端或 SessionManager 验签后的 Claims。
func (r *IdentityResolver) ResolveVerifiedClaims(ctx context.Context, claims map[string]interface{}) (Identity, error) {
	issuer, _ := claims["iss"].(string)
	subject, _ := claims["sub"].(string)
	if issuer != r.Config.Issuer || subject == "" || len(subject) > 255 || !hasAudience(claims["aud"], r.Config.ClientID) {
		return Identity{}, ErrIdentity
	}
	if authorized, ok := claims["azp"].(string); ok && authorized != r.Config.ClientID {
		return Identity{}, ErrIdentity
	}
	return r.Member(ctx, subject)
}

func hasAudience(value interface{}, expected string) bool {
	switch audience := value.(type) {
	case string:
		return audience == expected
	case []interface{}:
		for _, item := range audience {
			if item == expected {
				return true
			}
		}
	case []string:
		for _, item := range audience {
			if item == expected {
				return true
			}
		}
	}
	return false
}

// Member 的有效缓存最多保留 15 秒；查询失败不会新增或延长授权。
func (r *IdentityResolver) Member(ctx context.Context, subject string) (Identity, error) {
	r.mu.Lock()
	cached, found := r.cache[subject]
	r.mu.Unlock()
	if found && time.Now().Before(cached.until) {
		return cached.identity, nil
	}
	var profile memberProfile
	if err := r.get(ctx, "/users/"+url.PathEscape(subject), &profile); err != nil {
		return Identity{}, err
	}
	if profile.ID != subject || profile.IsSuspended {
		return Identity{}, ErrNotMember
	}
	var organizations []struct {
		ID string `json:"id"`
	}
	if err := r.get(ctx, "/users/"+url.PathEscape(subject)+"/organizations", &organizations); err != nil {
		return Identity{}, err
	}
	for _, organization := range organizations {
		if organization.ID == r.Config.OrganizationID {
			identity := Identity{Subject: subject, Account: AccountKey(r.Config.Issuer, subject), Name: profile.Name, Username: profile.Username}
			r.mu.Lock()
			r.cache[subject] = cachedIdentity{identity, time.Now().Add(15 * time.Second)}
			r.mu.Unlock()
			return identity, nil
		}
	}
	return Identity{}, ErrNotMember
}

func (r *IdentityResolver) get(ctx context.Context, path string, result interface{}) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(r.Config.ManagementEndpoint, "/")+path, nil)
	if err != nil {
		return err
	}
	response, err := r.client.Do(request)
	if err != nil {
		return errors.New("暂时无法核对工作室成员身份，请稍后重试")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusForbidden {
		return ErrNotMember
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("成员查询失败（%d）", response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(result)
}
