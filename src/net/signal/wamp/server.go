package wamp

import (
	"log"

	"github.com/gammazero/nexus/v3/router"
	"github.com/gammazero/nexus/v3/wamp"
)

type Server struct {
	address    string
	router     router.Router
	shutdownCh chan struct{}
}

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

	res := &Server{
		address:    address,
		router:     nxr,
		shutdownCh: make(chan struct{}),
	}

	return res, nil
}

func (s *Server) Run() error {
	defer s.router.Close()

	// Create and run server.
	closer, err := router.NewWebsocketServer(s.router).ListenAndServe(s.address)
	if err != nil {
		log.Fatal(err)
		return err
	}
	log.Printf("Websocket server listening on ws://%s/", s.address)

	// Wait for shutdown
	<-s.shutdownCh

	return closer.Close()
}

func (s *Server) Shutdown() {
	close(s.shutdownCh)
}
