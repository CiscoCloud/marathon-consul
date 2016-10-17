package health

import (
	"encoding/json"
	"fmt"
	"github.com/CiscoCloud/marathon-consul/utils"
)

type Health struct {
	AppID     string `json:"appId"`
	TaskID    string `json:"taskId"`
	Timestamp string `json:"timestamp"`
	Alive     bool   `json:"alive"`
}

func ParseHealth(event []byte) (*Health, error) {
	health := &Health{}
	err := json.Unmarshal(event, health)
	return health, err
}

func (health *Health) TaskKey() string {
	return fmt.Sprintf(
		"%s/tasks/%s",
		utils.CleanID(health.AppID),
		health.TaskID,
	)
}
