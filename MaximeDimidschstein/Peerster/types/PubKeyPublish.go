package types

import (
	"crypto/ecdsa"
	"fmt"
	//"math/rand"
)
import "github.com/dedis/protobuf"

type PubKeyPublish struct {
	Origin		string
	PubKey		ecdsa.PublicKey
	HopLimit	uint32
}

func (msg *PubKeyPublish) GetContent() string{
	return fmt.Sprint(msg.PubKey)
}

func (msg *PubKeyPublish) GetDestination() string {
	return ""
}

func (msg *PubKeyPublish) String() string{
	return "PUBLIC-KEY from " + msg.Origin + " key " + fmt.Sprint(msg.PubKey)
}

func (msg *PubKeyPublish) Display(origin string) string {
	return msg.GetContent()
}

/********************************** TREATMENT **********************************/

func (msg *PubKeyPublish) TreatMsg(g *Gossiper, origin string) {
	if origin!="" {
		g.pubKeys[msg.Origin] = msg.PubKey
	}

	msg.HopLimit -= 1

	if msg.HopLimit > 0 {
		for _, peer := range g.peers {
			if peer != origin {
				g.sendMessage(peer, msg)
			}
		}
	}
}

func (msg *PubKeyPublish) WrapMsg(g *Gossiper) []byte{
	packetBytes, _ := protobuf.Encode(&GossipPacket{PubKeyPublish:msg})
	return packetBytes
}