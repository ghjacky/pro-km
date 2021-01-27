package git

import (
	"github.com/xanzy/go-gitlab"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

// Client the client of git repo
type Client struct {
	*gitlab.Client
	Username string
	Password string
}

// NewGitBasicClient return a git client by username, password
func NewGitBasicClient(gitlabURL, username, password string) (*Client, error) {

	client, err := gitlab.NewBasicAuthClient(username, password, gitlab.WithBaseURL(gitlabURL))
	if err != nil {
		return nil, err
	}
	return &Client{
		Username: username,
		Password: password,
		Client:   client,
	}, nil
}

// NewGitTokenClient return a git client by username, password
// Acquire token: Settings -> Access Tokens -> Scopes -> api (in gitlab)
func NewGitTokenClient(gitlabURL, token string) (*Client, error) {
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(gitlabURL))
	if err != nil {
		return nil, err
	}
	return &Client{
		Username: "maya", //  anything without empty, will not participate in authentication
		Password: token,
		Client:   client,
	}, nil
}

// SearchGroups return groups for the authenticated user with pagination.
func (g *Client) SearchGroups(name string, page, size int) ([]*gitlab.Group, error) {

	groups, _, err := g.Groups.ListGroups(&gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: size,
		},
		Search: &name,
	})
	if err != nil {
		return nil, err
	}

	return groups, nil
}

// ListAllGroups return all groups for the authenticated user.
func (g *Client) ListAllGroups() ([]*gitlab.Group, error) {

	var allGroups []*gitlab.Group
	groups, response, err := g.Groups.ListGroups(&gitlab.ListGroupsOptions{})
	if err != nil {
		return nil, err
	}
	allGroups = append(allGroups, groups...)

	totalPage := response.TotalPages
	for i := 2; i < totalPage+1; i++ {
		groups, _, err := g.Groups.ListGroups(&gitlab.ListGroupsOptions{
			ListOptions: gitlab.ListOptions{
				Page: i,
			},
		})
		if err != nil {
			return nil, err
		}
		allGroups = append(allGroups, groups...)
	}
	return allGroups, nil
}

// SearchProjects return projects for the authenticated user with pagination.
// simple means that return only limited fields for each project.
func (g *Client) SearchProjects(name string, page, size int, simple bool) ([]*gitlab.Project, error) {
	searchNamespaces := true
	projects, _, err := g.Projects.ListProjects(&gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: size,
		},
		Simple:           &simple,
		Search:           &name,
		SearchNamespaces: &searchNamespaces,
	})
	if err != nil {
		return nil, err
	}
	return projects, nil
}

// GetProject return the specify project
func (g *Client) GetProject(project string) (*gitlab.Project, error) {
	prj, _, err := g.Projects.GetProject(project, &gitlab.GetProjectOptions{})
	return prj, err
}

// SearchProjectsFromGroup return projects of the group for the authenticated user with pagination.
// simple means that return only limited fields for each project.
func (g *Client) SearchProjectsFromGroup(group, project string, page, size int, simple bool) ([]*gitlab.Project, error) {

	projects, _, err := g.Groups.ListGroupProjects(group, &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: size,
		},
		Search: &project,
		Simple: &simple,
	})

	if err != nil {
		return nil, err
	}
	return projects, nil
}

// ListAllProjects return all projects for the authenticated user.
// membership means that return only limited projects whose member include this user.
// simple means that return only limited fields for each project.
func (g *Client) ListAllProjects(membership, simple bool) ([]*gitlab.Project, error) {

	var allProjects []*gitlab.Project
	projects, response, err := g.Projects.ListProjects(&gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
		Simple:     &simple,
		Membership: &membership,
	})
	if err != nil {
		return nil, err
	}
	allProjects = append(allProjects, projects...)

	totalPage := response.TotalPages
	for i := 2; i < totalPage+1; i++ {
		projects, _, err := g.Projects.ListProjects(&gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    i,
				PerPage: 100,
			},
			Simple:     &simple,
			Membership: &membership,
		})
		if err != nil {
			return nil, err
		}
		allProjects = append(allProjects, projects...)
	}
	return allProjects, nil
}

// ListAllProjectsFromGroup return all projects of the group for the authenticated user.
// simple means that return only limited fields for each project.
func (g *Client) ListAllProjectsFromGroup(group string, simple bool) ([]*gitlab.Project, error) {

	var allProjects []*gitlab.Project

	projects, response, err := g.Groups.ListGroupProjects(group, &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
		Simple: &simple,
	})
	if err != nil {
		return nil, err
	}
	allProjects = append(allProjects, projects...)

	totalPage := response.TotalPages
	for i := 2; i < totalPage+1; i++ {
		projects, _, err := g.Groups.ListGroupProjects(group, &gitlab.ListGroupProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    i,
				PerPage: 100,
			},
			Simple: &simple,
		})
		if err != nil {
			return nil, err
		}
		allProjects = append(allProjects, projects...)
	}
	return allProjects, nil
}

// SearchBranch return matched branches of the project
// project means that project id or url (eg. 2924 or platform/maya)
func (g *Client) SearchBranch(project string, branch string, page, size int) ([]*gitlab.Branch, error) {
	branches, _, err := g.Branches.ListBranches(project, &gitlab.ListBranchesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: size,
		},
		Search: &project,
	})
	if err != nil {
		return nil, err
	}
	return branches, nil
}

// GetBranch return specified branch of a project
func (g *Client) GetBranch(project, branch string) (*gitlab.Branch, error) {
	b, _, err := g.Branches.GetBranch(project, branch)
	return b, err
}

// ListAllBranches return all branches of the project
// project means that project id or url (eg. 2924 or platform/maya)
func (g *Client) ListAllBranches(project string) ([]*gitlab.Branch, error) {
	var allBranches []*gitlab.Branch
	branches, response, err := g.Branches.ListBranches(project, &gitlab.ListBranchesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	})
	if err != nil {
		return nil, err
	}
	allBranches = append(allBranches, branches...)

	totalPage := response.TotalPages
	for i := 2; i < totalPage+1; i++ {
		branches, _, err := g.Branches.ListBranches(project, &gitlab.ListBranchesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    i,
				PerPage: 100,
			},
		})
		if err != nil {
			return nil, err
		}
		allBranches = append(allBranches, branches...)
	}
	return allBranches, nil
}

// SearchTag return matched tags of the project
// project means that project id or url (eg. 2924 or platform/maya)
func (g *Client) SearchTag(project string, tag string, page, size int) ([]*gitlab.Tag, error) {
	tags, _, err := g.Tags.ListTags(project, &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: size,
		},
		Search: &tag,
	})
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// GetTag return specified tag of a project
func (g *Client) GetTag(project, tag string) (*gitlab.Tag, error) {
	t, _, err := g.Tags.GetTag(project, tag)
	return t, err
}

// ListAllTags return all tags of the project
// project means that project id or url (eg. 2924 or platform/maya)
func (g *Client) ListAllTags(project string) ([]*gitlab.Tag, error) {
	var allTags []*gitlab.Tag
	tags, response, err := g.Tags.ListTags(project, &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	})
	if err != nil {
		return nil, err
	}
	allTags = append(allTags, tags...)

	totalPage := response.TotalPages
	for i := 2; i < totalPage+1; i++ {
		tags, _, err := g.Tags.ListTags(project, &gitlab.ListTagsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    i,
				PerPage: 100,
			},
		})
		if err != nil {
			return nil, err
		}
		allTags = append(allTags, tags...)
	}
	return allTags, nil
}

// Clone clone a repo from url.
// Tag has a higher priority than Branch.
// Depth Limit fetching to the specified number of commits, 0 will no limit.
func (g *Client) Clone(workDir, gitURL, branch, tag string, depth int) (*git.Repository, error) {

	cloneOpts := &git.CloneOptions{
		URL: gitURL,
		Auth: &githttp.BasicAuth{
			Username: g.Username,
			Password: g.Password,
		},
	}

	// Tag has a higher priority than Branch
	if branch != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(branch)
	}
	if tag != "" {
		cloneOpts.ReferenceName = plumbing.NewTagReferenceName(tag)
	}

	if depth != 0 {
		cloneOpts.Depth = depth
	}

	repo, err := git.PlainClone(workDir, false, cloneOpts)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// ListAllFiles return all files in repo
func (g *Client) ListAllFiles(project string, path string, branch string, tag string, recursive bool) ([]*gitlab.TreeNode, error) {
	listTreeOption := &gitlab.ListTreeOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
		Path:      &path,
		Recursive: &recursive,
	}
	if branch != "" {
		listTreeOption.Ref = &branch
	}
	if tag != "" {
		listTreeOption.Ref = &tag
	}

	treeNodes, response, err := g.Repositories.ListTree(project, listTreeOption)
	if err != nil {
		return nil, err
	}

	totalPage := response.TotalPages
	for i := 2; i < totalPage+1; i++ {
		listTreeOption.ListOptions.Page = i
		files, _, err := g.Repositories.ListTree(project, listTreeOption)
		if err != nil {
			return nil, err
		}
		treeNodes = append(treeNodes, files...)
	}

	return treeNodes, nil
}
