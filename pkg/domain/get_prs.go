package domain

import (
	"fmt"
	"sort"
	"strings"

	"github.com/plumming/chilly/pkg/config"

	"github.com/plumming/chilly/pkg/pr"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/plumming/chilly/pkg/cmd"
)

// GetPrs defines get pull request response.
type GetPrs struct {
	cmd.CommonOptions
	ShowDependabot bool
	ShowOnHold     bool
	PullRequests   []pr.PullRequest
}

// Data.
type Data struct {
	Search Search `json:"search"`
}

// Search.
type Search struct {
	PullRequests []pr.PullRequest `json:"nodes"`
}

// NewGetPrs.
func NewGetPrs() *GetPrs {
	g := &GetPrs{}
	return g
}

// Validate input.
func (g *GetPrs) Validate() error {
	return nil
}

// Run the cmd.
func (g *GetPrs) Run() error {
	client, err := g.GithubClient()
	if err != nil {
		return err
	}

	query := `{
	search(query: "is:pr is:open %s", type: ISSUE, first: 100) {
      nodes {
      ... on PullRequest {
        number
        title
        url
        createdAt
        closed
        author {
          login
        }
        repository {
          nameWithOwner
        }
        mergeable
        labels(first: 10) {
          nodes {
            name
          }
        }
        commits(last: 1){
          nodes{
            commit{
              statusCheckRollup {
                state
                contexts(last:100) {
                  totalCount
                  nodes {
                    ...on StatusContext {
                      state
                      context
                      description
                    }
                    ...on CheckRun {
                      conclusion
                      name
                      title
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}`

	data := Data{}
	repos, hiddenLabels, err := reposAndHiddenLabels()
	if err != nil {
		return err
	}
	queryToRun := fmt.Sprintf(query, strings.Join(repos, " "))
	log.Logger().Debugf("running query %s", queryToRun)

	err = client.GraphQL(queryToRun, nil, &data)
	if err != nil {
		return err
	}

	pulls := data.Search.PullRequests
	sort.Sort(pr.ByPullsString(pulls))

	pullsToReturn := []pr.PullRequest{}

	for _, pr := range pulls {
		pullRequest := pr
		if pr.Display(g.ShowDependabot, g.ShowOnHold, hiddenLabels...) {
			pullsToReturn = append(pullsToReturn, pullRequest)
		}
	}

	g.PullRequests = pullsToReturn

	return nil
}

func reposAndHiddenLabels() ([]string, []string, error) {
	var repos []string
	var hiddenLabels []string

	config, err := config.LoadFromDefaultLocation()
	if err != nil {
		return nil, nil, err
	}
	repos = config.Repos
	hiddenLabels = config.HiddenLabels

	var repoList []string
	for _, r := range repos {
		repoList = append(repoList, fmt.Sprintf("repo:%s", r))
	}
	return repoList, hiddenLabels, nil
}

// Retrigger failed prs.
func (g *GetPrs) Retrigger() error {
	client, err := g.GithubClient()
	if err != nil {
		return err
	}

	log.Logger().Infof("Retriggering Failed & Non Conflicting PRs...")

	for _, pr := range g.PullRequests {
		if pr.ContextsString() == "FAILURE" && pr.Mergeable == "MERGEABLE" && pr.HasLabel("updatebot") {
			failedContexts := pr.FailedContexts()
			for _, f := range failedContexts {
				testCommand := fmt.Sprintf("/test %s", f.Context)
				if f.Context == "pr-build" {
					testCommand = "/test this"
				}
				log.Logger().Infof("%s with '%s'", pr.URL, testCommand)

				url := fmt.Sprintf("repos/%s/issues/%d/comments", pr.Repository.NameWithOwner, pr.Number)
				body := fmt.Sprintf("{ \"body\": \"%s\" }", testCommand)

				err := client.REST("POST", url, strings.NewReader(body), nil)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
