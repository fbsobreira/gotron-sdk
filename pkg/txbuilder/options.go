package txbuilder

import "time"

// Option configures a Tx.
type Option func(*config)

type config struct {
	permissionID *int32
	memo         string
	pollInterval time.Duration
}

func applyOptions(opts []Option) config {
	var cfg config
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	return cfg
}

// WithMemo attaches a memo (stored in RawData.Data) to the transaction.
func WithMemo(memo string) Option {
	return func(c *config) {
		c.memo = memo
	}
}

// WithPermissionID sets the permission ID used for multi-signature transactions.
func WithPermissionID(id int32) Option {
	return func(c *config) {
		c.permissionID = &id
	}
}

// WithPollInterval sets the interval between confirmation checks in
// SendAndConfirm. If not set, txcore.DefaultPollInterval is used.
func WithPollInterval(d time.Duration) Option {
	return func(c *config) {
		c.pollInterval = d
	}
}
