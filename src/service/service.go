package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"

	hg "github.com/mosaicnetworks/babble/src/hashgraph"

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
	http.HandleFunc("/blocks/", s.makeHandler(s.GetBlocks))
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

// GetBlocks ...
func (s *Service) GetBlocks(w http.ResponseWriter, r *http.Request) {
	maxLimit := 5

	qi := r.URL.Path[len("/blocks/"):]

	var blocks []*hg.Block

	ql := r.URL.Query().Get("limit")
	if ql == "" {
		ql = string(maxLimit)
	}

	stringLastBlockIndex := s.node.GetStats()["last_block_index"]
	if stringLastBlockIndex == "-1" {
		s.logger.WithError(errors.New("No blocks found")).Errorf("No blocks found")
		http.Error(w, "No blocks found", http.StatusInternalServerError)
		return
	}

	lastBlockIndex, err := strconv.Atoi(stringLastBlockIndex)
	if err != nil {
		s.logger.WithError(err).Errorf("Converting last block index to int")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	i, err := strconv.Atoi(qi)
	if err != nil {
		s.logger.WithError(err).Errorf("Converting block index to int")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	l, err := strconv.Atoi(ql)
	if err != nil {
		s.logger.WithError(err).Errorf("Converting blocks limit to int")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if l > maxLimit {
		l = maxLimit
	}

	if l > lastBlockIndex {
		l = lastBlockIndex
	}

	for i <= l {
		block, err := s.node.GetBlock(i)
		if err != nil {
			s.logger.WithError(err).Errorf("Retrieving block %d", i)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		blocks = append(blocks, block)
		i++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blocks)
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
