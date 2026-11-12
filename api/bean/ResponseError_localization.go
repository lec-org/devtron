package bean

import (
	"encoding/json"
	"github.com/devtron-labs/devtron/internal/i18n"
)

// MarshalJSON 与统一响应保持相同的中文显示规则，保留原始诊断字段。
func (e ApiError) MarshalJSON() ([]byte, error) {
	type wireError ApiError
	result := wireError(e)
	if message, ok := e.UserMessage.(string); ok {
		if result.InternalMessage == "" {
			result.InternalMessage = message
		}
		result.UserMessage = i18n.UserMessage(message, 0)
	}
	result.UserDetailMessage = i18n.UserMessage(e.UserDetailMessage, 0)
	return json.Marshal(result)
}
