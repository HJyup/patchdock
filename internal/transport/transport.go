package transport

import (
	"context"
	"fmt"
	"net"
	"os"
)

func Listen(socket string) (net.Listener, error) {
	l, err := net.Listen("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise listener to %v: %w", socket, err)
	}

	if err := os.Chmod(socket, 0o600); err != nil {
		l.Close()
		return nil, fmt.Errorf("failed to change the permission to the socket %v: %w", socket, err)
	}

	return l, nil
}

func Dial(ctx context.Context, socket string) (net.Conn, error) {
	var d net.Dialer

	conn, err := d.DialContext(ctx, "unix", socket)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", socket, err)
	}

	return conn, nil
}
