//go:build falcon && !cgo

package pqc

// The liboqs backend behind the falcon build tag needs cgo. Without it the
// build fails first with the self-explanatory identifier below (an
// "undefined: falcon" error still follows): build with CGO_ENABLED=1 and
// liboqs discoverable via pkg-config.
var _ = falcon_build_tag_requires_CGO_ENABLED_1_and_liboqs
