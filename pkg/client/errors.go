package client

import "errors"

// ErrEstimateEnergyNotSupported is returned when the connected TRON node
// does not support the EstimateEnergy RPC.
var ErrEstimateEnergyNotSupported = errors.New("this node does not support estimate energy")
