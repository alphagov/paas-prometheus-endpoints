package authenticator_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	cfApiUrl  = "http://cf.api"
	uaaApiUrl = "http://uaa.api"
)

func TestAuthenticator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authenticator Suite")
}

func setupCfV2InfoHttpmock() {
	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("%s/v2/info", cfApiUrl),
		httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
			"token_endpoint": fmt.Sprintf("%s", uaaApiUrl),
		}),
	)
}

func setupSuccessfulUaaOauthLoginHttpmock() {
	httpmock.RegisterResponder(
		"POST",
		fmt.Sprintf("%s/oauth/token", uaaApiUrl),
		httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
			// Copy and pasted from UAA docs
			"access_token":  "acb6803a48114d9fb4761e403c17f812",
			"token_type":    "bearer",
			"id_token":      "eyJhbGciOiJIUzI1NiIsImprdSI6Imh0dHBzOi8vbG9jYWxob3N0OjgwODAvdWFhL3Rva2VuX2tleXMiLCJraWQiOiJsZWdhY3ktdG9rZW4ta2V5IiwidHlwIjoiSldUIn0.eyJzdWIiOiIwNzYzZTM2MS02ODUwLTQ3N2ItYjk1Ny1iMmExZjU3MjczMTQiLCJhdWQiOlsibG9naW4iXSwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3VhYS9vYXV0aC90b2tlbiIsImV4cCI6MTU1NzgzMDM4NSwiaWF0IjoxNTU3Nzg3MTg1LCJhenAiOiJsb2dpbiIsInNjb3BlIjpbIm9wZW5pZCJdLCJlbWFpbCI6IndyaHBONUB0ZXN0Lm9yZyIsInppZCI6InVhYSIsIm9yaWdpbiI6InVhYSIsImp0aSI6ImFjYjY4MDNhNDgxMTRkOWZiNDc2MWU0MDNjMTdmODEyIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImNsaWVudF9pZCI6ImxvZ2luIiwiY2lkIjoibG9naW4iLCJncmFudF90eXBlIjoiYXV0aG9yaXphdGlvbl9jb2RlIiwidXNlcl9uYW1lIjoid3JocE41QHRlc3Qub3JnIiwicmV2X3NpZyI6ImI3MjE5ZGYxIiwidXNlcl9pZCI6IjA3NjNlMzYxLTY4NTAtNDc3Yi1iOTU3LWIyYTFmNTcyNzMxNCIsImF1dGhfdGltZSI6MTU1Nzc4NzE4NX0.Fo8wZ_Zq9mwFks3LfXQ1PfJ4ugppjWvioZM6jSqAAQQ",
			"refresh_token": "f59dcb5dcbca45f981f16ce519d61486-r",
			"expires_in":    43199,
			"scope":         "openid oauth.approvals",
			"jti":           "acb6803a48114d9fb4761e403c17f812",
		}),
	)
}

func setupFailedUaaOauthLoginHttpmock() {
	httpmock.RegisterResponder(
		"POST",
		fmt.Sprintf("%s/oauth/token", uaaApiUrl),
		httpmock.NewJsonResponderOrPanic(401, map[string]interface{}{
			"access_token": "fake-access-token-despite-error-status-code",
		}),
	)
}

func authorizationHeader(user, password string) string {
	base := user + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(base))
}
