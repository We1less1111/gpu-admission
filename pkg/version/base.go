/*
 * Tencent is pleased to support the open source community by making TKEStack available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */
package version

// Base version information.
//
// This is the fallback data used when version information from git is not
// provided via go ldflags. It provides an approximation of the Apiswitch
// version for ad-hoc builds (e.g. `go build`) that cannot get the version
// information from git.
//
// If you are looking at these fields in the git tree, they look
// strange. They are modified on the fly by the build process. The
// in-tree values are dummy values used for "git archive", which also
// works for GitHub tar downloads.
var (
	// rpm package name, if install by rpm
	rpmPKGName string = "Not build from rpm package"
	// branch of git
	gitBranch string = "Not a git repo"
	// sha1 from git, output of $(git rev-parse HEAD)
	gitCommit string = "$Format:%H$"
	// state of git tree, either "clean" or "dirty"
	gitTreeState string = "Not a git tree"
	// build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
	buildDate string = "1970-01-01T00:00:00Z"
)
