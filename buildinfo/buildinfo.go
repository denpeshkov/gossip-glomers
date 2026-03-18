// Package buildinfo provides information about the service's build.
package buildinfo

var (
	Name    = "go-template"
	Version = "devel" // Set via -ldflags '-X'
)
