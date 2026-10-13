//go:build falcon

package cmd

// pqBuild reports that this binary is a tronctl-pq build (FN-DSA-512 via
// liboqs), so `tronctl upgrade` must install a tronctl-pq archive.
const pqBuild = true
