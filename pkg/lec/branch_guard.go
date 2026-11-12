package lec

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var ErrCommitOutsideBranch = errors.New("提交不属于流水线配置的分支")
var fullCommitHash = regexp.MustCompile(`^[0-9a-f]{40}$`)

// ChangesAfterCommit 确认提交属于目标分支，并返回之后的文件变动；超过 API 文件上限时不允许据此跳过检查。
func (app *GitHubApp) ChangesAfterCommit(ctx context.Context, repo, branch, commit, token string) ([]string, bool, error) {
	if !repositoryName.MatchString(repo) || repo == "." || repo == ".." || branch == "" || !fullCommitHash.MatchString(commit) {
		return nil, false, errors.New("仓库、分支或完整提交编号无效")
	}
	var comparison struct {
		Status string `json:"status"`
		Files  []struct {
			Filename         string `json:"filename"`
			PreviousFilename string `json:"previous_filename"`
		} `json:"files"`
	}
	path := "/repos/" + app.organization + "/" + repo + "/compare/" + commit + "..." + url.PathEscape(branch)
	if err := app.request(ctx, path, "GET", token, nil, &comparison); err != nil {
		return nil, false, err
	}
	if comparison.Status == "identical" {
		return nil, true, nil
	}
	if comparison.Status != "ahead" {
		return nil, false, ErrCommitOutsideBranch
	}
	paths := make([]string, 0, len(comparison.Files))
	for _, file := range comparison.Files {
		paths = append(paths, file.Filename)
		if file.PreviousFilename != "" {
			paths = append(paths, file.PreviousFilename)
		}
	}
	return paths, len(comparison.Files) < 300, nil
}

// HasRelevantChanges 保持 Web/API 的独立发布；无法安全解析的规则保守要求重建分支最新版本。
func HasRelevantChanges(paths, filters []string) bool {
	if len(paths) == 0 {
		return false
	}
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if strings.ContainsAny(filter, "!*") {
			return true
		}
		pattern, err := regexp.Compile(filter)
		if err != nil {
			return true
		}
		for _, path := range paths {
			if pattern.MatchString(path) {
				return true
			}
		}
	}
	return false
}
