package types

import "fmt"
import "github.com/dedis/protobuf"

type DataReply struct {
	Origin string
	Destination string
	HopLimit uint32
	HashValue []byte
	Data []byte
}

func (msg *DataReply) GetContent() string {
	return fmt.Sprint(msg.Data)
}

func (msg *DataReply) GetDestination() string {
	return msg.Destination
}

func (msg *DataReply) String() string{
	return "REPLY origin " + msg.Origin + " hop-limit " + fmt.Sprint(msg.HopLimit) + " contents " + msg.GetContent()
}

func (msg *DataReply) Display(origin string) string {
	return msg.String()
}


/********************************** TREATMENT **********************************/

func (msg *DataReply) TreatMsg(g *Gossiper, origin string) {
	if msg.Destination == g.name {
		hashRequest, sendNext := g.treatReply(msg)
		if sendNext == nil || sendNext.Code == 0 {
			destination := msg.Origin
			if sendNext!=nil && sendNext.Dest!="" {
				destination = sendNext.Dest
			}
			go g.requestLoop(&DataRequest{g.name, destination, HopLimit, hashRequest[:]})
		} else if sendNext.Code == 1 { // Transfer finished
			g.saveFile(hashRequest)
			fmt.Println("RECONSTRUCTED file " + g.metafiles[hashRequest].Filename)
		}
	} else {
		g.sendMessage("", msg)
	}
}

func (msg *DataReply) WrapMsg(g *Gossiper) []byte{
	msg.HopLimit -= 1
	if msg.HopLimit <= 0 {
		return nil
	}
	packetBytes, _ := protobuf.Encode(&GossipPacket{DataReply:msg})
	return packetBytes
}