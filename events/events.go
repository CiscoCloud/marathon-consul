package events

import (
	"github.com/CiscoCloud/marathon-consul/apps"
)

type Event interface {
	Apps() []*apps.App
	GetType() string
}

type BaseEvent struct {
	Type string `json:"eventType"`
}

type APIPostEvent struct {
	Type string    `json:"eventType"`
	App  *apps.App `json:"appDefinition"`
}

func (event APIPostEvent) Apps() []*apps.App {
	return []*apps.App{event.App}
}

func (event APIPostEvent) GetType() string {
	return event.Type
}

type DeploymentInfoEvent struct {
	Type string `json:"eventType"`
	Plan struct {
		Target struct {
			Apps []*apps.App `json:"apps"`
		} `json:"target"`
	} `json:"plan"`
	CurrentStep struct {
		Action string `json:"action"`
		App    string `json:"app"`
	} `json:"currentStep"`
}

func (event DeploymentInfoEvent) Apps() []*apps.App {
	return event.Plan.Target.Apps
}

func (event DeploymentInfoEvent) GetType() string {
	return event.Type
}

type AppTerminatedEvent struct {
	Type      string `json:"eventType"`
	AppID     string `json:"appId"`
	Timestamp string `json:"timestamp"`
}

func (event AppTerminatedEvent) Apps() []*apps.App {
	return []*apps.App{
		&apps.App{ID: event.AppID},
	}
}

func (event AppTerminatedEvent) GetType() string {
	return event.Type
}

type HealthStatusChangeEvent struct {
	Type      string `json:"eventType"`
	AppID     string `json:"appId"`
	TaskID    string `json:"taskId"`
	Timestamp string `json:"timestamp"`
	Alive     bool   `json:"alive"`
}

func (event HealthStatusChangeEvent) Apps() []*apps.App {
	return []*apps.App{
		&apps.App{ID: event.AppID},
	}
}

func (event HealthStatusChangeEvent) GetType() string {
	return event.Type
}
