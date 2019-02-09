package types

import "fmt"
import "github.com/dedis/protobuf"

/********************************** OBJECT **********************************/

type DataRequest struct {
	Origin string
	Destination string
	HopLimit uint32
	HashValue []byte
}

func (msg *DataRequest) GetContent() string{
	return fmt.Sprint(msg.HashValue)
}

func (msg *DataRequest) GetDestination() string {
	return msg.Destination
}

func (msg *DataRequest) String() string{
	return "REQUEST origin " + msg.Origin + " hop-limit " + fmt.Sprint(msg.HopLimit) + " contents " + msg.GetContent()
}

func (msg *DataRequest) Display(origin string) string {
	return msg.String()
}

/********************************** TREATMENT **********************************/

func (msg *DataRequest) TreatMsg(g *Gossiper, origin string) {

	if origin=="" {
		msg.Origin = g.name

		if msg.Destination=="" {
			var metaHash [32]byte
			copy(metaHash[:], msg.HashValue)
			chunkMap, ok := g.chunksFound[metaHash]
			if !ok {
				return
			}
			for _, k := range chunkMap {
				msg.Destination=k
				break
			}
		}

		g.sendMessage("", msg)
	} else {
		if msg.Destination==g.name {
			data := g.treatRequest(msg)
			if data != nil {
				target := msg.Origin
				g.sendMessage("", &DataReply{g.name, target, HopLimit, msg.HashValue, data})
			}
		} else {
			g.sendMessage("", msg)
		}
	}
}

func (msg *DataRequest) WrapMsg(g *Gossiper) []byte{
	var packetBytes []byte
	if g==nil {
		packetBytes, _ = protobuf.Encode(&ClientPacket{DataRequest:msg})
	} else {
		msg.HopLimit -= 1
		if msg.HopLimit <= 0 {
			return nil
		}
		packetBytes, _ = protobuf.Encode(&GossipPacket{DataRequest:msg})
	}
	return packetBytes
}