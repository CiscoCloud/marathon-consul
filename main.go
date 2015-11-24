package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/CiscoCloud/marathon-consul/config"
	"github.com/CiscoCloud/marathon-consul/consul"
	"github.com/CiscoCloud/marathon-consul/events"
	"github.com/CiscoCloud/marathon-consul/marathon"
	log "github.com/Sirupsen/logrus"
	version "github.com/hashicorp/go-version"
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

	v, err := remote.Version()
	if err != nil {
		log.WithError(err).Warn("version parsing failed, assuming >= 0.9.0")
		v, _ = version.NewVersion("0.9.0")
	}
	minVersion, _ := version.NewConstraint(">= 0.9.0")

	if minVersion.Check(v) {
		log.Info(fmt.Sprintf("detected Marathon v%s with /v2/events endpoint", v))
		SubscribeToEventStream(config, remote, fh)
	} else {
		log.Info(fmt.Sprintf("detected Marathon v%s -- make sure to set up an eventSubscription for this process", v))
		ServeWebhookReceiver(config, fh)
	}
}

func SubscribeToEventStream(config *config.Config, m marathon.Marathon, fh *ForwardHandler) {
Reconnect:
	for {
		resp, err := makeEventStreamRequest(m.Url("/v2/events"))
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		log.Info("connected to /v2/events endpoint")

		if err != nil {
			log.WithError(err).Error("error connecting to event stream!")
			time.Sleep(10 * time.Second)
			log.Info("reconnecting...")
			continue Reconnect
		}

		for {
			body, err := reader.ReadBytes('\n')

			if err != nil {
				log.WithError(err).Error("error reading from event stream!")
				time.Sleep(10 * time.Second)
				log.Info("reconnecting...")
				continue Reconnect
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

				eventLogger := log.WithField("eventType", eventType)
				switch eventType {
				case "api_post_event", "deployment_info":
					eventLogger.Info("handling event")
					err = fh.HandleAppEvent(body)
				case "app_terminated_event":
					eventLogger.Info("handling event")
					err = fh.HandleTerminationEvent(body)
				case "status_update_event":
					eventLogger.Info("handling event")
					err = fh.HandleStatusEvent(body)
				default:
					eventLogger.Info("not handling event")
				}

				if err != nil {
					eventLogger.WithError(err).Error("body generated error")
					continue
				}
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

func makeEventStreamRequest(url string) (*http.Response, error) {
	buffer := make([]byte, 1024)
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(buffer))
	if err != nil {
		log.WithError(err).Error("Could not GET /v2/events")
		os.Exit(1)
	}
	req.Header.Set("Accept", "text/event-stream")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("HTTP request for /v2/events failed!")
		return nil, err
	}

	return resp, nil
}
