// Package leader provides leader election
package leader

// Leader provides leadership election
type Leader interface {
	// elect leader
	Elect(id string, opts ...ElectOption) (Elected, error)
	// follow the leader
	Follow() chan string
}

type Elected interface {
	// id of leader
	Id() string
	// seek re-election
	Reelect() error
	// resign leadership
	Resign() error
	// observe leadership revocation
	Revoked() chan bool
}

type Option func(o *Options)

type ElectOption func(o *ElectOptions)
