package main

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/egoist/mygo"

	"vitals/internal/core"
)

func TestCompactRate(t *testing.T) {
	tests := []struct {
		value uint64
		want  string
	}{
		{0, "0B/s"},
		{7400, "7.2KB/s"},
		{1048473, "1024KB/s"},
		{1258291, "1.2m/s"},
		{1288490189, "1.2g/s"},
	}
	for _, test := range tests {
		if got := compactRate(test.value); got != test.want {
			t.Errorf("compactRate(%d) = %q, want %q", test.value, got, test.want)
		}
	}
}

func TestMenuBarStatusImageSizeIsStable(t *testing.T) {
	for _, snapshot := range []core.Snapshot{
		{},
		{GPUPercent: 18, CPUPercent: 20, MemoryPercent: 82, NetworkDown: 1258291, NetworkUp: 7400},
	} {
		image, err := png.Decode(bytes.NewReader(menuBarStatusImage(snapshot, defaultSettings())))
		if err != nil {
			t.Fatal(err)
		}
		if got := image.Bounds().Size(); got.X != 274 || got.Y != 36 {
			t.Fatalf("image size = %v, want 274x36", got)
		}
	}
}

func TestMenuBarStatusImageShrinksWithVisibleMetrics(t *testing.T) {
	settings := defaultSettings()
	settings.ShowGPU = false
	settings.ShowRAM = false
	settings.ShowNetwork = false
	image, err := png.Decode(bytes.NewReader(menuBarStatusImage(core.Snapshot{CPUPercent: 20}, settings)))
	if err != nil {
		t.Fatal(err)
	}
	if got := image.Bounds().Size(); got.X != 58 || got.Y != 36 {
		t.Fatalf("image size = %v, want 58x36", got)
	}
}

func TestWindowCloseAllowsApplicationQuit(t *testing.T) {
	app := newApplication()
	ordinaryClose := &mygo.CloseEvent{}
	app.handleWindowClose(ordinaryClose)
	if !ordinaryClose.DefaultPrevented() {
		t.Fatal("ordinary window close should hide the window")
	}

	app.quitting = true
	quitClose := &mygo.CloseEvent{}
	app.handleWindowClose(quitClose)
	if quitClose.DefaultPrevented() {
		t.Fatal("application quit should be allowed to close the window")
	}
}
