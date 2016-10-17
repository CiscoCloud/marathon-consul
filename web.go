package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/CiscoCloud/marathon-consul/consul"
	"github.com/CiscoCloud/marathon-consul/events"
	"github.com/CiscoCloud/marathon-consul/health"
	"github.com/CiscoCloud/marathon-consul/tasks"
	log "github.com/Sirupsen/logrus"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

type ForwardHandler struct {
	consul consul.Consul
}

func (fh *ForwardHandler) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		w.WriteHeader(500)
		fmt.Fprintln(w, "could not read request body")
		return
	}

	eventType, err := events.EventType(body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		return
	}

	switch eventType {
	case "api_post_event", "deployment_info":
		log.WithField("eventType", eventType).Info("handling event")
		err = fh.HandleAppEvent(body)
	case "app_terminated_event":
		log.WithField("eventType", "app_terminated_event").Info("handling event")
		err = fh.HandleTerminationEvent(body)
	case "status_update_event":
		log.WithField("eventType", "status_update_event").Info("handling event")
		err = fh.HandleStatusEvent(body)
	case "health_status_changed_event":
		log.WithField("eventType", "health_status_changed_event").Info("handling event")
		err = fh.HandleHealthStatusEvent(body)
	default:
		log.WithField("eventType", eventType).Info("not handling event")
		w.WriteHeader(200)
		fmt.Fprintf(w, "cannot handle %s\n", eventType)
		return
	}

	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.WithError(err).Error("body generated error")
	} else {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	}
	log.Debug(string(body))
}

func (fh *ForwardHandler) HandleAppEvent(body []byte) error {
	event, err := events.ParseEvent(body)
	if err != nil {
		return err
	}

	for _, app := range event.Apps() {
		err = fh.consul.UpdateApp(app)
		if err != nil {
			return err
		}
	}
	return nil
}

func (fh *ForwardHandler) HandleTerminationEvent(body []byte) error {
	event, err := events.ParseEvent(body)
	if err != nil {
		return err
	}

	// app_terminated_event only has one app in it, so we will just take care of
	// it instead of looping
	return fh.consul.DeleteApp(event.Apps()[0])
}

func (fh *ForwardHandler) HandleHealthStatusEvent(body []byte) error {
	health, err := health.ParseHealth(body)

	if err != nil {
		return err
	}

	err = fh.consul.UpdateHealth(health)

	return err
}

func (fh *ForwardHandler) HandleStatusEvent(body []byte) error {
	// for every other use of Tasks, Marathon uses the "id" field for the task ID.
	// Here, it uses "taskId", with most of the other fields being equal. We'll
	// just swap "taskId" for "id" in the body so that we can successfully parse
	// incoming events.
	body = bytes.Replace(body, []byte("taskId"), []byte("id"), -1)

	task, err := tasks.ParseTask(body)

	if err != nil {
		return err
	}

	switch task.TaskStatus {
	case "TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST":
		err = fh.consul.DeleteTask(task)
	case "TASK_STAGING", "TASK_STARTING", "TASK_RUNNING":
		err = fh.consul.UpdateTask(task)
	default:
		err = errors.New("unknown task status")
	}
	return err
}
