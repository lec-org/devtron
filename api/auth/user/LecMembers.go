package user

import (
	"github.com/devtron-labs/devtron/api/restHandler/common"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"github.com/devtron-labs/devtron/pkg/auth/user/bean"
	"github.com/devtron-labs/devtron/pkg/lec"
	"net/http"
)

// LecMembers 位于成员权限管理页；只有平台管理员可以查询可授权的正式成员。
func (handler UserRestHandlerImpl) LecMembers(w http.ResponseWriter, r *http.Request) {
	userID, err := handler.userService.GetLoggedInUser(r)
	if err != nil || userID == 0 {
		common.HandleUnauthorized(w, r)
		return
	}
	if !handler.enforcer.Enforce(r.Header.Get("token"), casbin.ResourceGlobal, casbin.ActionGet, "*") {
		common.WriteJsonResp(w, lec.ErrIdentity, nil, http.StatusForbidden)
		return
	}
	resolver, err := lec.DefaultIdentity()
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusServiceUnavailable)
		return
	}
	response := struct {
		Enabled bool           `json:"enabled"`
		Members []lec.Identity `json:"members"`
	}{Members: make([]lec.Identity, 0)}
	if resolver != nil {
		response.Enabled = true
		response.Members, err = resolver.ListMembers(r.Context())
		if err != nil {
			common.WriteJsonResp(w, err, nil, http.StatusServiceUnavailable)
			return
		}
	}
	common.WriteJsonResp(w, nil, response, http.StatusOK)
}

// fillMemberNames 只修饰展示字段，查询失败不改变账号键和既有权限。
func (handler UserRestHandlerImpl) fillMemberNames(r *http.Request, users []bean.UserInfo) {
	resolver, err := lec.DefaultIdentity()
	if err != nil || resolver == nil {
		return
	}
	members, err := resolver.ListMembers(r.Context())
	if err != nil {
		return
	}
	labels := make(map[string]string, len(members))
	for _, member := range members {
		label := member.Name
		if label == "" {
			label = member.Username
		}
		if label == "" {
			label = "未命名成员"
		}
		labels[member.Account] = label
	}
	for i := range users {
		users[i].DisplayName = labels[users[i].EmailId]
	}
}
