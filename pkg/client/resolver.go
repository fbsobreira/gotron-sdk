package client

import (
	"errors"
	"strings"

	"google.golang.org/grpc/naming"
)

var errWatcherClose = errors.New("watcher has been closed")

// NewPseudoResolver creates a new pseudo resolver which returns fixed addrs.
func NewPseudoResolver(addrs []string) naming.Resolver {
	return &pseudoResolver{addrs}
}

type pseudoResolver struct {
	addrs []string
}

func (r *pseudoResolver) Resolve(target string) (naming.Watcher, error) {
	w := &pseudoWatcher{
		updatesChan: make(chan []*naming.Update, 1),
	}
	updates := []*naming.Update{}
	for _, addr := range r.addrs {
		updates = append(updates, &naming.Update{Op: naming.Add, Addr: addr})
	}
	w.updatesChan <- updates
	return w, nil
}

// This watcher is implemented based on ipwatcher below
// https://github.com/grpc/grpc-go/blob/30fb59a4304034ce78ff68e21bd25776b1d79488/naming/dns_resolver.go#L151-L171
type pseudoWatcher struct {
	updatesChan chan []*naming.Update
}

func (w *pseudoWatcher) Next() ([]*naming.Update, error) {
	us, ok := <-w.updatesChan
	if !ok {
		return nil, errWatcherClose
	}
	return us, nil
}

func (w *pseudoWatcher) Close() {
	close(w.updatesChan)
}

func GetIPList() []string {
	return strings.Split(
		strings.Replace(
			`54.236.37.243:50051,
			52.53.189.99:50051,
			18.196.99.16:50051,
			34.253.187.192:50051,
			52.56.56.149:50051,
			35.180.51.163:50051,
			54.252.224.209:50051,
			18.228.15.36:50051,
			52.15.93.92:50051,
			34.220.77.106:50051,
			13.127.47.162:50051,
			13.124.62.58:50051,
			13.229.128.108:50051,
			35.182.37.246:50051,
			47.90.215.84:50051,
			47.254.77.146:50051,
			47.74.242.55:50051,
			47.75.249.119:50051,
			47.90.201.118:50051,
			34.250.140.143:50051,
			35.176.192.130:50051,
			52.47.197.188:50051,
			52.62.210.100:50051,
			13.231.4.243:50051,
			47.254.27.69:50051,
			35.154.90.144:50051,
			13.125.210.234:50051,
			47.88.174.175:50051,
			47.75.249.4:50051,
			grpc.trongrid.io:50051`,
			"\n\t\t\t", "", -1), ",")
}
