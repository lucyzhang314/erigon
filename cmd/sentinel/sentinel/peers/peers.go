/*
   Copyright 2022 Erigon-Lightclient contributors
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

package peers

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/ledgerwatch/log/v3"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	maxBadPeers     = 50
	DefaultMaxPeers = 33
	MaxBadResponses = 10
)

// Time to wait before asking the same peer again.
const reqRetryTime = 300 * time.Millisecond

// Record Peer data.
type Peer struct {
	lastQueried time.Time
	busy        bool
}

type Peers struct {
	badPeers  *lru.Cache // Keep track of bad peers
	penalties *lru.Cache // Keep track on how many penalties a peer accumulated, PeerId => penalties
	host      host.Host

	peers map[peer.ID]Peer

	mu sync.Mutex
}

func New(host host.Host) *Peers {
	badPeers, err := lru.New(maxBadPeers)
	if err != nil {
		panic(err)
	}

	penalties, err := lru.New(maxBadPeers)
	if err != nil {
		panic(err)
	}
	return &Peers{
		badPeers:  badPeers,
		penalties: penalties,
		host:      host,
		peers:     make(map[peer.ID]Peer),
	}
}

func (p *Peers) IsBadPeer(pid peer.ID) bool {
	return p.badPeers.Contains(pid)
}

func (p *Peers) Penalize(pid peer.ID) {
	penaltyInterface, has := p.penalties.Get(pid)
	if !has {
		p.penalties.Add(pid, 1)
		return
	}
	penalties := penaltyInterface.(int) + 1

	p.penalties.Add(pid, penalties)
	// Drop peer and delete the map element.
	if penalties > MaxBadResponses {
		p.DisconnectPeer(pid)
		p.penalties.Remove(pid)
	}
}

func (p *Peers) Forgive(pid peer.ID) {
	penaltyInterface, has := p.penalties.Get(pid)
	if !has {
		return
	}
	penalties := penaltyInterface.(int) - 1
	if penalties < 0 {
		penalties = 0
	}
	p.penalties.Add(pid, penalties)
}

func (p *Peers) BanBadPeer(pid peer.ID) {
	p.DisconnectPeer(pid)
	p.badPeers.Add(pid, []byte{0})
	log.Debug("[Sentinel Peers] bad peers has been banned", "peer-id", pid)
}

func (p *Peers) DisconnectPeer(pid peer.ID) {
	log.Trace("[Sentinel Peers] disconnecting from peer", "peer-id", pid)
	p.host.Peerstore().RemovePeer(pid)
	p.host.Network().ClosePeer(pid)
}

// PeerDoRequest signals that the peer is doing a request.
func (p *Peers) PeerDoRequest(pid peer.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers[pid] = Peer{
		lastQueried: time.Now(),
		busy:        true,
	}
}

// IsPeerAvaiable returns if the peer is in cooldown or is being requested already .
func (p *Peers) IsPeerAvaiable(pid peer.ID) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	peer, ok := p.peers[pid]
	return !ok || (!peer.busy && time.Since(peer.lastQueried) >= reqRetryTime)
}

// PeerFinishRequest signals that the peer is done doing a request.
func (p *Peers) PeerFinishRequest(pid peer.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers[pid] = Peer{
		busy:        false,
		lastQueried: time.Now(),
	}
}
