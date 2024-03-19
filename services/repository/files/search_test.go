package files

import (
	"testing"

	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/services/contexttest"

	"github.com/stretchr/testify/assert"
)

func TestNewRepoGrep(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, _ := contexttest.MockContext(t, "user2/repo1")
	ctx.SetParams(":id", "1")
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadRepoCommit(t, ctx)
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadGitRepo(t, ctx)
	defer ctx.Repo.GitRepo.Close()

	t.Run("with result", func(t *testing.T) {
		res, err := NewRepoGrep(ctx, ctx.Repo.Repository, "Description")
		assert.NoError(t, err)

		expected := []*Result{
			{
				RepoID:      0,
				Filename:    "README.md",
				CommitID:    "master",
				UpdatedUnix: 0,
				Language:    "Markdown",
				Color:       "#083fa1",
				Lines: []ResultLine{
					{Num: 2, FormattedContent: ""},
					{Num: 3, FormattedContent: "Description for repo1"},
				},
			},
		}

		assert.EqualValues(t, res, expected)
	})

	t.Run("empty result", func(t *testing.T) {
		res, err := NewRepoGrep(ctx, ctx.Repo.Repository, "keyword that does not match in the repo")
		assert.NoError(t, err)

		assert.EqualValues(t, res, []*Result{})
	})
}
