// Package bus wires the convert service onto the NATS observer bus via the shared
// cofiswarm-observer-sdk service component: it announces presence and serves
// .convert.{info,health} alongside the convert HTTP job API.
package bus

import (
	"github.com/keepdevops/cofiswarm-observer-sdk/pkg/servicecomponent"
)

// Routes wires the convert service's capability subjects. Reply field names carry
// schema_version for the major-version gate, mirroring the other service components.
func Routes() map[string]servicecomponent.Handler {
	return map[string]servicecomponent.Handler{
		servicecomponent.Prefix + ".convert.info":   infoHandler(),
		servicecomponent.Prefix + ".convert.health": healthHandler(),
	}
}

func infoHandler() servicecomponent.Handler {
	return func([]byte) (any, error) {
		return infoReply{SchemaVersion: servicecomponent.SchemaVersion, OK: true, Service: "convert"}, nil
	}
}

func healthHandler() servicecomponent.Handler {
	return func([]byte) (any, error) {
		return healthReply{SchemaVersion: servicecomponent.SchemaVersion, OK: true, Status: "ok"}, nil
	}
}

type infoReply struct {
	SchemaVersion string `json:"schema_version"`
	OK            bool   `json:"ok"`
	Error         string `json:"error,omitempty"`
	Service       string `json:"service"`
}

type healthReply struct {
	SchemaVersion string `json:"schema_version"`
	OK            bool   `json:"ok"`
	Error         string `json:"error,omitempty"`
	Status        string `json:"status"`
}
