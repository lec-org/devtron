package i18n

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestResponseMessagesPreserveData(t *testing.T) {
	const source = `{"result":{"id":9007199254740993,"name":"App synced successfully.","token":"opaque-token","items":[{"message":"App synced successfully."}],"metadata":{"description":"user input"}}}`
	data, err := MarshalResponse(json.RawMessage(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, unchanged := range []string{`9007199254740993`, `"name":"App synced successfully."`, `"token":"opaque-token"`, `"description":"user input"`} {
		if !strings.Contains(string(data), unchanged) {
			t.Errorf("资源数据发生变化，缺少 %s：%s", unchanged, data)
		}
	}
	if !strings.Contains(string(data), `"message":"应用同步成功。"`) {
		t.Errorf("结果消息未中文化：%s", data)
	}
}

func TestTranslate(t *testing.T) {
	for _, message := range []string{"my-production-app", "eyJhbGciOiJSUzI1NiJ9.test.signature", ""} {
		if Translate(message) != message {
			t.Errorf("不应改写资源或令牌：%q", message)
		}
	}
	const source = "requested by %d, error: %v"
	got := Translate(fmt.Sprintf(source, 42, "connection refused"))
	if got != "由用户 42 发起，错误：connection refused" {
		t.Errorf("格式化消息未正确保留参数：%q", got)
	}
}

func TestUnknownErrorFallback(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict, http.StatusBadRequest, http.StatusInternalServerError} {
		if got := UserMessage("unmapped external failure", status); !strings.ContainsAny(got, "登权资参操") {
			t.Errorf("状态 %d 没有中文提示：%q", status, got)
		}
	}
	if got := UserMessage("服务暂不可用", 500); got != "服务暂不可用" {
		t.Errorf("已有中文消息被改写：%q", got)
	}
}
