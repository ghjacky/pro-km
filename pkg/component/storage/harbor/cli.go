package harbor

import (
	"fmt"
)

// Cli harbor client
type Cli struct {
	*Client
	BaseURL  string
	UserName string
	Password string
}

// NewHarborBasicClient return a harbor client by url, username, password
func NewHarborBasicClient(harborURL, username, password string) *Cli {
	return &Cli{
		Client:   NewClient(nil, harborURL, username, password),
		BaseURL:  harborURL,
		UserName: username,
		Password: password,
	}
}

// SearchRepositories search list of repositories
func (c *Cli) SearchRepositories(query string) ([]SearchRepository, error) {
	if query == "" {
		// when query is empty return empty result
		return []SearchRepository{}, nil
	}
	result, resp, errs := c.Search(&SearchOption{
		ListOptions: ListOptions{
			Page:     1,
			PageSize: 10,
		},
		Q: query,
	})
	if len(errs) > 0 {
		return nil, fmt.Errorf("search repositories failed: %v, response: %v", errs, *resp)
	}
	return result.Repositories, nil
}

// ListRepositoryTags return all tags of one repository
func (c *Cli) ListRepositoryTags(repoName string) ([]TagResp, error) {
	tags, resp, errs := c.Repositories.ListRepositoryTags(repoName)
	if len(errs) > 0 {
		return nil, fmt.Errorf("list tags failed: %v, reponse: %v", errs, *resp)
	}
	return tags, nil
}
