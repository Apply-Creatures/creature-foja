// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package actions

import (
	"net/url"

	actions_service "code.gitea.io/gitea/services/actions"
	context_module "code.gitea.io/gitea/services/context"
)

func ManualRunWorkflow(ctx *context_module.Context) {
	workflowID := ctx.FormString("workflow")
	if len(workflowID) == 0 {
		ctx.ServerError("workflow", nil)
		return
	}

	ref := ctx.FormString("ref")
	if len(ref) == 0 {
		ctx.ServerError("ref", nil)
		return
	}

	if empty, err := ctx.Repo.GitRepo.IsEmpty(); err != nil {
		ctx.ServerError("IsEmpty", err)
		return
	} else if empty {
		ctx.NotFound("IsEmpty", nil)
		return
	}

	workflow, err := actions_service.GetWorkflowFromCommit(ctx.Repo.GitRepo, ref, workflowID)
	if err != nil {
		ctx.ServerError("GetWorkflowFromCommit", err)
		return
	}

	location := ctx.Repo.RepoLink + "/actions?workflow=" + url.QueryEscape(workflowID) +
		"&actor=" + url.QueryEscape(ctx.FormString("actor")) +
		"&status=" + url.QueryEscape(ctx.FormString("status"))

	formKeyGetter := func(key string) string {
		formKey := "inputs[" + key + "]"
		return ctx.FormString(formKey)
	}

	if err := workflow.Dispatch(ctx, formKeyGetter, ctx.Repo.Repository, ctx.Doer); err != nil {
		if actions_service.IsInputRequiredErr(err) {
			ctx.Flash.Error(ctx.Locale.Tr("actions.workflow.dispatch.input_required", err.(actions_service.InputRequiredErr).Name))
			ctx.Redirect(location)
			return
		}
		ctx.ServerError("workflow.Dispatch", err)
		return
	}

	// forward to the page of the run which was just created
	ctx.Flash.Info(ctx.Locale.Tr("actions.workflow.dispatch.success"))
	ctx.Redirect(location)
}
