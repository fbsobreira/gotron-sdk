//go:build !falcon

package cmd

// pqBuild reports that this binary is a default tronctl build, so
// `tronctl upgrade` installs a plain tronctl archive.
const pqBuild = false
