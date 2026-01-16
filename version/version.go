package version

import "runtime"

var build = "Unknown"
var gitRev = "Unknown"
var gitBranch = "Unknown"
var gitTag = "Unknown"

type BuildInfo struct {
	BuildTime string `json:"build_time"`
	GitBranch string `json:"git_branch"`
	GitRev    string `json:"git_rev"`
	GitTag    string `json:"git_tag"`
	GoVersion string `json:"-"`
}

func GetBuildInfo() BuildInfo {
	return BuildInfo{
		BuildTime: build,
		GitBranch: gitBranch,
		GitRev:    gitRev,
		GoVersion: runtime.Version(),
		GitTag:    gitTag,
	}
}
