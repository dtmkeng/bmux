package bmux

import "net"

// Listener ...
type Listener struct {
	*net.TCPListener
}
