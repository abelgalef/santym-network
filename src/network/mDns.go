package network

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type mDns struct {
	peerChan chan peer.AddrInfo
	host     host.Host
	ctx      context.Context
}

func (m *mDns) HandlePeerFound(pi peer.AddrInfo) {
	m.peerChan <- pi
}
