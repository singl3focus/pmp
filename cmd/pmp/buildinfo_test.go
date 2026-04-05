package main

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersionInfoPrefersLdflags(t *testing.T) {
	t.Cleanup(func() {
		readBuildInfo = debug.ReadBuildInfo
	})

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/singl3focus/pmp/cmd/pmp",
				Version: "v0.2.0",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "deadbeef"},
				{Key: "vcs.time", Value: "2026-04-05T12:00:00Z"},
			},
		}, true
	}

	build := resolveVersionInfo("v9.9.9", "abc123", "2026-04-01T00:00:00Z")
	if build.Version != "v9.9.9" {
		t.Fatalf("expected ldflags version to win, got %q", build.Version)
	}
	if build.Commit != "abc123" {
		t.Fatalf("expected ldflags commit to win, got %q", build.Commit)
	}
	if build.Date != "2026-04-01T00:00:00Z" {
		t.Fatalf("expected ldflags date to win, got %q", build.Date)
	}
}

func TestResolveVersionInfoFallsBackToBuildInfo(t *testing.T) {
	t.Cleanup(func() {
		readBuildInfo = debug.ReadBuildInfo
	})

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/singl3focus/pmp/cmd/pmp",
				Version: "v0.2.0",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "deadbeef"},
				{Key: "vcs.time", Value: "2026-04-05T12:00:00Z"},
			},
		}, true
	}

	build := resolveVersionInfo("dev", "none", "unknown")
	if build.Version != "v0.2.0" {
		t.Fatalf("expected module version from build info, got %q", build.Version)
	}
	if build.Commit != "deadbeef" {
		t.Fatalf("expected vcs revision from build info, got %q", build.Commit)
	}
	if build.Date != "2026-04-05T12:00:00Z" {
		t.Fatalf("expected vcs time from build info, got %q", build.Date)
	}
}

func TestResolveVersionInfoKeepsDevWhenBuildInfoIsDevel(t *testing.T) {
	t.Cleanup(func() {
		readBuildInfo = debug.ReadBuildInfo
	})

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/singl3focus/pmp/cmd/pmp",
				Version: "(devel)",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "deadbeef"},
				{Key: "vcs.time", Value: "2026-04-05T12:00:00Z"},
			},
		}, true
	}

	build := resolveVersionInfo("dev", "none", "unknown")
	if build.Version != "dev" {
		t.Fatalf("expected dev version to remain for local builds, got %q", build.Version)
	}
	if build.Commit != "deadbeef" {
		t.Fatalf("expected vcs revision from build info, got %q", build.Commit)
	}
	if build.Date != "2026-04-05T12:00:00Z" {
		t.Fatalf("expected vcs time from build info, got %q", build.Date)
	}
}
