package gitProvider

import (
	"context"
	"time"

	"github.com/devtron-labs/common-lib/securestore"
	"github.com/devtron-labs/devtron/internal/sql/constants"
	"github.com/devtron-labs/devtron/pkg/build/git/gitProvider/bean"
	"github.com/devtron-labs/devtron/pkg/lec"
)

// startLecGitHubApp 定期把组织短期凭据同步给原生 Git Sensor，学生不需要个人 PAT。
func (impl *GitRegistryConfigImpl) startLecGitHubApp() {
	app, err := lec.DefaultGitHubApp()
	if err != nil {
		impl.logger.Error("GitHub App 配置无效")
		return
	}
	if app == nil {
		return
	}
	go func() {
		for {
			if err := impl.refreshLecGitHubApp(app); err != nil {
				impl.logger.Errorw("组织 GitHub 凭据同步失败", "err", err)
			}
			time.Sleep(5 * time.Minute)
		}
	}()
}

func (impl *GitRegistryConfigImpl) refreshLecGitHubApp(app *lec.GitHubApp) error {
	token, err := app.Token(context.Background(), "")
	if err != nil {
		return err
	}
	providers, err := impl.gitProviderRepo.FindAll()
	if err != nil {
		return err
	}
	for _, provider := range providers {
		if provider.Name != lec.GitHubProviderName || provider.Deleted {
			continue
		}
		provider.AccessToken = securestore.ToEncryptedString(token)
		provider.UpdatedOn = time.Now()
		if err = impl.gitProviderRepo.Update(&provider); err != nil {
			return err
		}
		return impl.UpdateGitSensor(&provider)
	}
	_, err = impl.Create(&bean.GitRegistry{Name: lec.GitHubProviderName, Url: "https://github.com/lec-org", UserName: "x-access-token", AccessToken: token, AuthMode: constants.AUTH_MODE_ACCESS_TOKEN, Active: true, UserId: 1, GitHostId: 1, EnableTLSVerification: true})
	return err
}
