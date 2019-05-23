package libp2p

import (
	"github.com/ipfs/go-ipns"
	host "github.com/libp2p/go-libp2p-host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/fx"

	"github.com/ipfs/go-ipfs/core/node/helpers"
)

func FloodSub(pubsubOptions ...pubsub.Option) interface{} {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle, host host.Host) (service *pubsub.PubSub, err error) {
		return pubsub.NewFloodSub(helpers.LifecycleCtx(mctx, lc), host, pubsubOptions...)
	}
}

func GossipSub(pubsubOptions ...pubsub.Option) interface{} {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle, host host.Host) (service *pubsub.PubSub, err error) {
		return pubsub.NewGossipSub(helpers.LifecycleCtx(mctx, lc), host, pubsubOptions...)
	}
}

func LWWGossipSub(pubsubOptions ...pubsub.Option) interface{} {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle, host host.Host, routing BaseIpfsRouting) (service *pubsub.PubSub, err error) {
		ipnsCache := pubsub.NewLWWMessageCache(ipns.Validator{})
		return pubsub.NewGossipSyncLWW(helpers.LifecycleCtx(mctx, lc), host, ipnsCache, "ipnsps/0.0.1")
	}
}
