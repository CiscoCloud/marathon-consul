package marathon

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUrl(t *testing.T) {
	t.Parallel()

	m, _ := NewMarathon("localhost:8080", "http", nil)
	url := m.Url("/v2/apps")

	assert.Equal(t, url, "http://localhost:8080/v2/apps")
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