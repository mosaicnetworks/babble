package service

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mosaicnetworks/babble/node"
	"github.com/sirupsen/logrus"
)

type Service struct {
	bindAddress string
	node        *node.Node
	logger      *logrus.Logger
}

func NewService(bindAddress string, node *node.Node, logger *logrus.Logger) *Service {
	service := Service{
		bindAddress: bindAddress,
		node:        node,
		logger:      logger,
	}

	return &service
}

func (s *Service) Serve() {
	s.logger.WithField("bind_address", s.bindAddress).Debug("Service serving")
	http.HandleFunc("/stats", s.GetStats)
	http.HandleFunc("/block/", s.GetBlock)
	err := http.ListenAndServe(s.bindAddress, nil)
	if err != nil {
		s.logger.WithField("error", err).Error("Service failed")
	}
}

func (s *Service) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := s.node.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

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
