package main

import (
	"runtime/debug"

	"github.com/singl3focus/pmp/cli"
)

var readBuildInfo = debug.ReadBuildInfo

func resolveVersionInfo(version, commit, date string) cli.VersionInfo {
	build := cli.VersionInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	info, ok := readBuildInfo()
	if !ok {
		return build
	}

	if build.Version == "" || build.Version == "dev" {
		if resolvedVersion := resolveModuleVersion(info); resolvedVersion != "" {
			build.Version = resolvedVersion
		}
	}

	if build.Commit == "" || build.Commit == "none" {
		if resolvedCommit := buildSetting(info, "vcs.revision"); resolvedCommit != "" {
			build.Commit = resolvedCommit
		}
	}

	if build.Date == "" || build.Date == "unknown" {
		if resolvedDate := buildSetting(info, "vcs.time"); resolvedDate != "" {
			build.Date = resolvedDate
		}
	}

	return build
}

func resolveModuleVersion(info *debug.BuildInfo) string {
	if info == nil {
		return ""
	}

	if isTaggedVersion(info.Main.Version) {
		return info.Main.Version
	}

	return ""
}

func isTaggedVersion(version string) bool {
	return version != "" && version != "(devel)"
}

func buildSetting(info *debug.BuildInfo, key string) string {
	if info == nil {
		return ""
	}

	for _, setting := range info.Settings {
		if setting.Key == key {
			return setting.Value
		}
	}

	return ""
}
