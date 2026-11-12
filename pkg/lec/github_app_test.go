package lec

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestGitHubAppOrganizationAndRepositoryScope(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	owner := "lec-org"
	issued := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/app/") {
			assertion := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			parsed, err := jwt.Parse(assertion, func(token *jwt.Token) (interface{}, error) { return &key.PublicKey, nil }, jwt.WithValidMethods([]string{"RS256"}))
			if err != nil || !parsed.Valid || parsed.Claims.(jwt.MapClaims)["iss"] != "1" {
				t.Fatal("App 身份未正确签名")
			}
		}
		switch r.URL.Path {
		case "/app/installations/2":
			json.NewEncoder(w).Encode(map[string]interface{}{"app_id": 1, "target_type": "Organization", "account": map[string]string{"login": owner}})
		case "/app/installations/2/access_tokens":
			issued++
			var body struct {
				Repositories []string          `json:"repositories"`
				Permissions  map[string]string `json:"permissions"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			if body.Permissions["contents"] != "read" || body.Permissions["metadata"] != "read" || len(body.Permissions) != 2 {
				t.Fatal("申请了多余权限")
			}
			if issued == 2 && (len(body.Repositories) != 1 || body.Repositories[0] != "LecPunch") {
				t.Fatal("构建凭据没有限定单个仓库")
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"token": "installation-token", "expires_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339)})
		case "/installation/repositories":
			if r.Header.Get("Authorization") != "Bearer installation-token" {
				t.Fatal("没有使用安装身份")
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"repositories": []map[string]interface{}{
				{"id": 1, "name": "LecPunch", "full_name": "lec-org/LecPunch", "clone_url": "https://github.com/lec-org/LecPunch.git", "private": true, "default_branch": "main", "owner": map[string]string{"login": "lec-org"}},
				{"id": 2, "name": "other", "owner": map[string]string{"login": "other-org"}},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	app := &GitHubApp{appID: "1", installationID: "2", organization: "lec-org", key: key, client: server.Client(), endpoint: server.URL}
	repositories, err := app.Repositories(context.Background())
	if err != nil || len(repositories) != 1 || !repositories[0].Private || repositories[0].DefaultBranch != "main" {
		t.Fatalf("仓库同步失败: %v %v", repositories, err)
	}
	if _, err := app.Token(context.Background(), ""); err != nil || issued != 1 {
		t.Fatal("组织凭据没有复用有效缓存")
	}
	if _, err := app.Token(context.Background(), "LecPunch"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Token(context.Background(), "../other"); err == nil {
		t.Fatal("接受了非法仓库名")
	}
	owner = "other-org"
	if _, err := app.Token(context.Background(), "LecPunch"); err == nil {
		t.Fatal("接受了其他组织的安装")
	}
}
