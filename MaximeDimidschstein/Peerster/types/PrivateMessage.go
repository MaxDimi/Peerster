package types

import "fmt"
import "github.com/dedis/protobuf"

/********************************** OBJECT **********************************/

type PrivateMessage struct {
	Origin 	string
	ID		uint32
	Text	string
	Destination string
	HopLimit uint32
}

func (msg *PrivateMessage) GetContent() string{
	return msg.Text
}

func (msg *PrivateMessage) GetDestination() string {
	return msg.Destination
}

func (msg *PrivateMessage) String() string{
	if msg.Origin == "" && msg.ID == 0{
		return "CLIENT MESSAGE " + msg.Text
	}
	return "PRIVATE origin " + msg.Origin + " hop-limit " + fmt.Sprint(msg.HopLimit) + " contents " + msg.Text
}

func (msg *PrivateMessage) Display(origin string) string {
	if origin=="" || (msg.Origin == "" && msg.ID == 0) {
		return "CLIENT MESSAGE " + msg.Text
	}
	return "PRIVATE origin " + msg.Origin + " hop-limit " + fmt.Sprint(msg.HopLimit) + " contents " + msg.Text

}

/********************************** TREATMENT **********************************/

func (msg *PrivateMessage) TreatMsg(g *Gossiper, origin string) {
	if origin=="" {
		g.printMsg(msg, msg.Origin)
		msg.Origin = g.name
		g.sendMessage("", msg)
	} else {
		if msg.Destination==g.name {
			g.printMsg(msg, origin)
		} else {
			g.sendMessage("", msg)
		}
	}
}

func (msg *PrivateMessage) WrapMsg(g *Gossiper) []byte{
	var packetBytes []byte
	if g==nil {
		packetBytes, _ = protobuf.Encode(&ClientPacket{Private:msg})
	} else {
		msg.HopLimit -= 1
		if msg.HopLimit <= 0 {
			return nil
		}
		packetBytes, _ = protobuf.Encode(&GossipPacket{Private:msg})
	}
	return packetBytes
}