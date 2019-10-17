package requesthandlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-shell/internal/gitlabnet/testserver"
)

func BuildDisallowedByApiHandlers(t *testing.T) []testserver.TestRequestHandler {
	requests := []testserver.TestRequestHandler{
		{
			Path: "/api/v4/internal/allowed",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				body := map[string]interface{}{
					"status":  false,
					"message": "Disallowed by API call",
				}
				w.WriteHeader(http.StatusForbidden)
				require.NoError(t, json.NewEncoder(w).Encode(body))
			},
		},
	}

	return requests
}

func BuildAllowedWithGitalyHandlers(t *testing.T, gitalyAddress string) []testserver.TestRequestHandler {
	requests := []testserver.TestRequestHandler{
		{
			Path: "/api/v4/internal/allowed",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				body := map[string]interface{}{
					"status": true,
					"gl_id":  "1",
					"gitaly": map[string]interface{}{
						"repository": map[string]interface{}{
							"storage_name":                     "storage_name",
							"relative_path":                    "relative_path",
							"git_object_directory":             "path/to/git_object_directory",
							"git_alternate_object_directories": []string{"path/to/git_alternate_object_directory"},
							"gl_repository":                    "group/repo",
							"gl_project_path":                  "group/project-path",
						},
						"address": gitalyAddress,
						"token":   "token",
					},
				}
				require.NoError(t, json.NewEncoder(w).Encode(body))
			},
		},
	}

	return requests
}
