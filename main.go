package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"os"

	"github.com/CiscoCloud/marathon-consul/config"
	"github.com/CiscoCloud/marathon-consul/consul"
	"github.com/CiscoCloud/marathon-consul/events"
	"github.com/CiscoCloud/marathon-consul/marathon"
	log "github.com/Sirupsen/logrus"
)

const Name = "marathon-consul"
const Version = "0.2.0"

func main() {
	config := config.New()
	apiConfig, err := config.Registry.Config()
	if err != nil {
		log.Fatal(err.Error())
	}

	kv, err := consul.NewKV(apiConfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	consul := consul.NewConsul(kv, config.Registry.Prefix)

	// set up initial sync
	remote, err := config.Marathon.NewMarathon()
	if err != nil {
		log.Fatal(err.Error())
	}
	sync := marathon.NewMarathonSync(remote, consul)
	go sync.Sync()

	fh := &ForwardHandler{consul}

	version, err := remote.Version()
	if version > "0.10.0" {
		log.Info(fmt.Sprintf("detected Marathon v%s with /v2/events endpoint", version))
		SubscribeToEventStream(config, remote, fh)
	} else {
		log.Info(fmt.Sprintf("detected Marathon v%s -- make sure to set up an eventSubscription for this process", version))
		ServeWebhookReceiver(config, fh)
	}
}

func SubscribeToEventStream(config *config.Config, m marathon.Marathon, fh *ForwardHandler) {
	buffer := make([]byte, 1024)
	req, err := http.NewRequest("GET", m.Url("/v2/events"), bytes.NewBuffer(buffer))
	if err != nil {
		log.WithError(err).Error("Could not GET /v2/events")
		os.Exit(1)
	}
	req.Header.Set("Accept", "text/event-stream")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("HTTP request for /v2/events failed!")
		os.Exit(1)
	}
	defer resp.Body.Close()
	log.Info("reader here")
	reader := bufio.NewReader(resp.Body)

	for {
		body, err := reader.ReadBytes('\n')
		if err != nil {
			panic(err)
		}

		// marathon sends blank lines to keep the connection alive
		if bytes.Equal(body, []byte{'\r', '\n'}) {
			continue
		}

		// we don't care about these headers, since the data blob has an
		// "eventType" field
		if string(body[0:6]) == "event:" {
			continue
		}

		if string(body[0:5]) == "data:" {
			body = body[6:]
			eventType, err := events.EventType(body)
			if err != nil {
				log.WithError(err).Error("error parsing event")
				continue
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
			default:
				log.WithField("eventType", eventType).Info("not handling event")
			}

			if err != nil {
				log.WithError(err).Error("body generated error")
				continue
			}
		}
	}
}

func ServeWebhookReceiver(config *config.Config, fh *ForwardHandler) {
	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/events", fh.Handle)

	log.WithField("port", config.Web.Listen).Info("listening")
	log.Fatal(http.ListenAndServe(config.Web.Listen, nil))
}
