package lec

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConfiguredBranchProvenance(t *testing.T) {
	status, count := "identical", 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer scoped-token" || r.Header.Get("Cache-Control") != "no-cache" {
			t.Error("分支核验必须使用受控凭据并重新验证缓存")
		}
		if !strings.HasSuffix(r.URL.EscapedPath(), "...feature%2Flogin") {
			t.Error("分支名称没有安全编码")
		}
		if status == "http-error" {
			w.WriteHeader(503)
			return
		}
		files := make([]map[string]string, count)
		for i := range files {
			files[i] = map[string]string{"filename": "apps/api/new.ts", "previous_filename": "apps/web/old.ts"}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"status": status, "files": files})
	}))
	defer server.Close()
	app := &GitHubApp{organization: "lec-org", endpoint: server.URL, client: server.Client()}
	check := func() ([]string, bool, error) {
		return app.ChangesAfterCommit(context.Background(), "LecPunch", "feature/login", strings.Repeat("a", 40), "scoped-token")
	}
	if paths, complete, err := check(); err != nil || !complete || len(paths) != 0 {
		t.Fatalf("分支当前提交被拒绝：%v", err)
	}
	status, count = "ahead", 1
	if paths, complete, err := check(); err != nil || !complete || len(paths) != 2 {
		t.Fatalf("没有检查重命名前后的路径：%v", err)
	}
	count = 300
	if _, complete, err := check(); err != nil || complete {
		t.Fatal("达到文件上限不能推断其他路径未变")
	}
	for _, value := range []string{"behind", "diverged", ""} {
		status = value
		if _, _, err := check(); !errors.Is(err, ErrCommitOutsideBranch) {
			t.Fatalf("接受了未合入目标分支的提交：%s", value)
		}
	}
	status = "http-error"
	if _, _, err := check(); err == nil || errors.Is(err, ErrCommitOutsideBranch) {
		t.Fatal("网络故障不能按过期提交直接跳过")
	}
	if _, _, err := app.ChangesAfterCommit(context.Background(), "LecPunch", "main", "abcdef0", "scoped-token"); err == nil {
		t.Fatal("接受了不完整提交编号")
	}
}

func TestAutomaticDeploymentChangeScope(t *testing.T) {
	filters := []string{`^apps/web/`, `^packages/shared/`, `^pnpm-lock\.yaml$`}
	for _, path := range []string{"apps/web/index.ts", "packages/shared/types.ts", "pnpm-lock.yaml"} {
		if !HasRelevantChanges([]string{path}, filters) {
			t.Fatalf("遗漏需要重新构建的路径 %s", path)
		}
	}
	if HasRelevantChanges([]string{"apps/api/main.ts", "apps/desktop/main.ts"}, filters) {
		t.Fatal("API/桌面端更新阻断了独立的 Web 发布")
	}
	if HasRelevantChanges(nil, nil) || !HasRelevantChanges([]string{"a"}, nil) {
		t.Fatal("无路径规则的应用应检查全部变更")
	}
	for _, filter := range []string{"!vendor/**", "apps/web/**", "[invalid"} {
		if !HasRelevantChanges([]string{"a"}, []string{filter}) {
			t.Fatal("无法判定的规则没有保守要求重建")
		}
	}
}
