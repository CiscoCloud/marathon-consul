package marathon

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/CiscoCloud/marathon-consul/apps"
	"github.com/CiscoCloud/marathon-consul/tasks"
	log "github.com/Sirupsen/logrus"
	version "github.com/hashicorp/go-version"
	"github.com/sethgrid/pester"
)

type Marathoner interface {
	Apps() ([]*apps.App, error)
	Tasks(string) ([]*tasks.Task, error)
}

type Marathon struct {
	Location    string
	Protocol    string
	Auth        *url.Userinfo
	NoVerifySsl bool
}

func NewMarathon(location, protocol string, auth *url.Userinfo) (Marathon, error) {
	return Marathon{location, protocol, auth, false}, nil
}

func (m Marathon) Url(path string) string {
	marathon := url.URL{
		Scheme: m.Protocol,
		User:   m.Auth,
		Host:   m.Location,
		Path:   path,
	}

	return marathon.String()
}

func (m Marathon) getClient() *pester.Client {
	client := pester.New()
	client.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: m.NoVerifySsl,
		},
	}

	return client
}

func (m Marathon) Apps() ([]*apps.App, error) {
	log.WithField("location", m.Location).Debug("asking Marathon for apps")
	client := m.getClient()

	request, err := http.NewRequest("GET", m.Url("/v2/apps"), nil)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	request.Header.Add("Accept", "application/json")

	appsResponse, err := client.Do(request)
	if err != nil || (appsResponse.StatusCode != 200) {
		m.logHTTPError(appsResponse, err)
		return nil, err
	}

	body, err := ioutil.ReadAll(appsResponse.Body)
	if err != nil {
		m.logHTTPError(appsResponse, err)
		return nil, err
	}

	appList, err := m.ParseApps(body)
	if err != nil {
		m.logHTTPError(appsResponse, err)
	}

	return appList, err
}

type AppResponse struct {
	Apps []*apps.App `json:"apps"`
}

func (m Marathon) ParseApps(jsonBlob []byte) ([]*apps.App, error) {
	apps := &AppResponse{}
	err := json.Unmarshal(jsonBlob, apps)

	return apps.Apps, err
}

func (m Marathon) Version() (*version.Version, error) {
	log.WithField("location", m.Location).Debug("asking Marathon for its version")
	client := m.getClient()

	request, err := http.NewRequest("GET", m.Url("/v2/info"), nil)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	request.Header.Add("Accept", "application/json")

	infoResponse, err := client.Do(request)
	if err != nil || (infoResponse.StatusCode != 200) {
		m.logHTTPError(infoResponse, err)
		return nil, err
	}

	body, err := ioutil.ReadAll(infoResponse.Body)
	if err != nil {
		m.logHTTPError(infoResponse, err)
		return nil, err
	}

	v, err := m.ParseVersion(body)
	if err != nil {
		m.logHTTPError(infoResponse, err)
		return nil, err
	}

	parsedVersion, err := version.NewVersion(v)
	if err != nil {
		log.WithError(err).Error("error parsing version: %s", v)
		return nil, err
	}

	return parsedVersion, err
}

type InfoResponse struct {
	Version string `json:"version"`
}

func (m Marathon) ParseVersion(resp []byte) (string, error) {
	info := &InfoResponse{}
	err := json.Unmarshal(resp, info)
	return info.Version, err
}

func (m Marathon) Tasks(app string) ([]*tasks.Task, error) {
	log.WithFields(log.Fields{
		"location": m.Location,
		"app":      app,
	}).Debug("asking Marathon for tasks")
	client := m.getClient()

	if app[0] == '/' {
		app = app[1:]
	}

	request, err := http.NewRequest("GET", m.Url(fmt.Sprintf("/v2/apps/%s/tasks", app)), nil)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	request.Header.Add("Accept", "application/json")

	tasksResponse, err := client.Do(request)
	if err != nil || (tasksResponse.StatusCode != 200) {
		m.logHTTPError(tasksResponse, err)
		return nil, err
	}

	body, err := ioutil.ReadAll(tasksResponse.Body)
	if err != nil {
		m.logHTTPError(tasksResponse, err)
		return nil, err
	}

	taskList, err := m.ParseTasks(body)
	if err != nil {
		m.logHTTPError(tasksResponse, err)
	}

	return taskList, err
}

type TasksResponse struct {
	Tasks []*tasks.Task `json:"tasks"`
}

func (m Marathon) ParseTasks(jsonBlob []byte) ([]*tasks.Task, error) {
	tasks := &TasksResponse{}
	err := json.Unmarshal(jsonBlob, tasks)

	return tasks.Tasks, err
}

func (m Marathon) logHTTPError(resp *http.Response, err error) {
	var statusCode string = "???"
	if resp != nil {
		statusCode = string(resp.StatusCode)
	}

	log.WithFields(log.Fields{
		"location":   m.Location,
		"protocol":   m.Protocol,
		"statusCode": statusCode,
	}).Error(err)
}
