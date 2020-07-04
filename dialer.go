package oob

import (
	"context"
	"net"
)

// Dialer - wrapper around *net.Dialer that wraps net.UnixConn in oob.UnixConn
type Dialer struct {
	*net.Dialer
}

// Dial - wraps *net.Dialer.Dial such that net.UnixConn is returned as oob.UnixConn
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	dialer := d.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	conn, err := dialer.Dial(network, address)
	if unixConn, ok := conn.(*net.UnixConn); ok && err == nil {
		return New(unixConn), nil
	}
	return conn, err
}

// DialContext - wraps *net.Dialer.DialContext such that net.UnixConn is returned as oob.UnixConn
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := d.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	conn, err := dialer.DialContext(ctx, network, address)
	if unixConn, ok := conn.(*net.UnixConn); ok && err == nil {
		return New(unixConn), nil
	}
	return conn, err
}
