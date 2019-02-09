package types

import "fmt"
import "github.com/dedis/protobuf"

type RumorMessage struct {
	Origin 	string
	ID 		uint32
	Text	string
}

func (msg *RumorMessage) GetContent() string{
	return msg.Text
}

func (msg *RumorMessage) GetDestination() string {
	return ""
}

func (msg *RumorMessage) String() string{
	if msg.Origin == "" && msg.ID == 0{
		return "CLIENT MESSAGE " + msg.Text
	}
	return "ID " + fmt.Sprint(msg.ID) + " contents " + msg.Text
}

func (msg *RumorMessage) Display(origin string) string {
	if origin=="" || (msg.Origin == "" && msg.ID == 0) {
		return "CLIENT MESSAGE " + msg.Text
	}
	return "RUMOR origin " + msg.Origin + " from " + origin + " ID " + fmt.Sprint(msg.ID) + " contents " + msg.Text
}

/********************************** TREATMENT **********************************/

func (msg *RumorMessage) TreatMsg(g *Gossiper, origin string) {
	spread := false
	if len(msg.Text) > 0 {
		g.printMsg(msg, origin)
	}
	index := IndexOf(msg.Origin, g.want)
	if index < 0 && msg.ID == 1 {
		g.addMessage(msg)
		spread = true
	} else if index >= 0 && g.want[index].NextID == msg.ID {
		g.addMessage(msg)
		spread = true
	}
	g.sendAck(origin)
	if spread {
		g.updateRoutingTable(msg.Origin, origin)
		go g.propagate(msg, origin, false)
	}
}

func (msg *RumorMessage) WrapMsg(g *Gossiper) []byte{
	packetBytes, _ := protobuf.Encode(&GossipPacket{Rumor:msg})
	return packetBytes
}