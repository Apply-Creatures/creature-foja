package files

import (
	"context"
	"html/template"
	"strconv"
	"strings"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/gitrepo"
	"code.gitea.io/gitea/modules/highlight"
	"code.gitea.io/gitea/modules/timeutil"

	"github.com/go-enry/go-enry/v2"
)

type Result struct {
	RepoID         int64 // ignored
	Filename       string
	CommitID       string             // branch
	UpdatedUnix    timeutil.TimeStamp // ignored
	Language       string
	Color          string
	LineNumbers    []int64
	FormattedLines template.HTML
}

const pHEAD = "HEAD:"

func NewRepoGrep(ctx context.Context, repo *repo_model.Repository, keyword string) ([]*Result, error) {
	t, _, err := gitrepo.RepositoryFromContextOrOpen(ctx, repo)
	if err != nil {
		return nil, err
	}

	data := []*Result{}

	stdout, _, err := git.NewCommand(ctx,
		"grep",
		"-1", // n before and after lines
		"-z",
		"--heading",
		"--break",         // easier parsing
		"--fixed-strings", // disallow regex for now
		"-n",              // line nums
		"-i",              // ignore case
		"--full-name",     // full file path, rel to repo
		//"--column",        // for adding better highlighting support
	).
		AddDynamicArguments(keyword).
		AddArguments("HEAD").
		RunStdString(&git.RunOpts{Dir: t.Path})
	if err != nil {
		return data, nil // non zero exit code when there are no results
	}

	for _, block := range strings.Split(stdout, "\n\n") {
		res := Result{CommitID: repo.DefaultBranch}
		code := []string{}

		for _, line := range strings.Split(block, "\n") {
			if strings.HasPrefix(line, pHEAD) {
				res.Filename = strings.TrimPrefix(line, pHEAD)
				continue
			}

			if ln, after, ok := strings.Cut(line, "\x00"); ok {
				i, err := strconv.ParseInt(ln, 10, 64)
				if err != nil {
					continue
				}

				res.LineNumbers = append(res.LineNumbers, i)
				code = append(code, after)
			}
		}

		if res.Filename == "" || len(code) == 0 || len(res.LineNumbers) == 0 {
			continue
		}

		res.FormattedLines, res.Language = highlight.Code(res.Filename, "", strings.Join(code, "\n"))
		res.Color = enry.GetColor(res.Language)

		data = append(data, &res)
	}

	return data, nil
}
