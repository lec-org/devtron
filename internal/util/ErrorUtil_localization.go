package util

import (
	"encoding/json"
	"github.com/devtron-labs/devtron/internal/i18n"
)

// MarshalJSON 仅在响应序列化时翻译，Error() 与服务内部的既有错误比较保持不变。
func (e ApiError) MarshalJSON() ([]byte, error) {
	type wireError ApiError
	result := wireError(e)
	if message, ok := e.UserMessage.(string); ok {
		if result.InternalMessage == "" {
			result.InternalMessage = message
		}
		result.UserMessage = i18n.UserMessage(message, e.HttpStatusCode)
	}
	result.UserDetailMessage = i18n.UserMessage(e.UserDetailMessage, e.HttpStatusCode)
	return json.Marshal(result)
}
