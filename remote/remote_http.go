package remote

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/entry"
	"github.com/maxime915/glauncher/utils"
)

const (
	routePing     = "/ping"
	routeHandle   = "/"
	routeClose    = "/close"
	ParameterAddr = "addr"
)

var (
	ErrInvalidAddress = errors.New("parameter '" + ParameterAddr + "' is invalid")
)

type ErrInvalidStatus struct {
	route          string
	expectedStatus int
	response       http.Response
}

func (e ErrInvalidStatus) Error() string {
	errHeader := fmt.Sprintf("err at %s: expected %s but received %v.", e.route, http.StatusText(e.expectedStatus), e.response.Status)

	msgBytes, err := io.ReadAll(e.response.Body)
	var errBody string
	if err == nil {
		errBody = "response body: " + string(msgBytes)
	} else {
		errBody = "error while decoding body: " + err.Error()
	}

	return errHeader + errBody
}

// HTTPConfig : configuration for the HTTP server and remote

type HTTPConfig struct {
	Addr string `json:"addr"`
}

func GetHTTPConfig(conf *config.Config) (HTTPConfig, error) {
	serialized := conf.Remotes[RemoteHTTP]
	var err error

	if len(serialized) == 0 {
		config := defaultHTTPConfig()
		serialized, err = utils.ValToJSON(config)
		if err != nil {
			return HTTPConfig{}, err
		}

		conf.Remotes[RemoteHTTP] = serialized
		err = conf.Save()
		return config, err
	}

	var config HTTPConfig
	err = utils.FromJSON(serialized, &config)
	if err != nil {
		return HTTPConfig{}, err
	}

	err = config.Validate()
	return config, err
}

func defaultHTTPConfig() HTTPConfig {
	return HTTPConfig{Addr: "localhost:8080"}
}

func (c HTTPConfig) Validate() error {
	// check if the address is valid
	_, err := url.Parse(c.Addr)
	return err
}

// HTTPConnection : Remote interface to the HTTP server, and server itself

type HTTPConnection struct {
	HTTPConfig
	done chan struct{}
}

func NewHTTPConnection(config HTTPConfig) HTTPConnection {
	return HTTPConnection{
		HTTPConfig: config,
		done:       make(chan struct{}, 1),
	}
}

func (c HTTPConnection) url(route string) string {
	return "http://" + c.Addr + route
}

func (c HTTPConnection) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(routePing, httpPing)
	mux.HandleFunc(routeHandle, httpReceiver)
	mux.HandleFunc(routeClose, c.closeRoute)

	errChan := make(chan error, 1)

	go func() {
		err := http.ListenAndServe(c.Addr, mux)
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case <-c.done:
		return nil
	case err := <-errChan:
		return err
	}
}

func (c HTTPConnection) Close() error {
	resp, err := http.Get(c.url(routeClose))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return ErrInvalidStatus{routeClose, http.StatusOK, *resp}
	}
	return nil
}

func (c HTTPConnection) closeRoute(rw http.ResponseWriter, req *http.Request) {
	select {
	case <-c.done:
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("already stopped"))
	default:
		close(c.done)
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("server stopped"))
	}
}

func (c HTTPConnection) Connect() error {
	// localhost server, 500ms is more than enough
	client := http.Client{Timeout: time.Second / 2}
	resp, err := client.Get(c.url(routePing))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return ErrInvalidStatus{routePing, http.StatusOK, *resp}
	}
	return nil
}

func httpPing(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func (c HTTPConnection) HandleEntry(e entry.Entry, options map[string]string) error {
	data, err := entry.SerializeWithOptions(e, options)
	if err != nil {
		return err
	}

	resp, err := http.Post(c.url(routeHandle), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return ErrInvalidStatus{routeHandle, http.StatusOK, *resp}
	}

	return nil
}

func httpReceiver(rw http.ResponseWriter, req *http.Request) {
	data, err := io.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(err.Error()))
		return
	}

	e, options, err := entry.DeserializeWithOption(data)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(err.Error()))
		return
	}

	err = e.RemoteLaunch(options)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	rw.WriteHeader(http.StatusOK)
}
