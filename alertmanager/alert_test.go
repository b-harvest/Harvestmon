package main

import "testing"

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
