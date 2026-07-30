package cmd

import "testing"

func TestShortCommit(t *testing.T) {
	tests := []struct {
		name string
		dump string
		want string
	}{
		{
			name: "goreleaser release build",
			dump: "v0.26.0-abc1234",
			want: "abc1234",
		},
		{
			name: "makefile local build carries a dirty suffix",
			dump: "v366-5484f0cc-dirty",
			want: "5484f0cc",
		},
		{
			name: "prerelease tag must not yield the prerelease identifier",
			dump: "v0.27.0-rc1-abc1234",
			want: "abc1234",
		},
		{
			name: "prerelease identifier long enough to look like a hash is still skipped",
			dump: "v0.27.0-beta123-abc1234",
			want: "abc1234",
		},
		{
			// main.go sets VersionWrapDump to the bare version when no commit is
			// recorded. Indexing field 1 here is what panicked in getGitVersion.
			name: "no commit recorded",
			dump: "v0.26.0",
			want: "",
		},
		{
			name: "empty dump",
			dump: "",
			want: "",
		},
		{
			name: "trailing separator leaves an empty field",
			dump: "v0.26.0-",
			want: "",
		},
		{
			name: "abbreviation shorter than a commit hash is rejected",
			dump: "v0.26.0-abc",
			want: "",
		},
		{
			name: "non-hex field of sufficient length is rejected",
			dump: "v0.26.0-snapshot",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortCommit(tt.dump); got != tt.want {
				t.Errorf("shortCommit(%q) = %q, want %q", tt.dump, got, tt.want)
			}
		})
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"abc1234", true},
		{"5484F0CC", true},
		{"", true},
		{"dirty", false},
		{"rc1", false},
		{"beta123", false},
		{"abc123g", false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := isHexString(tt.in); got != tt.want {
				t.Errorf("isHexString(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
