package main

import (
	"testing"

	cmd "github.com/fbsobreira/gotron-sdk/cmd/subcommands"
	"github.com/stretchr/testify/require"
)

// A test binary is built without the release ldflags, yet init's build-info
// fallback still fills version in (with "(devel)"). ReleaseBuild must record
// the ldflags, not the filled-in version, or `tronctl upgrade` would take a
// `go install` binary for a release and overwrite it.
func TestInitReleaseBuildIgnoresBuildInfoFallback(t *testing.T) {
	if version == "" {
		t.Skip("no build info in this test binary, so the fallback set no version")
	}
	require.False(t, cmd.ReleaseBuild, "version %q came from the build info, not from ldflags", version)
}
