package service

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	hg "github.com/mosaicnetworks/babble/src/hashgraph"

	"github.com/mosaicnetworks/babble/src/node"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/sirupsen/logrus"
)

// MAXBLOCKS is the maximum number of blocks returned by the /blocks/ endpoint
const MAXBLOCKS = 50

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
	http.HandleFunc("/validators/", s.makeHandler(s.GetValidatorSet))
	http.HandleFunc("/history", s.makeHandler(s.GetAllValidatorSets))
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

// GetBlock - DEPRECATED used /blocks/ instead
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

/*
GetBlocks will fetch an array of blocks starting at {startIndex} and finishing
{counts<=MAXBLOCKS} blocks later. If no count param is provided it will just
return the index requested rather than listing blocks.

GET /blocks/{startIndex}?count={x}
example: /blocks/0?count=50
returns: JSON []hashgraph.Block
*/
func (s *Service) GetBlocks(w http.ResponseWriter, r *http.Request) {
	// parse starting block index
	qs := r.URL.Path[len("/blocks/"):]
	requestStart, err := strconv.Atoi(qs)
	if err != nil {
		s.logger.WithError(err).Errorf("Parsing block index parameter")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lastBlockIndex := s.node.GetLastBlockIndex()

	if requestStart > lastBlockIndex {
		http.Error(w, "Requested starting index larger than last block index", http.StatusInternalServerError)
		return
	}

	count := 1

	// get max limit, if empty just send requested index
	qc := r.URL.Query().Get("count")
	if qc != "" {
		// parse to int
		count, err = strconv.Atoi(qc)
		if err != nil {
			s.logger.WithError(err).Errorf("Converting blocks count to int")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// make sure requested limit does not exceed max
		if count > MAXBLOCKS {
			count = MAXBLOCKS
		}

		// make limit does not exceed last block index
		if requestStart+count-1 > lastBlockIndex {
			count = lastBlockIndex - requestStart + 1
		}
	}

	// blocks slice
	var blocks []*hg.Block

	// get blocks
	for c := 0; c < count; c++ {
		block, err := s.node.GetBlock(requestStart + c)
		if err != nil {
			s.logger.WithError(err).Errorf("Retrieving block %d", requestStart+c)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		blocks = append(blocks, block)
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

/*
GetPeers returns the node's current peers, which is not necessarily equivalent
to the current validator-set.

GET /peers
returns: JSON []peers.Peer
*/
func (s *Service) GetPeers(w http.ResponseWriter, r *http.Request) {
	returnPeerSet(w, r, s.node.GetPeers())
}

/*
GetGenesisPeers returns the genesis validator-set

Get /genesispeers
returns: JSON []peers.Peer
*/
func (s *Service) GetGenesisPeers(w http.ResponseWriter, r *http.Request) {
	ps, err := s.node.GetValidatorSet(0)
	if err != nil {
		s.logger.WithError(err).Errorf("Fetching genesis validator-set")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	returnPeerSet(w, r, ps)
}

/*
GetValidatorSet returns the validator-set associated to a specific hashgraph
round. If no round is specified, it returns the current validator-set.

Get /validators/{round}
returns: JSON []peers.Peer
*/
func (s *Service) GetValidatorSet(w http.ResponseWriter, r *http.Request) {
	round := s.node.GetLastConsensusRoundIndex()

	param := r.URL.Path[len("/validators/"):]
	if param != "" {
		r, err := strconv.Atoi(param)
		if err != nil {
			s.logger.WithError(err).Errorf("Parsing round parameter %s", param)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		round = r
	}

	validators, err := s.node.GetValidatorSet(round)
	if err != nil {
		s.logger.WithError(err).Errorf("Fetching round %d validators", round)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	returnPeerSet(w, r, validators)
}

/*
GetAllValidatorSets returns the entire map of round to validator-sets which
represents the history of the validator-set from the inception of the network.

Get /history
returns: JSON map[int][]peers.Peer
*/
func (s *Service) GetAllValidatorSets(w http.ResponseWriter, r *http.Request) {
	allPeerSets, err := s.node.GetAllValidatorSets()
	if err != nil {
		s.logger.WithError(err).Errorf("Fetching validator-sets")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allPeerSets)
}

func returnPeerSet(w http.ResponseWriter, r *http.Request, peers []*peers.Peer) {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)

	encoder.Encode(peers)
}
