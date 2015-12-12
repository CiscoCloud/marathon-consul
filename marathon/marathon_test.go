package marathon

import (
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestUrl(t *testing.T) {
	t.Parallel()

	m, _ := NewMarathon("localhost:8080", "http", nil)
	url := m.Url("/v2/apps")

	assert.Equal(t, url, "http://localhost:8080/v2/apps")
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	infoBlob := []byte(`{
		"http_config": {
			"https_port": 8443,
			"http_port": 8080,
			"assets_path": null
		},
		"name": "marathon",
		"version": "0.11.1",
		"elected": true,
		"leader": "marathon-leader.example.com:8080",
		"frameworkId": "20150714-191408-4163031306-5050-1590-0000",
		"marathon_config": {
			"mesos_leader_ui_url": "http://marathon-leader.example.com:5050/",
			"leader_proxy_read_timeout_ms": 10000,
			"leader_proxy_connection_timeout_ms": 5000,
			"executor": "//cmd",
			"local_port_max": 20000,
			"local_port_min": 10000,
			"checkpoint": true,
			"ha": true,
			"framework_name": "marathon",
			"failover_timeout": 604800,
			"master": "zk://zk.example.com:2181/mesos",
			"hostname": "marathon-leader.example.com",
			"webui_url": null,
			"mesos_role": null,
			"task_launch_timeout": 300000,
			"reconciliation_initial_delay": 15000,
			"reconciliation_interval": 300000,
			"marathon_store_timeout": 2000,
			"mesos_user": "root"
		},
		"zookeeper_config": {
			"zk_max_versions": 25,
			"zk_session_timeout": 1800000,
			"zk_timeout": 10000,
			"zk": "zk://zk.example.com:2181/marathon"
		},
		"event_subscriber": {
			"http_endpoints": null,
			"type": "http_callback"
		}
  }`)
	m, _ := NewMarathon("localhost:8080", "http", nil)
	v, err := m.ParseVersion(infoBlob)
	assert.Equal(t, v, "0.11.1")
	assert.Nil(t, err)

	// quickly verify that this version can be parsed with the version library
	_, err = version.NewVersion(v)
	assert.Nil(t, err)
}

func TestParseApps(t *testing.T) {
	t.Parallel()

	appBlob := []byte(`{
    "apps": [{
        "args": null,
        "backoffFactor": 1.15,
        "backoffSeconds": 1,
        "maxLaunchDelaySeconds": 3600,
        "cmd": "python3 -m http.server 8080",
        "constraints": [],
        "container": {
            "docker": {
                "image": "python:3",
                "network": "BRIDGE",
                "portMappings": [
                    {"containerPort": 8080, "hostPort": 0, "servicePort": 9000, "protocol": "tcp"},
                    {"containerPort": 161, "hostPort": 0, "protocol": "udp"}
                ]
            },
            "type": "DOCKER",
            "volumes": []
        },
        "cpus": 0.5,
        "dependencies": [],
        "deployments": [],
        "disk": 0.0,
        "env": {},
        "executor": "",
        "healthChecks": [{
            "command": null,
            "gracePeriodSeconds": 5,
            "intervalSeconds": 20,
            "maxConsecutiveFailures": 3,
            "path": "/",
            "portIndex": 0,
            "protocol": "HTTP",
            "timeoutSeconds": 20
        }],
        "id": "/bridged-webapp",
        "instances": 2,
        "mem": 64.0,
        "ports": [10000, 10001],
        "requirePorts": false,
        "storeUrls": [],
        "tasksRunning": 2,
        "tasksHealthy": 2,
        "tasksUnhealthy": 0,
        "tasksStaged": 0,
        "upgradeStrategy": {"minimumHealthCapacity": 1.0},
        "uris": [],
        "user": null,
        "version": "2014-09-25T02:26:59.256Z"
    }
]}
`)

	m, _ := NewMarathon("localhost:8080", "http", nil)
	apps, err := m.ParseApps(appBlob)
	assert.Nil(t, err)
	assert.Equal(t, len(apps), 1)
}

func TestParseTasks(t *testing.T) {
	t.Parallel()

	tasksBlob := []byte(`{
    "tasks": [
        {
            "appId": "/test",
            "host": "192.168.2.114",
            "id": "test.47de43bd-1a81-11e5-bdb6-e6cb6734eaf8",
            "ports": [31315],
            "stagedAt": "2015-06-24T14:57:06.353Z",
            "startedAt": "2015-06-24T14:57:06.466Z",
            "version": "2015-06-24T14:56:57.466Z"
        },
        {
            "appId": "/test",
            "host": "192.168.2.114",
            "id": "test.4453212c-1a81-11e5-bdb6-e6cb6734eaf8",
            "ports": [31797],
            "stagedAt": "2015-06-24T14:57:00.474Z",
            "startedAt": "2015-06-24T14:57:00.611Z",
            "version": "2015-06-24T14:56:57.466Z"
        }
    ]
}
`)

	m, _ := NewMarathon("localhost:8080", "http", nil)
	tasks, err := m.ParseTasks(tasksBlob)
	assert.Nil(t, err)
	assert.Equal(t, len(tasks), 2)
}
