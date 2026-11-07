package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/prometheus/common/model"
)

func TestPdDedupKey(t *testing.T) {
	got := pdDedupKey(InstanceName("node-1"), alertEvent("disk-full"))
	want := "node-1_disk-full"
	if got != want {
		t.Fatalf("pdDedupKey() = %q, want %q", got, want)
	}
}

// Regression guard for the bug where PagerDuty DedupKey was derived from the
// instance alone, causing unrelated alert events on the same instance to
// collapse into a single incident (resolving one closed all of them).
func TestPdDedupKey_DistinctPerAlertEvent(t *testing.T) {
	diskFull := pdDedupKey(InstanceName("node-1"), alertEvent("disk-full"))
	cpuHigh := pdDedupKey(InstanceName("node-1"), alertEvent("cpu-high"))
	if diskFull == cpuHigh {
		t.Fatalf("expected distinct dedup keys for different alert events on the same instance, both were %q", diskFull)
	}
}

// capturePDEvents replaces pdSendEvent for the duration of the test so events
// are recorded instead of sent to PagerDuty.
func capturePDEvents(t *testing.T) *[]pagerduty.V2Event {
	t.Helper()
	var sent []pagerduty.V2Event
	orig := pdSendEvent
	pdSendEvent = func(_ context.Context, e pagerduty.V2Event) (*pagerduty.V2EventResponse, error) {
		sent = append(sent, e)
		return &pagerduty.V2EventResponse{}, nil
	}
	t.Cleanup(func() { pdSendEvent = orig })
	return &sent
}

func assertPDEvent(t *testing.T, sent []pagerduty.V2Event, action string, wantDetails map[string]string) {
	t.Helper()
	if len(sent) != 1 {
		t.Fatalf("sent %d events, want 1", len(sent))
	}
	e := sent[0]
	if e.Action != action {
		t.Errorf("Action = %q, want %q", e.Action, action)
	}
	if e.RoutingKey != "test-routing-key" {
		t.Errorf("RoutingKey = %q, want %q", e.RoutingKey, "test-routing-key")
	}
	if want := pdDedupKey("node-1", "disk-full"); e.DedupKey != want {
		t.Errorf("DedupKey = %q, want %q", e.DedupKey, want)
	}
	if e.Payload == nil {
		t.Fatal("Payload is nil, want custom_details for Event Orchestration routing")
	}
	if !reflect.DeepEqual(e.Payload.Details, wantDetails) {
		t.Errorf("Payload.Details = %v, want %v", e.Payload.Details, wantDetails)
	}
}

func testPDConfig() *PDConfig {
	return &PDConfig{Enabled: true, ApiKey: "test-routing-key", DefaultSeverity: "critical"}
}

func testAlertMessage(status model.AlertStatus) *AlertMessage {
	return &AlertMessage{
		Instance:   "node-1",
		AlertEvent: "disk-full",
		AlertLevel: "critical",
		Summary:    "disk usage above 90%",
		Status:     status,
		Labels:     map[string]string{"chain": "cosmoshub", "network": "mainnet", "team": "infra"},
	}
}

func TestPDNotify_TriggerCarriesChain(t *testing.T) {
	sent := capturePDEvents(t)
	if _, err := testPDConfig().notify(testAlertMessage(model.AlertFiring)); err != nil {
		t.Fatal(err)
	}
	assertPDEvent(t, *sent, "trigger", map[string]string{
		"chain": "cosmoshub", "network": "mainnet", "instance": "node-1", "alertEvent": "disk-full", "alertLevel": "critical",
	})
}

func TestPDNotify_ResolveCarriesChain(t *testing.T) {
	sent := capturePDEvents(t)
	if _, err := testPDConfig().notify(testAlertMessage(model.AlertResolved)); err != nil {
		t.Fatal(err)
	}
	assertPDEvent(t, *sent, "resolve", map[string]string{
		"chain": "cosmoshub", "network": "mainnet", "instance": "node-1", "alertEvent": "disk-full", "alertLevel": "critical",
	})
}

func TestPDAcknowledge_CarriesChain(t *testing.T) {
	sent := capturePDEvents(t)
	labels := map[string]string{"chain": "cosmoshub", "network": "mainnet"}
	if err := testPDConfig().acknowledge("node-1", "disk-full", labels); err != nil {
		t.Fatal(err)
	}
	assertPDEvent(t, *sent, "acknowledge", map[string]string{
		"chain": "cosmoshub", "network": "mainnet", "instance": "node-1", "alertEvent": "disk-full",
	})
}

func TestPDResolve_CarriesChain(t *testing.T) {
	sent := capturePDEvents(t)
	labels := map[string]string{"chain": "cosmoshub", "network": "mainnet"}
	if err := testPDConfig().resolve("node-1", "disk-full", labels); err != nil {
		t.Fatal(err)
	}
	assertPDEvent(t, *sent, "resolve", map[string]string{
		"chain": "cosmoshub", "network": "mainnet", "instance": "node-1", "alertEvent": "disk-full",
	})
}

// When the Slack path can't look up the agent, the event must still be sent
// (it falls to the catch-all Service) rather than dropped.
func TestPDAcknowledge_WithoutLabelsStillSends(t *testing.T) {
	sent := capturePDEvents(t)
	if err := testPDConfig().acknowledge("node-1", "disk-full", nil); err != nil {
		t.Fatal(err)
	}
	assertPDEvent(t, *sent, "acknowledge", map[string]string{
		"instance": "node-1", "alertEvent": "disk-full",
	})
}
