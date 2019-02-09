package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)
import "github.com/dedis/protobuf"


type File struct {
	Name string
	Size int64
	MetafileHash []byte
}

func (f *File) String() string {
	return fmt.Sprintf("%s:%s", hex.EncodeToString(f.MetafileHash), f.Name)
}

type TxPublish struct {
	File 		File
	HopLimit 	uint32
}

func (t *TxPublish) Hash() (out [32]byte) {
	h := sha256.New()
	binary.Write(h, binary.LittleEndian,
		uint32(len(t.File.Name)))
	h.Write([]byte(t.File.Name))
	h.Write(t.File.MetafileHash)
	copy(out[:], h.Sum(nil))
	return
}


func (msg *TxPublish) GetContent() string{
	return msg.File.String()
}

func (msg *TxPublish) GetDestination() string {
	return ""
}

func (msg *TxPublish) String() string{
	return msg.GetContent()
}

func (msg *TxPublish) Display(origin string) string {
	return msg.GetContent()
}

/********************************** TREATMENT **********************************/

func (msg *TxPublish) TreatMsg(g *Gossiper, origin string) {
	_, ok := g.txMap[msg.File.Name]
	if ok {
		return
	}

	g.txMap[msg.File.Name] = msg
	g.txPending = append(g.txPending, *msg)

	msg.HopLimit -= 1

	if msg.HopLimit > 0 {
		for _, peer := range g.peers {
			if peer != origin {
				g.sendMessage(peer, msg)
			}
		}
	}
}

func (msg *TxPublish) WrapMsg(g *Gossiper) []byte{
	packetBytes, _ := protobuf.Encode(&GossipPacket{TxPublish:msg})
	return packetBytes
}