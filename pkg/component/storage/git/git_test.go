package git

import (
	"testing"
)

var gitlabURL = "https://code.xxxxx.cn"

var username = ""

var password = ""

func TestListAllGroups(t *testing.T) {
	g, err := NewGitTokenClient(gitlabURL, "dueJccacVmCaQCHtz9Eh")
	if err != nil {
		t.Fatalf("create client err: %v", err)
	}
	groups, err := g.ListAllGroups()
	if err != nil {
		t.Fatalf("list group err: %v", err)
	}
	for _, group := range groups {
		t.Logf("%v, %v, %v", group.Name, group.FullPath, group.Path)
	}
	t.Logf("Found %d groups", len(groups))

	projs, err := g.SearchProjects("platform/maya", 1, 10, true)
	if err != nil {
		t.Fatalf("list projects err: %v", err)
	}
	for _, proj := range projs {

		branches, err := g.ListAllBranches(proj.PathWithNamespace)
		if err != nil {
			t.Logf("get branch for project %s failed: %v", proj.PathWithNamespace, err)
		}

		t.Logf("%v, %v, %v", proj.Name, proj.PathWithNamespace, proj.WebURL)
		for _, b := range branches {
			t.Logf("branch == %s, %s", b.Name, b.WebURL)
		}

		tags, err := g.ListAllTags(proj.PathWithNamespace)
		if err != nil {
			t.Logf("get tags for project %s failed: %v", proj.PathWithNamespace, err)
		}
		for _, tag := range tags {
			t.Logf("tag == %s", tag.Name)
		}
	}
	t.Logf("Found %d projects", len(projs))

}

func TestClient_GetProject(t *testing.T) {
	g, err := NewGitBasicClient(gitlabURL, username, password)
	if err != nil {
		t.Fatalf("create client err: %v", err)
	}
	project, err := g.GetProject("platform/maya")
	if err != nil {
		t.Fatalf("get project err: %v", err)
	}
	t.Logf("project: %v", project)
}

func TestClient_GetBranch(t *testing.T) {
	g, err := NewGitBasicClient(gitlabURL, username, password)
	if err != nil {
		t.Fatalf("create client err: %v", err)
	}
	project, err := g.GetBranch("platform/maya", "master")
	if err != nil {
		t.Fatalf("get branch err: %v", err)
	}
	t.Logf("branch: %v", project)
}

func TestClient_GetTag(t *testing.T) {
	g, err := NewGitBasicClient(gitlabURL, username, password)
	if err != nil {
		t.Fatalf("create client err: %v", err)
	}
	project, err := g.GetTag("platform/maya", "v1.0.0")
	if err != nil {
		t.Fatalf("get tag err: %v", err)
	}
	t.Logf("tag: %v", project)
}

func TestClient_ListFiles(t *testing.T) {
	g, err := NewGitBasicClient(gitlabURL, username, password)
	if err != nil {
		t.Fatalf("create client err: %v", err)
	}
	nodes, err := g.ListAllFiles("platform/maya", "pkg/agent/", "master", "", false)
	if err != nil {
		t.Fatalf("list files err: %v", err)
	}
	for _, node := range nodes {
		t.Logf("%v", node)
	}
}
