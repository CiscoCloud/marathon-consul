package health

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testHealth = &Health{
	AppID:     "/my-app",
	TaskID:    "my-app_0-1396592784349",
	Timestamp: "2014-03-01T23:29:30.158Z",
	Alive:     true,
}

func TestParseHealth(t *testing.T) {
	t.Parallel()

	jsonified, err := json.Marshal(testHealth)
	assert.Nil(t, err)

	health, err := ParseHealth(jsonified)
	assert.Nil(t, err)

	assert.Equal(t, testHealth.AppID, health.AppID)
	assert.Equal(t, testHealth.TaskID, health.TaskID)
	assert.Equal(t, testHealth.Timestamp, health.Timestamp)
	assert.Equal(t, testHealth.Alive, health.Alive)
}

func TestTaskKey(t *testing.T) {
	t.Parallel()

	tk := testHealth.TaskKey()

	assert.Equal(t, fmt.Sprintf("%s/tasks/%s", "my-app", testHealth.TaskID), tk)
}
