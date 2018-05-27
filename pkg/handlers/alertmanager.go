package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
)

const (
	alertPath   = "/api/v1/alerts"
	contentType = "application/json"
)

type Alert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	GeneratorURL string            `json:"generatorURL"`
}

// Alerts is the end data structure that is converted into JSON and posted to alert managers /api/v1/alerts
// endpoint.
type Alerts []Alert

// AlertManager is the underlying struct used by the alert manager handler receivers
type AlertManager struct {
	endpoint      string
	kindFilters   map[string]struct{}
	typeFilters   map[string]struct{}
	reasonFilters map[string]struct{}
}

type Config struct {
	Kinds   []string `json:"kinds" yaml:"kinds"`
	Types   []string `json:"types" yaml:"types"`
	Reasons []string `json:"reasons" yaml:"reasons"`
}

func New(alertmanagerURL string, config Config) AlertManager {
	endpoint := alertmanagerURL + alertPath
	kf := map[string]struct{}{}
	tf := map[string]struct{}{}
	rf := map[string]struct{}{}
	for _, k := range config.Kinds {
		kf[strings.ToLower(k)] = struct{}{}
	}
	for _, t := range config.Types {
		tf[strings.ToLower(t)] = struct{}{}
	}
	for _, r := range config.Reasons {
		rf[strings.ToLower(r)] = struct{}{}
	}
	return AlertManager{
		endpoint:      endpoint,
		kindFilters:   kf,
		typeFilters:   tf,
		reasonFilters: rf,
	}
}

func (a *AlertManager) ObjectCreated(obj interface{}) {
	event, ok := obj.(*v1.Event)
	if !ok {
		glog.V(2).Infof("object %v is not a *v1.Event", obj)
		return
	}
	kind := strings.ToLower(event.InvolvedObject.Kind)
	typ := strings.ToLower(event.Type)
	reason := strings.ToLower(event.Reason)
	ok1 := len(a.kindFilters) == 0 // if no kind filters, all kinds matched
	if !ok1 {
		_, ok1 = a.kindFilters[kind]
	}
	ok2 := len(a.typeFilters) == 0 // if no type filters, all types matched
	if !ok2 {
		_, ok2 = a.typeFilters[typ]
	}
	ok3 := len(a.reasonFilters) == 0 // if no reason filters, all reasons matched
	if !ok3 {
		_, ok3 = a.reasonFilters[reason]
	}

	if !ok1 && !ok2 && !ok3 {
		glog.Infof("event %v ignored to send alert", event)
		return
	}
	alertName := fmt.Sprintf("%s %s", event.InvolvedObject.Kind, event.Reason)
	labels := map[string]string{
		"alertname":               alertName,
		"namespace":               event.Namespace,
		"name":                    event.Name,
		"component":               event.Source.Component,
		"host":                    event.Source.Host,
		"reason":                  event.Reason,
		"kind":                    event.InvolvedObject.Kind,
		"message":                 event.Message,
		"client":                  "k8swatch",
		"level":                   event.Type,
		"involvedObjectNamespace": event.InvolvedObject.Namespace,
		"involvedObjectName":      event.InvolvedObject.Name,
		"fieldPath":               event.InvolvedObject.FieldPath,
	}
	alerts := Alerts{
		Alert{Labels: labels},
	}
	if err := a.fire(alerts); err != nil {
		glog.Errorf("failed to send alert %v to %s", alerts, a.endpoint)
		return
	}
	glog.Infof("Successfully send alert: %v", alerts)
}

func (a *AlertManager) fire(alerts Alerts) error {
	data, err := json.Marshal(alerts)
	if err != nil {
		return err
	}

	resp, err := http.Post(a.endpoint, contentType, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("fire response %v", resp.Status)
	}
	return nil
}
