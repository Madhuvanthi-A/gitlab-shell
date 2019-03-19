package twofactorrecover

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-shell/go/internal/command/commandargs"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/command/readwriter"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/config"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/gitlabnet/testserver"
)

var (
	testConfig = &config.Config{GitlabUrl: "http+unix://" + testserver.TestSocket}
	requests   = []testserver.TestRequestHandler{
		{
			Path: "/api/v4/internal/two_factor_recovery_codes",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				b, _ := ioutil.ReadAll(r.Body)
				defer r.Body.Close()

				var bodyRaw map[string]*json.RawMessage
				json.Unmarshal(b, &bodyRaw)

				var keyId string
				json.Unmarshal(*bodyRaw["key_id"], &keyId)

				switch keyId {
				case "1":
					body := map[string]interface{}{
						"success":        true,
						"recovery_codes": [2]string{"recovery", "codes"},
					}
					json.NewEncoder(w).Encode(body)
				case "broken_message":
					body := map[string]interface{}{
						"success": false,
						"message": "Forbidden!",
					}
					json.NewEncoder(w).Encode(body)
				case "broken":
					w.WriteHeader(http.StatusInternalServerError)
				default:
					fmt.Fprint(w, "null")
				}
			},
		},
	}
)

func TestExecute(t *testing.T) {
	cleanup, err := testserver.StartSocketHttpServer(requests)
	require.NoError(t, err)
	defer cleanup()

	testCases := []struct {
		desc           string
		arguments      *commandargs.CommandArgs
		answer         string
		expectedOutput string
	}{
		{
			desc:      "With a known key id",
			arguments: &commandargs.CommandArgs{GitlabKeyId: "1"},
			answer:    "yes\n",
			expectedOutput: "Are you sure you want to generate new two-factor recovery codes?\n" +
				"Any existing recovery codes you saved will be invalidated. (yes/no)\n\n" +
				"Your two-factor authentication recovery codes are:\n\nrecovery\ncodes\n\n" +
				"During sign in, use one of the codes above when prompted for\n" +
				"your two-factor code. Then, visit your Profile Settings and add\n" +
				"a new device so you do not lose access to your account again.\n",
		},
		{
			desc:      "With an unknown key",
			arguments: &commandargs.CommandArgs{GitlabKeyId: "-1"},
			answer:    "yes\n",
			expectedOutput: "Are you sure you want to generate new two-factor recovery codes?\n" +
				"Any existing recovery codes you saved will be invalidated. (yes/no)\n\n" +
				"An error occurred while trying to generate new recovery codes.\n\n",
		},
		{
			desc:      "With API returns an error",
			arguments: &commandargs.CommandArgs{GitlabKeyId: "broken_message"},
			answer:    "yes\n",
			expectedOutput: "Are you sure you want to generate new two-factor recovery codes?\n" +
				"Any existing recovery codes you saved will be invalidated. (yes/no)\n\n" +
				"An error occurred while trying to generate new recovery codes.\n" +
				"Forbidden!\n",
		},
		{
			desc:      "With API fails",
			arguments: &commandargs.CommandArgs{GitlabKeyId: "broken"},
			answer:    "yes\n",
			expectedOutput: "Are you sure you want to generate new two-factor recovery codes?\n" +
				"Any existing recovery codes you saved will be invalidated. (yes/no)\n\n" +
				"An error occurred while trying to generate new recovery codes.\n" +
				"Internal API error (500)\n",
		},
		{
			desc:      "With missing arguments",
			arguments: &commandargs.CommandArgs{},
			answer:    "yes\n",
			expectedOutput: "Are you sure you want to generate new two-factor recovery codes?\n" +
				"Any existing recovery codes you saved will be invalidated. (yes/no)\n\n" +
				"An error occurred while trying to generate new recovery codes.\nFailed to get key id\n",
		},
		{
			desc:      "With negative answer",
			arguments: &commandargs.CommandArgs{},
			answer:    "no\n",
			expectedOutput: "Are you sure you want to generate new two-factor recovery codes?\n" +
				"Any existing recovery codes you saved will be invalidated. (yes/no)\n\n" +
				"New recovery codes have *not* been generated. Existing codes will remain valid.\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output := &bytes.Buffer{}
			input := bytes.NewBufferString(tc.answer)

			cmd := &Command{Config: testConfig, Args: tc.arguments}

			err := cmd.Execute(&readwriter.ReadWriter{Out: output, In: input})

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedOutput, output.String())
		})
	}
}
