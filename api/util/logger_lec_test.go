package util

import "testing"

func TestCredentialAuditDoesNotRecordSecrets(t *testing.T) {
	for _, path := range []string{"/orchestrator/auth/callback", "/orchestrator/api/v1/session", "/orchestrator/git/provider", "/orchestrator/docker/registry"} {
		audit := &AuditLoggerDTO{UrlPath: path, QueryParams: "code=private", RequestPayload: []byte(`{"password":"private"}`), UserEmail: "admin"}
		hideCredentialDetails(audit)
		if audit.QueryParams != "" || string(audit.RequestPayload) != "[凭据已隐藏]" || audit.UserEmail != "admin" {
			t.Fatal(path)
		}
	}
}
