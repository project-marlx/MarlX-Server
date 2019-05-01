package socks

import (
	"net"
	"fmt"
)

// GetTCPListener is an alias for:
// GetTCPListenerOnPort(hostname, 8024) since 8024 is the default
// port for a MarlX-Server
func GetTCPListener(hostname string) (*net.TCPListener, error) {
	return GetTCPListenerOnPort(hostname, 8024)
}

// GetTCPListenerOnPort returns a TCP Listener
// for "hostname:port" + an error (if any occured).
func GetTCPListenerOnPort(hostname string, port uint16) (*net.TCPListener, error) {
	var tcpAddr *net.TCPAddr
	var err error
	
	tcpAddr, err = net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", hostname, port))
	if err != nil {
		return nil, err
	}

	return net.ListenTCP("tcp", tcpAddr)
}