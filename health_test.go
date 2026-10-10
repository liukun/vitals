package main

import (
	"testing"
	"time"

	"vitals/internal/core"
)

func TestHealthMonitorRequiresSustainedPressureAndCooldown(t *testing.T) {
	var monitor healthMonitor
	start := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	high := core.Snapshot{CPUPercent: 96}

	if alerts := monitor.Observe(high, start); len(alerts) != 0 {
		t.Fatalf("initial alerts = %#v", alerts)
	}
	if alerts := monitor.Observe(high, start.Add(29*time.Second)); len(alerts) != 0 {
		t.Fatalf("early alerts = %#v", alerts)
	}
	alerts := monitor.Observe(high, start.Add(30*time.Second))
	if len(alerts) != 1 || alerts[0].ID != "health-cpu" {
		t.Fatalf("sustained alerts = %#v", alerts)
	}
	if alerts := monitor.Observe(high, start.Add(time.Minute)); len(alerts) != 0 {
		t.Fatalf("cooldown alerts = %#v", alerts)
	}
	monitor.Observe(core.Snapshot{}, start.Add(2*time.Minute))
	if alerts := monitor.Observe(high, start.Add(3*time.Minute)); len(alerts) != 0 {
		t.Fatalf("reset alerts = %#v", alerts)
	}
}

func TestHealthSummaryIncludesStartupReview(t *testing.T) {
	title, detail, attention := healthSummary(
		core.Snapshot{MemoryPercent: 92},
		[]core.StartupItem{{NeedsReview: true}},
	)
	if title != "建议检查" || !attention || detail != "内存紧张 · 1 个启动项需关注" {
		t.Fatalf("summary = %q, %q, %v", title, detail, attention)
	}
}

func TestApplicationHealthAggregation(t *testing.T) {
	app := newApplication()
	target := core.AppInfo{Path: "/Applications/Example.app", BundleID: "dev.example.app"}
	app.snapshot.MemoryConsumers = []core.MemoryConsumer{{
		AppPath:      target.Path,
		RSS:          512,
		ProcessCount: 3,
	}}
	app.startupItems = []core.StartupItem{
		{AppPath: target.Path, Enabled: true},
		{BundleID: target.BundleID, Enabled: false},
		{Label: target.BundleID + ".helper", Enabled: true},
	}
	memory, processes := app.applicationMemory(target.Path)
	if memory != 512 || processes != 3 {
		t.Fatalf("memory = %d, processes = %d", memory, processes)
	}
	enabled, total := app.applicationStartupCounts(target)
	if enabled != 2 || total != 3 {
		t.Fatalf("startup = %d enabled / %d total", enabled, total)
	}
}
