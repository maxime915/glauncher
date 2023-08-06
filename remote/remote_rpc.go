package remote

import (
	"errors"
	"net/http"
	"net/rpc"
	"net/url"

	"github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/entry"
	"github.com/maxime915/glauncher/utils"
)

const (
	argKindEntry = iota
	argKindStop
	argKindPing
)

var (
	errInvalidArg    = errors.New("invalid argument kind passed")
	errInvalidServer = errors.New("invalid server")
)

// RPCConfig : configuration for the RPC server and remote

type RPCConfig struct {
	Addr string `json:"addr"`
}

func GetRPCConfig(conf *config.Config) (RPCConfig, error) {
	serialized := conf.Remotes[RemoteRPC]
	var err error

	if len(serialized) == 0 {
		config := defaultRPCConfig()
		serialized, err = utils.ValToJSON(config)
		if err != nil {
			return RPCConfig{}, err
		}

		conf.Remotes[RemoteRPC] = serialized
		err = conf.Save()
		return config, err
	}

	var config RPCConfig
	err = utils.FromJSON(serialized, &config)
	if err != nil {
		return RPCConfig{}, err
	}

	err = config.Validate()
	return config, err
}

func defaultRPCConfig() RPCConfig {
	return RPCConfig{Addr: "localhost:8867"}
}

func (c RPCConfig) Validate() error {
	// check if the address is valid
	_, err := url.Parse(c.Addr)
	return err
}

// RPCServer : RPC server to remotely launch entries

type RPCServer struct {
	valid bool
	RPCConfig
	done chan struct{}
}

type RPCArg struct {
	Entry []byte
	Kind  int
}

func newRPCServer(config RPCConfig) *RPCServer {
	return &RPCServer{
		valid:     true,
		RPCConfig: config,
		done:      make(chan struct{}, 1),
	}
}

func (s RPCServer) StartServer() error {
	if !s.valid {
		return errInvalidServer
	}

	err := rpc.Register(s)
	if err != nil {
		return err
	}

	rpc.HandleHTTP()

	server := &http.Server{
		Addr:    s.Addr,
		Handler: nil,
	}

	errChan := make(chan error, 1)

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case <-s.done:
		return nil
	case err := <-errChan:
		return err
	}
}

func (s RPCServer) CloseServer(args *RPCArg, reply *struct{}) error {
	if !s.valid {
		return errInvalidServer
	}

	if args.Kind != argKindStop {
		return errInvalidArg
	}

	s.done <- struct{}{}
	return nil
}

func (s RPCServer) Ping(args *RPCArg, reply *struct{}) error {
	if !s.valid {
		return errInvalidServer
	}

	if args.Kind != argKindPing {
		return errInvalidArg
	}

	return nil
}

func (s RPCServer) HandleEntry(args *RPCArg, reply *struct{}) error {
	if !s.valid {
		return errInvalidServer
	}

	if args.Kind != argKindEntry {
		return errInvalidArg
	}

	e, options, err := entry.DeserializeWithOption(args.Entry)
	if err != nil {
		return err
	}

	return e.RemoteLaunch(options)
}

// RPCConnection : Remote interface to the RPC server

type RPCConnection struct {
	RPCConfig
}

func NewRPCConnection(config RPCConfig) (*RPCConnection, error) {
	return &RPCConnection{
		RPCConfig: config,
	}, nil
}

func (c RPCConnection) connection() (*rpc.Client, error) {
	return rpc.DialHTTP("tcp", c.Addr)
}

func (c *RPCConnection) Start() error {
	server := newRPCServer(c.RPCConfig)
	return server.StartServer()
}
func (c RPCConnection) Close() error {
	client, err := c.connection()
	if err != nil {
		return err
	}

	arg := RPCArg{Kind: argKindStop}
	return client.Call("RPCServer.CloseServer", arg, nil)
}

func (c RPCConnection) Connect() error {
	client, err := c.connection()
	if err != nil {
		return err
	}

	arg := RPCArg{Kind: argKindPing}
	return client.Call("RPCServer.Ping", arg, nil)
}

func (c RPCConnection) HandleEntry(e entry.Entry, options map[string]string) (err error) {
	client, err := c.connection()
	if err != nil {
		return err
	}

	arg := RPCArg{Kind: argKindEntry}
	arg.Entry, err = entry.SerializeWithOptions(e, options)
	if err != nil {
		return err
	}

	return client.Call("RPCServer.HandleEntry", arg, nil)
}
