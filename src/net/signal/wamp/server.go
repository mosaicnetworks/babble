package wamp

import (
	"context"
	"log"
	"net/http"

	"github.com/gammazero/nexus/v3/router"
	"github.com/gammazero/nexus/v3/wamp"
)

// Server implements a WAMP server through which connected clients can make RPC
// requests to one-another. It is the server side of our WAMP signaling system
// for WebRTC connections.
type Server struct {
	address    string
	router     router.Router
	httpServer *http.Server
}

// NewServer instantiates a new Server which can be run at a specified address.
func NewServer(address string, realm string) (*Server, error) {
	// Create router instance.
	routerConfig := &router.Config{
		RealmConfigs: []*router.RealmConfig{
			&router.RealmConfig{
				URI:           wamp.URI(realm),
				AnonymousAuth: true,
			},
		},
	}

	nxr, err := router.NewRouter(routerConfig, nil)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	wss := router.NewWebsocketServer(nxr)

	httpServer := &http.Server{
		Handler: wss,
		Addr:    address,
	}

	res := &Server{
		address:    address,
		router:     nxr,
		httpServer: httpServer,
	}

	return res, nil
}

// Run starts the WAMP websocket server
func (s *Server) Run() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown stops the websocket server, and the wamp router
func (s *Server) Shutdown() {
	defer s.router.Close()

	if err := s.httpServer.Shutdown(context.Background()); err != nil {
		log.Println(err)
	}
}

// Addr returns the address of the server
func (s *Server) Addr() string {
	return s.address
}
