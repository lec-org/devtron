package types

import "testing"

func TestWorkflowProxyEnvironment(t *testing.T) {
	keys := []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy"}
	for _, key := range keys {
		t.Setenv(key, "")
	}
	request := &WorkflowRequest{}
	initial := request.getContainerEnvVariables(&CiCdConfig{}, []byte("{}"))
	t.Setenv("HTTPS_PROXY", "http://proxy.example:8118")
	t.Setenv("NO_PROXY", "localhost,.svc,.cluster.local,10.0.0.0/8")
	variables := request.getContainerEnvVariables(&CiCdConfig{}, []byte("{}"))
	if len(variables) != len(initial)+2 {
		t.Fatal("构建任务没有继承已配置的代理与直连规则")
	}
	values := make(map[string]string)
	for _, variable := range variables {
		values[variable.Name] = variable.Value
	}
	if values["HTTPS_PROXY"] != "http://proxy.example:8118" || values["NO_PROXY"] != "localhost,.svc,.cluster.local,10.0.0.0/8" || values["CI_CD_EVENT"] != "{}" {
		t.Fatal("代理配置改变了原有任务数据或丢失了内网直连规则")
	}
}
