package service

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mosaicnetworks/babble/src/node"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/sirupsen/logrus"
)

// Service ...
type Service struct {
	bindAddress string
	node        *node.Node
	graph       *node.Graph
	logger      *logrus.Logger
}

// NewService ...
func NewService(bindAddress string, n *node.Node, logger *logrus.Logger) *Service {
	service := Service{
		bindAddress: bindAddress,
		node:        n,
		graph:       node.NewGraph(n),
		logger:      logger,
	}

	return &service
}

// Serve registers the API handlers with the DefaultServerMux of the http
// package. It calls ListenAndServe but does not process errors returned by it.
// This is because we do not want to throw an error when Babble is used in-mem
// and we want to expose the Babble API on the same endpoint (address:port) as
// the application's API.
func (s *Service) Serve() {
	s.logger.WithField("bind_address", s.bindAddress).Debug("Starting Babble API service")

	// Add handlers to DefaultServerMux
	http.HandleFunc("/stats", s.GetStats)
	http.HandleFunc("/block/{index}", s.GetBlock)
	http.HandleFunc("/graph", s.GetGraph)
	http.HandleFunc("/peers", s.GetPeers)
	http.HandleFunc("/genesispeers", s.GetGenesisPeers)

	// It is possible that another server, started in the same process, is
	// simultaneously using the DefaultServerMux. In which case, the handlers
	// will be accessible from both servers.
	http.ListenAndServe(s.bindAddress, nil)
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
