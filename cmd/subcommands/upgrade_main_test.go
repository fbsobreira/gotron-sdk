package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"testing"
)

// The upgrade tests need a "new tronctl" that really runs on every OS,
// Windows included, so they ship this test binary as the release binary: run
// with fakeTronctlEnv=1 it acts as tronctl instead of running the tests.
// None of these names is one of the variables the updater strips before the
// version check (TRONCTL_*, TRONGRID_APIKEY, GOTRON_SDK_DEBUG), so they reach
// the fake.
const (
	fakeTronctlEnv = "GOTRON_TEST_FAKE_TRONCTL" // "1": act as tronctl
	fakeVersionEnv = "GOTRON_TEST_FAKE_VERSION" // the version `version` reports
	fakeExitEnv    = "GOTRON_TEST_FAKE_EXIT"    // the exit code of `version`
	// fakeHomeEnv is the test's home directory, which the fake must not see.
	fakeHomeEnv = "GOTRON_TEST_UPGRADE_HOME"
)

func TestMain(m *testing.M) {
	if os.Getenv(fakeTronctlEnv) == "1" {
		os.Exit(runFakeTronctl())
	}
	os.Exit(m.Run())
}

// runFakeTronctl answers `version` like cmd/tronctl/main.go does, on stderr,
// and fails when the updater did not isolate the version check from the
// user's settings. It prints os.Args[0] verbatim, not path.Base(os.Args[0]):
// that is what main prints on Windows, where path.Base does not split on `\`.
func runFakeTronctl() int {
	if len(os.Args) != 2 || os.Args[1] != "version" {
		_, _ = fmt.Fprintf(os.Stderr, "fake tronctl: unexpected arguments %q\n", os.Args[1:])
		return 2
	}
	if os.Getenv("TRONCTL_SIGNER") != "" {
		_, _ = fmt.Fprintln(os.Stderr, "fake tronctl: TRONCTL_SIGNER reached the version check")
		return 7
	}
	if home := os.Getenv(fakeHomeEnv); home != "" && (os.Getenv("HOME") == home || os.Getenv("USERPROFILE") == home) {
		_, _ = fmt.Fprintln(os.Stderr, "fake tronctl: the version check runs with the user's home directory")
		return 8
	}
	if _, err := os.Stat(".env"); err == nil {
		_, _ = fmt.Fprintln(os.Stderr, "fake tronctl: the version check runs next to a .env file")
		return 9
	} else if !errors.Is(err, fs.ErrNotExist) {
		_, _ = fmt.Fprintf(os.Stderr, "fake tronctl: stat .env: %v\n", err)
		return 10
	}
	code := 0
	if s := os.Getenv(fakeExitEnv); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "fake tronctl: bad %s %q\n", fakeExitEnv, s)
			return 11
		}
		code = n
	}
	_, _ = fmt.Fprintf(os.Stderr, "TronCTL. %s version %s (test now)\n", os.Args[0], os.Getenv(fakeVersionEnv))
	return code
}
