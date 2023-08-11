package mev

import (
	"context"
	"sync"
	"time"

	"github.com/palomachain/pigeon/config"
	"github.com/palomachain/pigeon/internal/mev/blxr"
	log "github.com/sirupsen/logrus"
)

type Client interface {
	IsHealthy() bool
	IsChainRegistered(string) bool
	GetHealthprobeInterval() time.Duration
	KeepAlive(context.Context, sync.Locker) error
	RegisterChain(string)
}

func New(cfg *config.Root) Client {
	if len(cfg.BloxrouteAuthorizationHeader) < 1 {
		log.Info("BLXR Auth header not found. No MEV relayer support.")
		return nil
	}

	// At the moment, only BLXR is supported
	c := blxr.New(cfg.BloxrouteAuthorizationHeader)
	for k, v := range cfg.EVM {
		if v.EVMSpecificClientConfig.BloxrouteIntegrationEnabled {
			log.Infof("Adding BLXR relayer support for chain %s", k)
			c.RegisterChain(k)
		}
	}

	return c
}