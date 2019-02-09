package types

import "github.com/dedis/protobuf"

/********************************** OBJECT **********************************/

type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

func (msg *SimpleMessage) GetContent() string {
	return msg.Contents
}

func (msg *SimpleMessage) GetDestination() string {
	return ""
}

func (msg *SimpleMessage) String() string {
	if msg==nil {
		return ""
	}
	if msg.OriginalName == "" && msg.RelayPeerAddr == ""{
		return "CLIENT MESSAGE " + msg.Contents
	}
	return "SIMPLE MESSAGE origin " + msg.OriginalName + " from " + msg.RelayPeerAddr + " contents " + msg.Contents
}

func (msg *SimpleMessage) Display(origin string) string {
	return msg.String()
}


/********************************** TREATMENT **********************************/

func (msg *SimpleMessage) TreatMsg(g *Gossiper, origin string) {
	g.printMsg(msg, origin)

	msg.RelayPeerAddr = (*g.address).String()

	for _, p := range g.peers {
		if origin != p && p != ""{
			g.sendMessage(p, msg)
		}
	}
}

func (msg *SimpleMessage) WrapMsg(g *Gossiper) []byte{
	packetBytes, _ := protobuf.Encode(&GossipPacket{Simple:msg})
	return packetBytes
}