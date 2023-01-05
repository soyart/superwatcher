package poller

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/artnoi43/superwatcher"
	"github.com/artnoi43/superwatcher/pkg/logger/debugger"
)

// poller implements superwatcher.WatcherPoller.
// It filters event logs and detecting chain reorg, i.e. it produces superwatcher.PollResult for the emitter.
// poller behavior can be changed on-the-fly using methods defined in this file.
type poller struct {
	sync.RWMutex

	addresses []common.Address
	topics    [][]common.Hash

	lastRecordedBlock uint64 // For clearing tracker if SetDoReorg(false) is called
	filterRange       uint64
	client            superwatcher.EthClient
	doReorg           bool
	doHeader          bool

	tracker  *blockTracker
	debugger *debugger.Debugger
}

func New(
	addresses []common.Address,
	topics [][]common.Hash,
	doReorg bool,
	doHeader bool,
	filterRange uint64,
	client superwatcher.EthClient,
	logLevel uint8,
) superwatcher.EmitterPoller {
	var tracker *blockTracker
	if doReorg {
		tracker = newTracker("poller", logLevel)
	}

	return &poller{
		addresses:   addresses,
		topics:      topics,
		filterRange: filterRange,
		client:      client,
		doReorg:     doReorg,
		doHeader:    doHeader,
		tracker:     tracker,
		debugger:    debugger.NewDebugger("poller", logLevel),
	}
}

// SetDoReorg changes poller behavior regarding processing chain reorg.
// If set to false, it clears all tracker data and deletes the tracker with nil value, and change p.deReorg to false.
// If set to true, it
func (p *poller) SetDoReorg(doReorg bool) {
	p.Lock()
	defer p.Unlock()

	switch p.doReorg {
	case true:
		if doReorg {
			return
		}

		p.debugger.Debug(1, "SetDoReorg(false) called - clearing tracker and deleting tracker")
		p.tracker.clearUntil(p.lastRecordedBlock)
		p.tracker = nil

	case false:
		if !doReorg {
			p.debugger.Debug(1, "SetDoReorg(true) called, but p.doReorg is already true - returning")
			return
		}

		if p.tracker == nil {
			p.debugger.Debug(1, "SetDoReorg(true) called - creating new tracker")
			p.tracker = newTracker("poller", p.debugger.Level)
		} else {
			p.debugger.Debug(1, "SetDoReorg(true) called but tracker is not nil - reusing tracker")
		}
	}

	p.doReorg = doReorg
}

func (p *poller) DoReorg() bool {
	p.RLock()
	defer p.RUnlock()

	return p.doReorg
}

func (p *poller) SetDoHeader(doHeader bool) {
	p.Lock()
	defer p.Unlock()

	p.doHeader = doHeader
}

func (p *poller) DoHeader() bool {
	p.RLock()
	defer p.RUnlock()

	return p.doHeader
}

func (p *poller) Addresses() []common.Address {
	p.RLock()
	defer p.RUnlock()

	return p.addresses
}

func (p *poller) Topics() [][]common.Hash {
	p.RLock()
	defer p.RUnlock()

	return p.topics
}

func (p *poller) AddAddresses(addresses ...common.Address) {
	p.Lock()
	defer p.Unlock()

	if len(p.addresses) == 0 {
		p.addresses = addresses
		return
	}

	p.addresses = append(p.addresses, addresses...)
}

func (p *poller) AddTopics(topics ...[]common.Hash) {
	p.Lock()
	defer p.Unlock()

	if len(p.topics) == 0 {
		p.topics = topics
		return
	}

	p.topics = append(p.topics, topics...)
}

func (p *poller) SetAddresses(addresses []common.Address) {
	p.Lock()
	defer p.Unlock()

	p.addresses = addresses
}

func (p *poller) SetTopics(topics [][]common.Hash) {
	p.Lock()
	defer p.Unlock()

	p.topics = topics
}
