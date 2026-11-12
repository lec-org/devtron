package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devtron-labs/devtron/internal/util"
)

func TestChineseResponseContract(t *testing.T) {
	if _, err := util.InitLogger(); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	WriteJsonResp(response, util.NewApiError(http.StatusForbidden, "unknown external error", "permission diagnostic"), nil, http.StatusInternalServerError)
	var body struct {
		Code   int              `json:"code"`
		Status string           `json:"status"`
		Errors []*util.ApiError `json:"errors"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusForbidden || body.Code != http.StatusForbidden || body.Status != "Forbidden" || response.Header().Get("Content-Language") != "zh-CN" {
		t.Fatalf("响应协议发生变化：%s", response.Body.String())
	}
	if len(body.Errors) != 1 || body.Errors[0].UserMessage != "没有执行此操作的权限。" || body.Errors[0].InternalMessage != "permission diagnostic" {
		t.Fatalf("中文提示与原始诊断没有正确分离：%s", response.Body.String())
	}
	for _, result := range []string{"my-production-app", "eyJhbGciOiJSUzI1NiJ9.token.signature"} {
		recorder := httptest.NewRecorder()
		WriteJsonResp(recorder, nil, result, http.StatusOK)
		var success struct {
			Result string `json:"result"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &success); err != nil || success.Result != result {
			t.Fatalf("成功结果被错误翻译：%s", recorder.Body.String())
		}
	}
}
