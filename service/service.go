package service

import (
	"encoding/json"
	"net/http"

	"github.com/babbleio/babble/node"
	"github.com/Sirupsen/logrus"
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

	http.HandleFunc("/Stats", service.GetStats)

	return &service
}

func (s *Service) Serve() {
	s.logger.WithField("bind_address", s.bindAddress).Debug("Service serving")
	err := http.ListenAndServe(s.bindAddress, nil)
	if err != nil {
		s.logger.WithField("error", err).Error("Service failed")
	}
}

func (s *Service) GetStats(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("Stats request")
	stats := s.node.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
