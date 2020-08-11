package hosts

import (
	"context"
	"strings"
	"sync"

	"github.com/zricethezav/gitleaks/v5/manager"
	"github.com/zricethezav/gitleaks/v5/options"
	"github.com/zricethezav/gitleaks/v5/scan"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	log "github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
)

// Gitlab wraps a gitlab client and manager. This struct implements what the Host interface defines.
type Gitlab struct {
	client  *gitlab.Client
	manager *manager.Manager
	ctx     context.Context
	wg      sync.WaitGroup
}

// NewGitlabClient accepts a manager struct and returns a Gitlab host pointer which will be used to
// perform a gitlab scan on an group or user.
func NewGitlabClient(m *manager.Manager) (*Gitlab, error) {
	var err error

	gitlabClient := &Gitlab{
		manager: m,
		ctx:     context.Background(),
		client:  gitlab.NewClient(nil, options.GetAccessToken(m.Opts)),
	}

	if m.Opts.BaseURL != "" {
		err = gitlabClient.client.SetBaseURL(m.Opts.BaseURL)
	}

	return gitlabClient, err
}

// Scan will scan a github user or organization's repos.
func (g *Gitlab) Scan() {
	var (
		projects []*gitlab.Project
		resp     *gitlab.Response
		err      error
	)

	page := 1
	listOpts := gitlab.ListOptions{
		PerPage: 100,
		Page:    page,
	}
	for {
		var _projects []*gitlab.Project
		if g.manager.Opts.User != "" {
			glOpts := &gitlab.ListProjectsOptions{
				ListOptions: listOpts,
			}
			_projects, resp, err = g.client.Projects.ListUserProjects(g.manager.Opts.User, glOpts)

		} else if g.manager.Opts.Organization != "" {
			glOpts := &gitlab.ListGroupProjectsOptions{
				ListOptions: listOpts,
			}
			_projects, resp, err = g.client.Groups.ListGroupProjects(g.manager.Opts.Organization, glOpts)
		}
		if err != nil {
			log.Error(err)
		}

		for _, p := range _projects {
			if g.manager.Opts.ExcludeForks && p.ForkedFromProject != nil {
				log.Debugf("excluding forked repo: %s", p.Name)
				continue
			}
			projects = append(projects, p)
		}

		if resp == nil {
			break
		}
		if page >= resp.TotalPages {
			// exit when we've seen all pages
			break
		}
		page = resp.NextPage
	}

	// iterate of gitlab projects
	for _, p := range projects {
		r := scan.NewRepo(g.manager)
		cloneOpts := g.manager.CloneOptions
		cloneOpts.URL = p.HTTPURLToRepo
		err := r.Clone(cloneOpts)
		if err != nil {
			log.Error(err)
			continue
		}
		// TODO handle clone retry with ssh like github host
		r.Name = p.Name

		if err = r.Scan(); err != nil {
			log.Error(err)
		}
	}
}

// ScanPR TODO not implemented
func (g *Gitlab) ScanPR() {
	log.Error("ScanPR is not implemented in Gitlab host yet...")
}

// ScanCommitURL scan a single gitlab commit link url
func (g *Gitlab) ScanCommitURL() {

	url := strings.Replace(g.manager.Opts.CommitURL, "/-/", "/", 1)
	splitPath := strings.SplitN(url, "/", 4)
	splits := strings.Split(splitPath[3], "/commit/")

	repoName := splits[0]
	commitSha := splits[1]
	repo := scan.NewRepo(g.manager)
	repo.Name = repoName
	log.Infof("scanning commit link %s\n", g.manager.Opts.CommitURL)

	commit, _, err := g.client.Commits.GetCommit(repoName, commitSha)
	if err != nil {
		log.Fatalf("Failed to retrieve commit %v", err)
	}

	commitObj := object.Commit{
		Hash: plumbing.NewHash(commit.ID),
		Author: object.Signature{
			Name:  commit.AuthorName,
			Email: commit.AuthorEmail,
			When:  *commit.AuthoredDate,
		},
	}

	// Loop through all diffs in the commit
	diffPage := 1
	for {
		diffs, diffResp, diffErr := g.client.Commits.GetCommitDiff(repoName, commitSha, &gitlab.GetCommitDiffOptions{
			PerPage: 100, Page: diffPage})
		if diffErr != nil {
			log.Errorf("Failed to retrieve commit %v", diffErr)
			continue
		}

		// Loop through each diff
		for _, d := range diffs {
			repo.CheckRules(&scan.Bundle{
				Content:  d.Diff,
				FilePath: d.NewPath,
				Commit:   &commitObj,
			})
		}

		if diffResp.CurrentPage >= diffResp.TotalPages {
			break
		}
		diffPage = diffResp.NextPage
	}

}
