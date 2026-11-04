// Package version holds the provider version, injected at build time via
// -ldflags -X.
package version

// Version is set by the build (GO_LDFLAGS) and is empty in dev builds.
var Version string
