package goal

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

var sampleGoalsJSON = `{
	"goals": [
		{
			"id": "goal-1",
			"pretty_id": "1",
			"name": "Q1 Revenue",
			"team_id": "12345",
			"creator": 1,
			"owner": null,
			"color": "#ff0000",
			"date_created": "1609459200000",
			"start_date": null,
			"due_date": "1617235200000",
			"description": "Hit revenue target",
			"private": false,
			"archived": false,
			"multiple_owners": false,
			"editor_token": "",
			"date_updated": "1609459200000",
			"last_update": "1609459200000",
			"folder_id": null,
			"pinned": false,
			"owners": [],
			"key_result_count": 3,
			"members": [],
			"group_members": [],
			"percent_completed": 45
		},
		{
			"id": "goal-2",
			"pretty_id": "2",
			"name": "Ship v2",
			"team_id": "12345",
			"creator": 1,
			"owner": null,
			"color": "#00ff00",
			"date_created": "1609459200000",
			"start_date": null,
			"due_date": "1625097600000",
			"description": "Launch version 2",
			"private": false,
			"archived": false,
			"multiple_owners": false,
			"editor_token": "",
			"date_updated": "1609459200000",
			"last_update": "1609459200000",
			"folder_id": null,
			"pinned": false,
			"owners": [],
			"key_result_count": 5,
			"members": [],
			"group_members": [],
			"percent_completed": 80
		}
	],
	"folders": []
}`

func goalsHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(body))
	}
}

func TestGoalList(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/goal", goalsHandler(sampleGoalsJSON))

	cmd := NewCmdGoalList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Q1 Revenue")
	assert.Contains(t, out, "goal-1")
	assert.Contains(t, out, "45%")
	assert.Contains(t, out, "Ship v2")
	assert.Contains(t, out, "goal-2")
	assert.Contains(t, out, "80%")
}

func TestGoalList_JSON(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/goal", goalsHandler(sampleGoalsJSON))

	cmd := NewCmdGoalList(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--json")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed), "output should be valid JSON")
	assert.Len(t, parsed, 2)
	assert.Equal(t, "goal-1", parsed[0]["id"])
}

func TestGoalList_Empty(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/goal", goalsHandler(`{"goals": [], "folders": []}`))

	cmd := NewCmdGoalList(tf.Factory)
	err := testutil.RunCommand(t, cmd)
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "No goals found.")
}

func TestGoalCreate(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/goal", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "New Goal", req["name"])

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Write([]byte(`{"goal": {"id": "goal-new", "name": "New Goal"}}`))
	})

	cmd := NewCmdGoalCreate(tf.Factory)
	err := testutil.RunCommand(t, cmd, "--name", "New Goal")
	require.NoError(t, err)

	out := tf.OutBuf.String()
	assert.Contains(t, out, "Goal")
	assert.Contains(t, out, "New Goal")
	assert.Contains(t, out, "created")
}
