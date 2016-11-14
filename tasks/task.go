package tasks

import (
	"encoding/json"
	"fmt"
	"github.com/CiscoCloud/marathon-consul/utils"
	"github.com/hashicorp/consul/api"
)

type Task struct {
	Timestamp          string                  `json:"timestamp"`
	SlaveID            string                  `json:"slaveId"`
	ID                 string                  `json:"id"`
	TaskStatus         string                  `json:"taskStatus"`
	AppID              string                  `json:"appId"`
	Host               string                  `json:"host"`
	Ports              []int                   `json:"ports"`
	Version            string                  `json:"version"`
	HealthCheckResults []TaskHealthCheckResult `json:"healthCheckResults"`
}

type TaskHealthCheckResult struct {
	Alive bool `json:"alive"`
}

func ParseTask(event []byte) (*Task, error) {
	task := &Task{}
	err := json.Unmarshal(event, task)
	return task, err
}

func (task *Task) Key() string {
	return fmt.Sprintf(
		"%s/tasks/%s",
		utils.CleanID(task.AppID),
		task.ID,
	)
}

func (task *Task) KV() *api.KVPair {
	serialized, _ := json.Marshal(task)

	return &api.KVPair{
		Key:   task.Key(),
		Value: serialized,
	}
}

// Include a derived 'healthy' field in the json output to summarize the
// health check results, making it easier to act on in a template
func (task *Task) MarshalJSON() ([]byte, error) {
	type Alias Task
	return json.Marshal(&struct {
		ReportsHealth bool `json:"reportsHealth"`
		Healthy       bool `json:"healthy"`
		*Alias
	}{
		ReportsHealth: true,
		Healthy:       task.IsHealthy(),
		Alias:         (*Alias)(task),
	})
}

// return true if any health check says the task is alive
func (task *Task) IsHealthy() bool {
	for _, r := range task.HealthCheckResults {
		if r.Alive {
			return true
		}
	}

	return false
}
