package txbuilder

// Option configures a Tx.
type Option func(*config)

type config struct {
	permissionID *int32
	memo         string
}

func applyOptions(opts []Option) config {
	var cfg config
	for _, o := range opts {
		o(&cfg)
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
