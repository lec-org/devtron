// Package i18n 只翻译面向使用者的消息，保留协议值、资源名称与原始诊断信息。
package i18n

import (
	"embed"
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

//go:embed zh-CN.json
var files embed.FS
var messages map[string]string
var formats []messageFormat
var placeholder = regexp.MustCompile(`%(?:\[[0-9]+\])?[+#\- .0-9]*[A-Za-z]`)

type messageFormat struct {
	source     string
	expression *regexp.Regexp
	parts      []string
}

func init() {
	data, err := files.ReadFile("zh-CN.json")
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(data, &messages); err != nil {
		panic(err)
	}
	for source, target := range messages {
		indexes := placeholder.FindAllStringIndex(source, -1)
		if len(indexes) == 0 {
			continue
		}
		targetParts := placeholder.Split(target, -1)
		if strings.Join(placeholder.FindAllString(source, -1), "|") != strings.Join(placeholder.FindAllString(target, -1), "|") {
			panic("中文消息占位符不一致：" + source)
		}
		var pattern strings.Builder
		pattern.WriteString("(?s)^")
		start := 0
		for _, index := range indexes {
			pattern.WriteString(regexp.QuoteMeta(source[start:index[0]]))
			pattern.WriteString("(.*?)")
			start = index[1]
		}
		pattern.WriteString(regexp.QuoteMeta(source[start:]))
		pattern.WriteString("$")
		formats = append(formats, messageFormat{source, regexp.MustCompile(pattern.String()), targetParts})
	}
	sort.Slice(formats, func(i, j int) bool { return len(formats[i].source) > len(formats[j].source) })
}

// Translate 保留未知字符串，适用于成功消息；不会改写令牌、枚举或资源标识符。
func Translate(message string) string {
	source := strings.TrimSpace(message)
	if value, ok := messages[source]; ok {
		return value
	}
	for _, format := range formats {
		match := format.expression.FindStringSubmatch(source)
		if match == nil {
			continue
		}
		var result strings.Builder
		for i, part := range format.parts {
			result.WriteString(part)
			if i+1 < len(match) {
				result.WriteString(match[i+1])
			}
		}
		return result.String()
	}
	return message
}

// UserMessage 为未知第三方错误提供中文提示，原始详情仍保存在 internalMessage 和服务日志。
func UserMessage(message string, status int) string {
	if strings.TrimSpace(message) == "" {
		return message
	}
	translated := Translate(message)
	if translated != message || strings.ContainsFunc(message, func(r rune) bool { return unicode.Is(unicode.Han, r) }) {
		return translated
	}
	switch status {
	case http.StatusUnauthorized:
		return "登录已失效，请重新登录。"
	case http.StatusForbidden:
		return "没有执行此操作的权限。"
	case http.StatusNotFound:
		return "未找到请求的资源。"
	case http.StatusConflict:
		return "资源状态冲突，请刷新后重试。"
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return "请求参数无效，请检查后重试。"
	default:
		return "操作失败，请重试；如仍有问题，请联系管理员查看服务日志。"
	}
}
