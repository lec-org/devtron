package lec

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const GitHubProviderName = "lec-org GitHub App"

// GitHubApp 以组织安装身份访问仓库；私钥只从挂载的 Secret 文件读取，不进入数据库和响应。
type GitHubApp struct {
	appID, installationID, organization string
	key                                 *rsa.PrivateKey
	client                              *http.Client
	endpoint                            string
	mu                                  sync.Mutex
	token                               string
	expires                             time.Time
}

var defaultGitHub struct {
	once  sync.Once
	value *GitHubApp
	err   error
}

func DefaultGitHubApp() (*GitHubApp, error) {
	defaultGitHub.once.Do(func() {
		appID, installationID, keyFile := os.Getenv("LEC_GITHUB_APP_ID"), os.Getenv("LEC_GITHUB_INSTALLATION_ID"), os.Getenv("LEC_GITHUB_PRIVATE_KEY_FILE")
		if appID == "" && installationID == "" && keyFile == "" {
			return
		}
		if !regexp.MustCompile(`^[0-9]+$`).MatchString(appID) || !regexp.MustCompile(`^[0-9]+$`).MatchString(installationID) || os.Getenv("LEC_GITHUB_ORG") != "lec-org" {
			defaultGitHub.err = errors.New("GitHub App 的组织或安装配置不完整")
			return
		}
		pem, err := os.ReadFile(keyFile)
		if err != nil {
			defaultGitHub.err = errors.New("无法读取 GitHub App 私钥")
			return
		}
		key, err := jwt.ParseRSAPrivateKeyFromPEM(pem)
		if err != nil {
			defaultGitHub.err = errors.New("GitHub App 私钥格式错误")
			return
		}
		defaultGitHub.value = &GitHubApp{appID: appID, installationID: installationID, organization: "lec-org", key: key, client: &http.Client{Timeout: 20 * time.Second}, endpoint: "https://api.github.com"}
	})
	return defaultGitHub.value, defaultGitHub.err
}

func (app *GitHubApp) request(ctx context.Context, path, method, token string, payload, result interface{}) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, app.endpoint+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("Content-Type", "application/json")
	response, err := app.client.Do(request)
	if err != nil {
		return errors.New("暂时无法连接 GitHub")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("GitHub App 请求失败（%d）", response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 10<<20)).Decode(result)
}

// Token 给服务端提供组织 Token；repo 非空时为构建任务签发仅限该仓库的一小时凭据。
func (app *GitHubApp) Token(ctx context.Context, repo string) (string, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if repo == "" && app.token != "" && time.Until(app.expires) > 15*time.Minute {
		return app.token, nil
	}
	if repo != "" && (!repositoryName.MatchString(repo) || repo == "." || repo == "..") {
		return "", errors.New("仓库名称无效")
	}
	now := time.Now()
	assertion := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": app.appID, "iat": now.Add(-time.Minute).Unix(), "exp": now.Add(9 * time.Minute).Unix()})
	appToken, err := assertion.SignedString(app.key)
	if err != nil {
		return "", errors.New("无法签发 GitHub App 身份")
	}
	var installation struct {
		AppID      int64  `json:"app_id"`
		TargetType string `json:"target_type"`
		Account    struct {
			Login string `json:"login"`
		} `json:"account"`
	}
	path := "/app/installations/" + app.installationID
	if err = app.request(ctx, path, "GET", appToken, nil, &installation); err != nil {
		return "", err
	}
	if fmt.Sprint(installation.AppID) != app.appID || installation.TargetType != "Organization" || !strings.EqualFold(installation.Account.Login, app.organization) {
		return "", errors.New("GitHub App 没有安装到 lec-org 组织")
	}
	payload := map[string]interface{}{"permissions": map[string]string{"contents": "read", "metadata": "read"}}
	if repo != "" {
		payload["repositories"] = []string{repo}
	}
	var issued struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err = app.request(ctx, path+"/access_tokens", "POST", appToken, payload, &issued); err != nil {
		return "", err
	}
	if issued.Token == "" || time.Until(issued.ExpiresAt) < 5*time.Minute || time.Until(issued.ExpiresAt) > 65*time.Minute {
		return "", errors.New("GitHub App 返回的凭据有效期异常")
	}
	if repo == "" {
		app.token, app.expires = issued.Token, issued.ExpiresAt
	}
	return issued.Token, nil
}

type GitHubRepository struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"fullName"`
	URL           string `json:"url"`
	DefaultBranch string `json:"defaultBranch"`
	Private       bool   `json:"private"`
}

func (app *GitHubApp) Repositories(ctx context.Context) ([]GitHubRepository, error) {
	token, err := app.Token(ctx, "")
	if err != nil {
		return nil, err
	}
	items := make([]GitHubRepository, 0)
	for page := 1; page <= 100; page++ {
		var response struct {
			Repositories []struct {
				ID            int64
				Name          string
				FullName      string `json:"full_name"`
				CloneURL      string `json:"clone_url"`
				DefaultBranch string `json:"default_branch"`
				Private       bool
				Archived      bool
				Disabled      bool
				Owner         struct{ Login string }
			} `json:"repositories"`
		}
		if err := app.request(ctx, fmt.Sprintf("/installation/repositories?per_page=100&page=%d", page), "GET", token, nil, &response); err != nil {
			return nil, err
		}
		for _, repository := range response.Repositories {
			if repository.Archived || repository.Disabled || !strings.EqualFold(repository.Owner.Login, app.organization) {
				continue
			}
			items = append(items, GitHubRepository{repository.ID, repository.Name, repository.FullName, repository.CloneURL, repository.DefaultBranch, repository.Private})
		}
		if len(response.Repositories) < 100 {
			return items, nil
		}
	}
	return nil, errors.New("GitHub 仓库数量超过当前查询上限")
}
