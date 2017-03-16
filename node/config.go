/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package node

import (
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/arrivets/babble/common"
)

type Config struct {
	HeartbeatTimeout time.Duration
	Logger           *logrus.Logger
}

func NewConfig(heartbeat time.Duration, logger *logrus.Logger) *Config {
	return &Config{
		HeartbeatTimeout: heartbeat,
		Logger:           logger,
	}
}

func DefaultConfig() *Config {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	return &Config{
		HeartbeatTimeout: 1000 * time.Millisecond,
		Logger:           logger,
	}
}

func TestConfig(t *testing.T) *Config {
	config := DefaultConfig()
	config.Logger = common.NewTestLogger(t)
	return config
}
