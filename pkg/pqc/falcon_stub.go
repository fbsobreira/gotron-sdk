//go:build !falcon

package pqc

// falcon is the FN-DSA-512 backend. In builds without the "falcon" tag it is
// a stub: FN-DSA-512 keys can be parsed and addressed, but every
// cryptographic operation returns ErrFalconUnavailable.
var falcon falconBackend = stubFalcon{}

type stubFalcon struct{}

func (stubFalcon) availErr() error { return ErrFalconUnavailable }

func (stubFalcon) generate() ([]byte, []byte, error) { return nil, nil, ErrFalconUnavailable }

func (stubFalcon) sign(_, _ []byte) ([]byte, error) { return nil, ErrFalconUnavailable }

func (stubFalcon) verify(_, _, _ []byte) bool { return false }
