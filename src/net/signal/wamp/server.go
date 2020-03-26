package wamp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/gammazero/nexus/v3/router"
	"github.com/gammazero/nexus/v3/wamp"
	"github.com/sirupsen/logrus"
)

// Server implements a WAMP server through which connected clients can make RPC
// requests to one-another. It is the server side of our WAMP signaling system
// for WebRTC connections.
type Server struct {
	address    string
	router     router.Router
	httpServer *http.Server
	logger     *logrus.Entry
}

// NewServer instantiates a new Server which can be run at a specified address.
func NewServer(address string,
	realm string,
	certFile string,
	keyFile string,
	logger *logrus.Entry) (*Server, error) {

	// Create router instance.
	routerConfig := &router.Config{
		RealmConfigs: []*router.RealmConfig{
			&router.RealmConfig{
				URI:           wamp.URI(realm),
				AnonymousAuth: true,
			},
		},
	}

	nxr, err := router.NewRouter(routerConfig, logger)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	wss := router.NewWebsocketServer(nxr)

	// prepare tls config with certFile and keyFile
	tlscfg := &tls.Config{}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("error loading X509 key pair: %s", err)
	}
	tlscfg.Certificates = append(tlscfg.Certificates, cert)

	httpServer := &http.Server{
		Handler:   wss,
		Addr:      address,
		TLSConfig: tlscfg,
	}

	res := &Server{
		address:    address,
		router:     nxr,
		httpServer: httpServer,
		logger:     logger,
	}

	return res, nil
}

// Run starts the WAMP websocket server
func (s *Server) Run() error {
	// The call to ListenAndServeTLS has empty arguments because the
	// certificates have already been loaded in the TLSConfig of the server in
	// the constructor
	err := s.httpServer.ListenAndServeTLS("", "")
	if err != nil && err != http.ErrServerClosed {
		s.logger.WithError(err).Error("Run")
	}
	return err
}

// Shutdown stops the websocket server, and the wamp router
func (s *Server) Shutdown() {
	defer s.router.Close()

	if err := s.httpServer.Shutdown(context.Background()); err != nil {
		s.logger.WithError(err).Error("Shutting down http server")
	}
}

// Addr returns the address of the server
func (s *Server) Addr() string {
	return s.address
}
