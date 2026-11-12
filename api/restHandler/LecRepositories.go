package restHandler

import (
	"github.com/devtron-labs/devtron/api/restHandler/common"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"github.com/devtron-labs/devtron/pkg/lec"
	"net/http"
)

// LecRepositories 只向管理员展示组织 App 已获授权的仓库，不发送任何 GitHub 凭据。
func (impl GitProviderRestHandlerImpl) LecRepositories(w http.ResponseWriter, r *http.Request) {
	userID, err := impl.userAuthService.GetLoggedInUser(r)
	if userID == 0 || err != nil {
		common.HandleUnauthorized(w, r)
		return
	}
	if !impl.enforcer.Enforce(r.Header.Get("token"), casbin.ResourceGlobal, casbin.ActionGet, "*") {
		common.WriteJsonResp(w, lec.ErrIdentity, nil, http.StatusForbidden)
		return
	}
	app, err := lec.DefaultGitHubApp()
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusServiceUnavailable)
		return
	}
	response := struct {
		Enabled      bool                   `json:"enabled"`
		Organization string                 `json:"organization"`
		Repositories []lec.GitHubRepository `json:"repositories"`
	}{Organization: "lec-org", Repositories: make([]lec.GitHubRepository, 0)}
	if app != nil {
		response.Enabled = true
		response.Repositories, err = app.Repositories(r.Context())
		if err != nil {
			common.WriteJsonResp(w, err, nil, http.StatusBadGateway)
			return
		}
	}
	common.WriteJsonResp(w, nil, response, http.StatusOK)
}
