package remote

import (
	"errors"

	"github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/entry"
)

const (
	RemoteHTTP = "http"
	RemoteRPC  = "rpc"
)

var (
	ErrInvalidRemote = errors.New("invalid remote")
)

// A Remote provide an EntryHandler
type Remote interface {
	// start the remote service, this can block until the service is done
	Start() error
	// This method shuts the remote service down
	Close() error
	// connect to the remote service (make sure it is running)
	Connect() error
	// handle an entry: forward it to the remote service
	HandleEntry(entry entry.Entry, options map[string]string) error
}

func GetRemote(config *config.Config) (remote Remote, err error) {
	return GetRemoteAndConnect(config, true)
}

func GetRemoteAndConnect(config *config.Config, check bool) (remote Remote, err error) {
	// set default
	if len(config.Selected) == 0 {
		config.Selected = RemoteHTTP
	}

	// validate rpc config
	rpcConfig, err := GetRPCConfig(config)
	if err != nil {
		return nil, err
	}

	// validate http config
	httpConfig, err := GetHTTPConfig(config)
	if err != nil {
		return nil, err
	}

	// check selection
	switch config.Selected {
	case RemoteHTTP:
		remote = NewHTTPConnection(httpConfig)
	case RemoteRPC:
		remote = &RPCConnection{RPCConfig: rpcConfig}
	default:
		return nil, ErrInvalidRemote
	}

	if check {
		// check connection
		return remote, remote.Connect()
	} else {
		return remote, nil
	}
}
