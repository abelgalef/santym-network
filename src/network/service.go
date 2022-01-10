package network

import (
	"context"
	"log"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
)

type NetworkService struct {
	ctx           context.Context
	host          host.Host
	pubsub        *pubsub.PubSub
	subbedTopic   *pubsub.Topic
	mDnsService   *mDns
	discoveryChan <-chan peer.AddrInfo
}

var bootstrapList [3]string = [3]string{
	"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	"/dnsaddr/bootstrap.libp2p.io/ipfs/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	"/dnsaddr/bootstrap.libp2p.io/ipfs/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
}

func NewNetworkService(ctx context.Context) *NetworkService {
	return &NetworkService{ctx: ctx}
}

func (ns *NetworkService) Init() bool {
	log.Println("Initializing network service...")
	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		log.Fatal(err)
	}

	ns.host = host
	log.Printf("Host ID: %s\nHost Addr:%s\n", host.ID(), host.Addrs())

	ps, err := pubsub.NewGossipSub(ns.ctx, host)
	if err != nil {
		panic(err)
	}
	ns.pubsub = ps

	kademliaDHT, err := dht.New(ns.ctx, host)
	if err != nil {
		panic(err)
	}

	if err = kademliaDHT.Bootstrap(ns.ctx); err != nil {
		panic(err)
	}
	log.Println("DHT Bootstrapped")

	ns.Bootstrap()

	log.Println("Announcing to the dht network")
	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	discovery.Advertise(ns.ctx, routingDiscovery, "santym-network")

	log.Println("Looking for peers through route discovery and mDns")
	ns.discoveryChan, err = routingDiscovery.FindPeers(ns.ctx, "santym-network")
	if err != nil {
		panic(err)
	}

	ns.mDnsService = &mDns{host: host, ctx: ns.ctx}
	ser := mdns.NewMdnsService(host, "santym-network", ns.mDnsService)

	if err := ser.Start(); err != nil {
		log.Panic(err)
	}

	ns.Subscribe(ns.ctx, ps, "santym-network")

	go ns.DiscoverPeers()

	return true
}

func (ns *NetworkService) Bootstrap() {
	var wg sync.WaitGroup
	for _, peerAddr := range bootstrapList {
		addr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			panic(err)
		}
		peerinfo, _ := peer.AddrInfoFromP2pAddr(addr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := ns.host.Connect(ns.ctx, *peerinfo); err != nil {
				log.Println("Failed to connect to peer:", err)
			} else {
				log.Println("Connection established with bootstrap node:", *peerinfo)
			}
		}()
	}
	wg.Wait()
}

func (ns *NetworkService) DiscoverPeers() {
	for {
		select {
		case peer := <-ns.mDnsService.peerChan:
			go ns.ConnectToPeer(peer)
		case peer := <-ns.discoveryChan:
			go ns.ConnectToPeer(peer)
		}
	}
}

func (ns *NetworkService) ConnectToPeer(peer peer.AddrInfo) {
	if peer.ID == ns.host.ID() {
		return
	}
	log.Println("New peer discovered:", peer.ID.Pretty())
	if err := ns.host.Connect(ns.ctx, peer); err != nil {
		log.Println("Failed to connect to peer:", peer.ID.Pretty(), err)
	} else {
		log.Println("Connected to peer:", peer.ID.Pretty())
	}
}

func (ns *NetworkService) Subscribe(ctx context.Context, ps *pubsub.PubSub, topic string) {
	psTopic, err := ps.Join(topic)
	if err != nil {
		log.Fatal(err)
	}

	sub, err := psTopic.Subscribe()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Subscribed to topic: ", topic)

	ns.subbedTopic = psTopic

	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				log.Fatal(err)
			}
			if msg.GetFrom() == ns.host.ID() {
				continue
			}

			//TODO: Handle message with channels
			log.Println(string(msg.GetData()))
		}
	}()

	// For testing purposes only
	// go func() {
	// 	for {
	// 		time.Sleep(time.Second * 2)
	// 		psTopic.Publish(ctx, []byte(time.Now().String()))
	// 	}
	// }()
}
