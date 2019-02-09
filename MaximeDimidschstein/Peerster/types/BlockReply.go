package types

import (
	"fmt"
	"github.com/dedis/protobuf"
)

type BlockReply struct {
	Length			int
	FileBlockList 	[]Block
	MoneyBlockList 	[]MoneyBlock
}

func (msg *BlockReply) GetContent() string {
	str := ""
	for _, block := range msg.FileBlockList {
		str += block.String() + " "
	}
	return str
}

func (msg *BlockReply) GetDestination() string {
	return ""
}

func (msg *BlockReply) String() string{
	return " contents " + msg.GetContent()
}

func (msg *BlockReply) Display(origin string) string {
	return "BLOCKREPLY origin " + origin + msg.String()
}


/********************************** TREATMENT **********************************/

func (msg *BlockReply) TreatMsg(g *Gossiper, origin string) {
	if msg.Length > g.lastBlock.Length && msg.Length==len(msg.FileBlockList) + len(msg.MoneyBlockList) {
		var newBlocks []Block_t
		i := 1 // First block = genesis block
		j := 0
		var previousHash [32]byte
		var currentBlock Block_t
		for i < len(msg.FileBlockList) || j < len(msg.MoneyBlockList) {
			if i < len(msg.FileBlockList) && msg.FileBlockList[i].PrevHash == previousHash {
				currentBlock = &msg.FileBlockList[i]
				i++
			} else if j < len(msg.MoneyBlockList) && msg.MoneyBlockList[j].PrevHash == previousHash {
				currentBlock = &msg.MoneyBlockList[j]
				j++
			} else {
				fmt.Println("\033[91m INVALID BLOCKCHAIN RECEIVED \033[0m" + fmt.Sprint(i) + " " + fmt.Sprint(j))
				break;
			}

			previousHash = currentBlock.Hash()
			if !BlockHashValid(previousHash) {
				return
			}
			fmt.Println(msg.Display(origin))

			newBlocks = append(newBlocks, currentBlock)
			_, ok := g.blockChain[previousHash]
			if !ok || i>=len(msg.FileBlockList)-1 || j>=len(msg.MoneyBlockList)-1 {
				NewBlockchainNode(newBlocks[i+j-2], nil).addNode(g)
			}
		}
	}
}

func (msg *BlockReply) WrapMsg(g *Gossiper) []byte{
	packetBytes, _ := protobuf.Encode(&GossipPacket{BlockReply:msg})
	return packetBytes
}