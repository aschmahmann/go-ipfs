package provider

import (
	"context"
	"gx/ipfs/QmXjKkjMDTtXAiLBwstVexofB8LeruZmE2eBd85GwGFFLA/go-ipfs-blockstore"
	"gx/ipfs/QmcxZXMqFu4vjLQRfG2tAcg6DPQNurgZ2SQ5iQVk6dXQjn/go-libp2p-routing"
	"time"
)

var (
	reprovideOutgoingWorkerLimit = 8
)

type Reprovider struct {
	ctx context.Context
	queue *Queue
	tracker *Tracker
	tick time.Duration
	blockstore blockstore.Blockstore
	contentRouting routing.ContentRouting
	trigger chan struct{}
}

// Reprovider periodically re-announces the cids that have been provided. These
// reprovides can be run on an interval and/or manually. Reprovider also untracks
// cids that are no longer in the blockstore.
func NewReprovider(ctx context.Context, queue *Queue, tracker *Tracker, tick time.Duration, blockstore blockstore.Blockstore, contentRouting routing.ContentRouting) *Reprovider {
	return &Reprovider{
		ctx: ctx,
		queue: queue,
		tracker: tracker,
		tick: tick,
		blockstore: blockstore,
		contentRouting: contentRouting,
		trigger: make(chan struct{}),
	}
}

// Begin listening for triggers and reprovide whatever is
// in the reprovider queue.
func (rp *Reprovider) Run() {
	go rp.handleTriggers()
	go rp.handleAnnouncements()
}

// Add all the cids in the tracker to the reprovide queue
func (rp *Reprovider) Reprovide() error {
	cids, err := rp.tracker.Tracking(rp.ctx)
	if err != nil {
		log.Warningf("error obtaining tracking information: %s", err)
		return err
	}
	for c := range cids {
		if err := rp.queue.Enqueue(c); err != nil {
			log.Warningf("unable to enqueue cid: %s, %s", c, err)
			continue
		}
	}
	return nil
}

// Trigger causes a reprovide
func (rp *Reprovider) Trigger(ctx context.Context) error {
	select {
	case <-rp.ctx.Done():
		return rp.ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	case rp.trigger <- struct{}{}:
	}
	return nil
}

func (rp *Reprovider) handleTriggers() {
	// dont reprovide immediately.
	// may have just started the daemon and shutting it down immediately.
	// probability( up another minute | uptime ) increases with uptime.
	after := time.After(time.Minute)
	for {
		if rp.tick == 0 {
			after = nil
		}

		select {
		case <-rp.ctx.Done():
			return
		case <-rp.trigger:
		case <-after:
		}

		err := rp.Reprovide()
		if err != nil {
			log.Debug(err)
		}

		after = time.After(rp.tick)
	}
}

func (rp *Reprovider) handleAnnouncements() {
	for workers := 0; workers < reprovideOutgoingWorkerLimit; workers++ {
		go func() {
			for {
				select {
				case <-rp.ctx.Done():
					return
				case entry := <-rp.queue.Dequeue():
					if err := doProvide(rp.ctx, rp.tracker, rp.blockstore, rp.contentRouting, entry.cid); err != nil {
						log.Warningf("Unable to reprovide entry: %s, %s", entry.cid, err)
					}
					if err := entry.Complete(); err != nil {
						log.Warningf("Unable to complete queue entry when reproviding: %s, %s", entry.cid, err)
					}
				}
			}
		}()
	}
}