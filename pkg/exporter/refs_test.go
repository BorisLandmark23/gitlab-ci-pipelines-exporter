// +build !race

package exporter

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/mvisonneau/gitlab-ci-pipelines-exporter/pkg/schemas"
	"github.com/stretchr/testify/assert"
	goGitlab "github.com/xanzy/go-gitlab"
)

func TestGetProjectRefs(t *testing.T) {
	resetGlobalValues()

	mux, server := configureMockedGitlabClient()
	defer server.Close()

	mux.HandleFunc(fmt.Sprintf("/api/v4/projects/1/repository/branches"),
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[{"name":"keep/dev"},{"name":"keep/main"}]`)
		})

	mux.HandleFunc(fmt.Sprintf("/api/v4/projects/1/repository/tags"),
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[{"name":"keep/dev"},{"name":"keep/0.0.2"}]`)
		})

	mux.HandleFunc(fmt.Sprintf("/api/v4/projects/1/pipelines"),
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[{"id":1,"ref":"refs/merge-requests/foo"}]`)
		})

	foundRefs, err := getProjectRefs(1, "^keep", true, 10)
	assert.NoError(t, err)

	expectedRefs := map[string]schemas.ProjectRefKind{
		"keep/0.0.2":              "tag",
		"keep/dev":                "branch",
		"keep/main":               "branch",
		"refs/merge-requests/foo": "merge-request",
	}
	assert.Equal(t, expectedRefs, foundRefs)
}

func TestPullProjectRefsFromProject(t *testing.T) {
	resetGlobalValues()

	mux, server := configureMockedGitlabClient()
	defer server.Close()
	configureStore()
	configurePullingQueue()

	mux.HandleFunc("/api/v4/projects/foo/bar",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"id":1}`)
		})

	mux.HandleFunc(fmt.Sprintf("/api/v4/projects/1/repository/branches"),
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[{"name":"main"},{"name":"nope"}]`)
		})

	mux.HandleFunc(fmt.Sprintf("/api/v4/projects/1/repository/tags"),
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[]`)
		})

	assert.NoError(t, pullProjectRefsFromProject(schemas.Project{Name: "foo/bar"}))

	projectsRefs, _ := store.ProjectsRefs()
	expectedProjectsRefs := schemas.ProjectsRefs{
		"3207122276": schemas.ProjectRef{
			Project: schemas.Project{
				Name: "foo/bar",
			},
			Kind: schemas.ProjectRefKindBranch,
			ID:   1,
			Ref:  "main",
			Jobs: make(map[string]goGitlab.Job),
		},
	}
	assert.Equal(t, expectedProjectsRefs, projectsRefs)
}

func TestPullProjectRefsFromPipelines(t *testing.T) {
	resetGlobalValues()

	mux, server := configureMockedGitlabClient()
	defer server.Close()
	configureStore()
	configurePullingQueue()

	mux.HandleFunc("/api/v4/projects/foo/bar",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"id":1}`)
		})

	mux.HandleFunc(fmt.Sprintf("/api/v4/projects/1/pipelines"),
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[{"id":1,"ref":"main"}]`)
		})

	assert.NoError(t, pullProjectRefsFromPipelines(schemas.Project{Name: "foo/bar"}))

	projectsRefs, _ := store.ProjectsRefs()
	expectedProjectsRefs := schemas.ProjectsRefs{
		"3207122276": schemas.ProjectRef{
			Project: schemas.Project{
				Name: "foo/bar",
			},
			Kind: schemas.ProjectRefKindBranch,
			ID:   1,
			Ref:  "main",
			Jobs: make(map[string]goGitlab.Job),
		},
	}
	assert.Equal(t, expectedProjectsRefs, projectsRefs)
}
