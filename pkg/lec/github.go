package lec

import (
	"errors"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var repositoryName = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// ValidateRepository 在变量解析后校验，不能靠界面筛选阻止组织外的仓库。
func ValidateRepository(value string) error {
	organization := os.Getenv("LEC_GITHUB_ORG")
	if organization == "" {
		return nil
	}
	if strings.HasPrefix(value, "git@github.com:") {
		value = "ssh://git@github.com/" + strings.TrimPrefix(value, "git@github.com:")
	}
	u, err := url.Parse(value)
	if err != nil || u.Host != "github.com" || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || (u.Scheme != "https" && u.Scheme != "ssh") {
		return errors.New("代码仓库必须来自已绑定的 GitHub 组织")
	}
	if u.Scheme == "https" && u.User != nil {
		return errors.New("仓库地址不能包含凭据")
	}
	if u.Scheme == "ssh" && (u.User == nil || u.User.String() != "git") {
		return errors.New("请使用标准 GitHub SSH 仓库地址")
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 2 || !strings.EqualFold(parts[0], organization) || !repositoryName.MatchString(parts[1]) || parts[1] == "." || parts[1] == ".." || strings.TrimSuffix(parts[1], ".git") == "" {
		return errors.New("只能绑定 " + organization + " 组织中的代码仓库")
	}
	return nil
}
