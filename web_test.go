package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CiscoCloud/marathon-consul/apps"
	"github.com/CiscoCloud/marathon-consul/consul"
	"github.com/CiscoCloud/marathon-consul/events"
	"github.com/CiscoCloud/marathon-consul/health"
	"github.com/CiscoCloud/marathon-consul/mocks"
	"github.com/CiscoCloud/marathon-consul/tasks"
	"github.com/stretchr/testify/assert"
)

var (
	testTask = &tasks.Task{
		Timestamp:  "2014-03-01T23:29:30.158Z",
		SlaveID:    "20140909-054127-177048842-5050-1494-0",
		ID:         "my-app_0-1396592784349",
		TaskStatus: "TASK_RUNNING",
		AppID:      "/my-app",
		Host:       "slave-1234.acme.org",
		Ports:      []int{31372},
		Version:    "2014-04-04T06:26:23.051Z",
	}
	testApp = &apps.App{ID: "test"}

	testTaskKV = testTask.KV()
	testAppKV  = testApp.KV()

	testHealth = &health.Health{
		AppID:     testTask.AppID,
		TaskID:    testTask.ID,
		Timestamp: testTask.Timestamp,
		Alive:     true,
	}
)

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest("GET", "http://example.com/health", nil)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()
	HealthHandler(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
}

func TestForwardHandlerHandleAppEvent(t *testing.T) {
	t.Parallel()

	// create a handler
	kv := mocks.NewKVer()
	consul := consul.NewConsul(kv, "")
	handler := ForwardHandler{consul}

	body, err := json.Marshal(events.APIPostEvent{"api_post_event", testApp})
	assert.Nil(t, err)

	// test!
	err = handler.HandleAppEvent(body)
	assert.Nil(t, err)

	result, _, err := kv.Get(testApp.Key())
	assert.Nil(t, err)
	assert.Equal(t, result, testAppKV)
}

func TestForwardHandlerHandleTerminationEvent(t *testing.T) {
	t.Parallel()

	// create a handler
	kv := mocks.NewKVer()
	consul := consul.NewConsul(kv, "")
	handler := ForwardHandler{consul}

	err := consul.UpdateApp(testApp)
	assert.Nil(t, err)

	body, err := json.Marshal(events.AppTerminatedEvent{
		Type:  "app_terminated_event",
		AppID: testApp.ID,
	})
	assert.Nil(t, err)

	// test!
	err = handler.HandleTerminationEvent(body)
	assert.Nil(t, err)

	result, _, err := kv.Get(testApp.Key())
	assert.Nil(t, err)
	assert.Nil(t, result)
}

func tempTaskBody(status string) []byte {
	body, _ := json.Marshal(testTask)
	return []byte(strings.Replace(
		string(body),
		testTask.TaskStatus,
		status,
		1,
	))
}

func TestForwardHandlerHandleStatusEvent(t *testing.T) {
	t.Parallel()

	// create a handler
	kv := mocks.NewKVer()
	consul := consul.NewConsul(kv, "")
	handler := ForwardHandler{consul}

	// deletes
	for _, status := range []string{"TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST"} {
		tempBody := tempTaskBody(status)

		err := consul.UpdateTask(testTask)
		assert.Nil(t, err)

		// test
		err = handler.HandleStatusEvent(tempBody)
		assert.Nil(t, err)

		// assert
		result, _, err := kv.Get(testTask.Key())
		assert.Nil(t, err)
		assert.Nil(t, result)
	}

	// puts
	for _, status := range []string{"TASK_STAGING", "TASK_STARTING", "TASK_RUNNING"} {
		tempBody := tempTaskBody(status)
		tempTask, _ := tasks.ParseTask(tempBody)

		// test
		err := handler.HandleStatusEvent(tempBody)
		assert.Nil(t, err)

		// assert
		result, _, err := kv.Get(tempTask.Key())
		assert.Nil(t, err)
		assert.Equal(t, result, tempTask.KV())

		// cleanup
		_, err = kv.Delete(testTask.Key())
		assert.Nil(t, err)
	}

	// bad status
	tempBody := tempTaskBody("TASK_BATMAN")
	err := handler.HandleStatusEvent(tempBody)
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "unknown task status")
}

func TestForwardHandlerHandleHealthStatusEvent(t *testing.T) {
	t.Parallel()

	kv := mocks.NewKVer()
	consul := consul.NewConsul(kv, "")
	handler := ForwardHandler{consul}

	testEvent := events.HealthStatusChangeEvent{"health_status_changed_event",
		testHealth.AppID, testHealth.TaskID, testHealth.Timestamp, testHealth.Alive}

	body, err := json.Marshal(testEvent)
	assert.Nil(t, err)

	// first check that we get an error if expected task isn't in the KV
	err = handler.HandleHealthStatusEvent(body)
	assert.NotNil(t, err)

	// populate task that gets updated by health
	err = consul.UpdateTask(testTask)
	err = handler.HandleHealthStatusEvent(body)
	assert.Nil(t, err)

	// check task is updated with alive=true
	result, _, err := kv.Get(testHealth.TaskKey())
	assert.Nil(t, err)
	assert.Contains(t, string(result.Value), "\"alive\":true")

	// test again with alive=false
	testEvent.Alive = false
	body, err = json.Marshal(testEvent)
	err = handler.HandleHealthStatusEvent(body)
	assert.Nil(t, err)

	result, _, err = kv.Get(testHealth.TaskKey())
	assert.Nil(t, err)
	assert.Contains(t, string(result.Value), "\"alive\":false")
}
