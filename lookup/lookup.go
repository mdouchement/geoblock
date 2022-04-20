package lookup

import "net"

// PrivateAddress is the country value for private network.
const PrivateAddress = "-"

// A Lookup is able to compute metadata about an IP.
type Lookup interface {
	Country(ip net.IP) (string, error)
}
