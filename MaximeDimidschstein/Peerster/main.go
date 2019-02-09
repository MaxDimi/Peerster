package main

import (
	"flag"
	. "github.com/MaximeDimidschstein/Peerster/types"
	"net"
)

func main() {
	UIPort := flag.String("UIPort", "8080", "port for the UI client")
	gossipAddr := flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper")
	name := flag.String("name", "", "name of the gossiper")
	peersStr := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	rtimer := flag.Int("rtimer", 0, "route rumors sending period in seconds, 0 to disable")
	simple := flag.Bool("simple", false, "run gossiper in simple broadcast mode")

	flag.Parse()

	UIAddr, _ := net.ResolveUDPAddr("udp4", ":"+*UIPort)

	g := NewGossiper(*gossipAddr, *name, *peersStr, *rtimer, *simple)

	go g.ClientLoop(UIAddr)
	g.GossipLoop()
}
