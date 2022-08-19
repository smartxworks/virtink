package tcpproxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"inet.af/tcpproxy"

	"github.com/smartxworks/virtink/pkg/daemon"
)

func NewRelayProvider() daemon.RelayProvider {
	return &relayProvider{}
}

type relayProvider struct{}

func (p *relayProvider) RelaySocketToTCP(ctx context.Context, socketPath string, tcpAddr string, tlsConfig *tls.Config) error {
	proxy := &tcpproxy.Proxy{
		ListenFunc: func(_ string, _ string) (net.Listener, error) {
			if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
				return nil, fmt.Errorf("create socket directory: %s", err)
			}
			if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("remove stale socket: %s", err)
			}
			return net.Listen("unix", socketPath)
		},
	}
	proxy.AddRoute("", &tcpproxy.DialProxy{
		DialContext: func(ctx context.Context, _ string, _ string) (net.Conn, error) {
			return (&tls.Dialer{Config: tlsConfig}).DialContext(ctx, "tcp", tcpAddr)
		},
	})

	go func() {
		<-ctx.Done()
		proxy.Close()
	}()

	return proxy.Start()
}

func (p *relayProvider) RelayTCPToSocket(ctx context.Context, tcpAddr string, tlsConfig *tls.Config, socketPath string) (int, error) {
	var port int
	proxy := &tcpproxy.Proxy{
		ListenFunc: func(_ string, _ string) (net.Listener, error) {
			l, err := tls.Listen("tcp", tcpAddr, tlsConfig)
			if err != nil {
				return nil, err
			}
			port = l.Addr().(*net.TCPAddr).Port
			return l, nil
		},
	}
	proxy.AddRoute("", &tcpproxy.DialProxy{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return new(net.Dialer).DialContext(ctx, "unix", socketPath)
		},
	})

	go func() {
		<-ctx.Done()
		proxy.Close()
	}()

	if err := proxy.Start(); err != nil {
		return 0, err
	}
	return port, nil
}
