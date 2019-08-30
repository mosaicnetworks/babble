package service

import (
	"encoding/json"
	"fmt"
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

// GetBlocks returns an array of blocks starting with blocks/[index]?limit=x and finishing
// at x blocks later
func (s *Service) GetBlocks(w http.ResponseWriter, r *http.Request) {
	// blocks slice
	var blocks []*hg.Block

	// max limit on blocks set back
	maxLimit := 5

	// check last block index and make sure a block exists
	sLastBlockIndex := s.node.GetStats()["last_block_index"]
	if sLastBlockIndex == "-1" {
		s.logger.Errorf("No blocks found")
		http.Error(w, "No blocks found", http.StatusInternalServerError)
		return
	}

	// convert to int
	lastBlockIndex, err := strconv.Atoi(sLastBlockIndex)
	if err != nil {
		s.logger.WithError(err).Errorf("Converting last block index to int")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// parse starting block index
	qi := r.URL.Path[len("/blocks/"):]
	i, err := strconv.Atoi(qi)
	if err != nil {
		s.logger.WithError(err).Errorf("Converting block index to int")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if i > lastBlockIndex {
		s.logger.Errorf("Requested index larger than last block index")
		http.Error(w, "Requested starting index larger than last block index", http.StatusInternalServerError)
		return
	}

	// get max limit, if empty set to maxlimit
	ql := r.URL.Query().Get("limit")
	if ql == "" {
		ql = strconv.Itoa(maxLimit)
	}

	// parse to int
	l, err := strconv.Atoi(ql)
	if err != nil {
		s.logger.WithError(err).Errorf("Converting blocks limit to int")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// make sure requested limit does not exceed max
	if l > maxLimit {
		l = maxLimit
	}

	// make limit does not exceed last block index
	if i+l > lastBlockIndex {
		l = lastBlockIndex - i
	}

	// get blocks
	for c := 0; c <= l; {
		fmt.Println("Fetching block: ", i+c)
		fmt.Println("Limit: ", l)

		block, err := s.node.GetBlock(i + c)
		if err != nil {
			s.logger.WithError(err).Errorf("Retrieving block %d", i+c)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		blocks = append(blocks, block)
		c++
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
