package gitProvider

import (
	"context"
	"testing"

	"github.com/devtron-labs/common-lib/securestore"
	"github.com/devtron-labs/devtron/client/gitSensor"
	"github.com/devtron-labs/devtron/pkg/build/git/gitProvider/bean"
	"github.com/devtron-labs/devtron/pkg/build/git/gitProvider/repository"
	"go.uber.org/zap"
)

type credentialRepository struct {
	repository.GitProviderRepository
	reads int
}

func (repo *credentialRepository) FindOne(id string) (repository.GitProvider, error) {
	repo.reads++
	return repository.GitProvider{Id: 1, Active: true, AccessToken: securestore.ToEncryptedString("decrypted-installation-token")}, nil
}

func (repo *credentialRepository) ProviderExists(url string) (bool, error) { return false, nil }

func (repo *credentialRepository) Save(provider *repository.GitProvider) error {
	provider.Id = 1
	return nil
}

type credentialSensor struct {
	gitSensor.Client
	value *gitSensor.GitProvider
}

func (sensor *credentialSensor) SaveGitProvider(ctx context.Context, value *gitSensor.GitProvider) error {
	sensor.value = value
	return nil
}

func TestGitSensorReceivesReadableCredentials(t *testing.T) {
	repo, sensor := &credentialRepository{}, &credentialSensor{}
	service := GitRegistryConfigImpl{gitProviderRepo: repo, GitSensorGrpcClient: sensor}
	err := service.UpdateGitSensor(&repository.GitProvider{Id: 1, Active: true, AccessToken: securestore.ToEncryptedString("encrypted-db-value")})
	if err != nil || sensor.value.AccessToken != "decrypted-installation-token" || repo.reads != 1 {
		t.Fatal("Git Sensor 收到了数据库密文")
	}
	err = service.UpdateGitSensor(&repository.GitProvider{Id: 1, Active: false})
	if err != nil || sensor.value.Active || repo.reads != 1 {
		t.Fatal("停用操作不应重新加载已删除的提供方")
	}
}

func TestGitProviderAcceptsSystemCertificates(t *testing.T) {
	repo, sensor := &credentialRepository{}, &credentialSensor{}
	service := GitRegistryConfigImpl{logger: zap.NewNop().Sugar(), gitProviderRepo: repo, GitSensorGrpcClient: sensor}
	_, err := service.Create(&bean.GitRegistry{Url: "https://github.com/lec-org", Active: true, EnableTLSVerification: true})
	if err != nil || sensor.value == nil {
		t.Fatalf("公共 HTTPS 不应要求上传自定义证书：%v", err)
	}
}
