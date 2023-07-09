// SPDX-License-Identifier: MIT

package private

import (
	"context"

	"code.gitea.io/gitea/modules/setting"
)

type ActionsRunnerRegisterRequest struct {
	Token   string
	Scope   string
	Labels  []string
	Name    string
	Version string
}

func ActionsRunnerRegister(ctx context.Context, token, scope string, labels []string, name, version string) (string, ResponseExtra) {
	reqURL := setting.LocalURL + "api/internal/actions/register"

	req := newInternalRequest(ctx, reqURL, "POST", ActionsRunnerRegisterRequest{
		Token:   token,
		Scope:   scope,
		Labels:  labels,
		Name:    name,
		Version: version,
	})

	resp, extra := requestJSONResp(req, &ResponseText{})
	return resp.Text, extra
}
