package service

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/mosaicnetworks/babble/src/node"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/sirupsen/logrus"
)

// Service ...
type Service struct {
	sync.Mutex

	bindAddress string
	node        *node.Node
	graph       *node.Graph
	logger      *logrus.Entry
}

// NewService ...
func NewService(bindAddress string, n *node.Node, logger *logrus.Entry) *Service {
	service := Service{
		bindAddress: bindAddress,
		node:        n,
		graph:       node.NewGraph(n),
		logger:      logger,
	}

	service.registerHandlers()

	return &service
}

// registerHandlers registers the API handlers with the DefaultServerMux of the
// http package. It is possible that another server in the same process is
// simultaneously using the DefaultServerMux. In which case, the handlers will
// be accessible from both servers. This is usefull when Babble is used
// in-memory and expecpted to use the same endpoint (address:port) as the
// application's API.
func (s *Service) registerHandlers() {
	s.logger.Debug("Registering Babble API handlers")
	http.HandleFunc("/stats", s.makeHandler(s.GetStats))
	http.HandleFunc("/block/", s.makeHandler(s.GetBlock))
	http.HandleFunc("/graph", s.makeHandler(s.GetGraph))
	http.HandleFunc("/peers", s.makeHandler(s.GetPeers))
	http.HandleFunc("/genesispeers", s.makeHandler(s.GetGenesisPeers))
}

func (s *Service) makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.Lock()
		defer s.Unlock()

		// enable CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")

		fn(w, r)
	}
}

// Serve calls ListenAndServe. This is a blocking call. It is not necessary to
// call Serve when Babble is used in-memory and another server has already been
// started with the DefaultServerMux and the same address:port combination.
// Indeed, Babble API handlers have already been registered when the service was
// instantiated.
func (s *Service) Serve() {
	s.logger.WithField("bind_address", s.bindAddress).Debug("Serving Babble API")

	// Use the DefaultServerMux
	err := http.ListenAndServe(s.bindAddress, nil)
	if err != nil {
		s.logger.Error(err)
	}
}

// GetStats ...
func (s *Service) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := s.node.GetStats()

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(stats)
}

// GetBlock ...
func (s *Service) GetBlock(w http.ResponseWriter, r *http.Request) {
	param := r.URL.Path[len("/block/"):]

	blockIndex, err := strconv.Atoi(param)

	if err != nil {
		s.logger.WithError(err).Errorf("Parsing block_index parameter %s", param)

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	block, err := s.node.GetBlock(blockIndex)

	if err != nil {
		s.logger.WithError(err).Errorf("Retrieving block %d", blockIndex)

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(block)
}

// GetGraph ...
func (s *Service) GetGraph(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)

	res, _ := s.graph.GetInfos()

	encoder.Encode(res)
}

// GetPeers ...
func (s *Service) GetPeers(w http.ResponseWriter, r *http.Request) {
	returnPeerSet(w, r, s.node.GetPeers())
}

// GetGenesisPeers ...
func (s *Service) GetGenesisPeers(w http.ResponseWriter, r *http.Request) {
	returnPeerSet(w, r, s.node.GetGenesisPeers())
}

func returnPeerSet(w http.ResponseWriter, r *http.Request, peers []*peers.Peer) {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)

	encoder.Encode(peers)
}
