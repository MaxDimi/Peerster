package types

import (
	"fmt"
	"github.com/dedis/protobuf"
)

type BlockRequest struct {
	Length		int
}

func (msg *BlockRequest) GetContent() string {
	return fmt.Sprint(msg.Length)
}

func (msg *BlockRequest) GetDestination() string {
	return ""
}

func (msg *BlockRequest) String() string {
	return " length " + msg.GetContent()
}

func (msg *BlockRequest) Display(origin string) string {
	return "BLOCKREQUEST origin " + origin +  msg.GetContent()
}

/********************************** TREATMENT **********************************/

func (msg *BlockRequest) TreatMsg(g *Gossiper, origin string) {
	var FileBlockList []Block
	var MoneyBlockList []MoneyBlock
	if g.lastBlock.Length > msg.Length {
		for _, node := range g.lastBlock.ancestors() {
			fBlock, ok := node.Block.(*Block)
			if ok {
				FileBlockList = append([]Block{*fBlock}, FileBlockList...)
			}
			mBlock, ok := node.Block.(*MoneyBlock)
			if ok {
				MoneyBlockList = append([]MoneyBlock{*mBlock}, MoneyBlockList...)
			}
		}
	}
	g.sendMessage(origin, &BlockReply{g.lastBlock.Length, FileBlockList, MoneyBlockList})
}

func (msg *BlockRequest) WrapMsg(g *Gossiper) []byte{
	packetBytes, _ := protobuf.Encode(&GossipPacket{BlockRequest:msg})
	return packetBytes
}
