package types

import "fmt"
import "github.com/dedis/protobuf"

type StatusPacket struct {
	Want []PeerStatus
}

func (sp StatusPacket) indexOf(identifier string) int {
	for i, b := range sp.Want {
		if b.Identifier == identifier {
			return i
		}
	}
	return -1
}

func (s *StatusPacket) GetContent() string{
	return s.String()
}

func (msg *StatusPacket) GetDestination() string {
	return ""
}

func (s *StatusPacket) String() string {
	str := ""
	for i,p := range s.Want {
		str += p.String()
		if i<len(s.Want)-1 {
			str += " "
		}
	}
	return str
}

func (s *StatusPacket) Display(origin string) string {

	str := "STATUS from " + origin
	statStr := s.String()
	if statStr != ""{
		str += " " + statStr
	}
	return str
}


type PeerStatus struct {
	Identifier 	string
	NextID		uint32
}

func (p *PeerStatus) String() string{
	return "peer " + p.Identifier + " nextID " + fmt.Sprint(p.NextID)
}


/********************************** TREATMENT **********************************/

func (msg *StatusPacket) TreatMsg(g *Gossiper, origin string) {
	g.printMsg(msg, origin)
	g.pWMutex.Lock()
	g.peerWant[origin] = msg.Want
	g.pWMutex.Unlock()
	toSend := g.getMessageToSend(msg.Want)
	if toSend != nil {
		g.sendMessage(origin, toSend)
	} else if g.askMessage(msg.Want) {
		g.sendAck(origin)
	} else {
		fmt.Println("IN SYNC WITH " + origin)
	}
}

func (msg *StatusPacket) WrapMsg(g *Gossiper) []byte{
	packetBytes, _ := protobuf.Encode(&GossipPacket{Status:msg})
	return packetBytes
}