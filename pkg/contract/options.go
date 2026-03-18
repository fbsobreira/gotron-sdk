package contract

// Option configures a ContractCall.
type Option func(*callConfig)

type callConfig struct {
	feeLimit     int64
	callValue    int64
	tokenID      string
	tokenAmount  int64
	permissionID *int32
}

// WithFeeLimit sets the maximum TRX (in SUN) the caller is willing to spend
// on energy for a state-changing contract call.
func WithFeeLimit(limit int64) Option {
	return func(c *callConfig) {
		c.feeLimit = limit
	}
}

// WithCallValue sets the TRX amount (in SUN) sent along with the call.
func WithCallValue(value int64) Option {
	return func(c *callConfig) {
		c.callValue = value
	}
}

// WithTokenValue sets the TRC10 token ID and amount sent with the call.
func WithTokenValue(tokenID string, amount int64) Option {
	return func(c *callConfig) {
		c.tokenID = tokenID
		c.tokenAmount = amount
	}
}

// WithPermissionID sets the permission ID for multi-signature transactions.
func WithPermissionID(id int32) Option {
	return func(c *callConfig) {
		c.permissionID = &id
	}
}
