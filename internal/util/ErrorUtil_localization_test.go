package util

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestApiErrorChineseWireMessage(t *testing.T) {
	err := NewApiError(http.StatusForbidden, "unmapped external failure", "raw diagnostic")
	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	var result map[string]interface{}
	if marshalErr = json.Unmarshal(data, &result); marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if result["userMessage"] != "没有执行此操作的权限。" || result["internalMessage"] != "raw diagnostic" || result["code"] != "403" {
		t.Fatalf("接口提示或诊断字段不正确：%s", data)
	}
	if err.Error() != "raw diagnostic" || err.UserMessage != "unmapped external failure" {
		t.Fatal("序列化不应改变服务内部的错误比较")
	}
	if _, exists := result["HttpStatusCode"]; exists {
		t.Fatal("内部状态字段不应出现在 JSON 中")
	}
}
