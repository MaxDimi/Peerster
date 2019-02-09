package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"runtime"
	"time"
)
import "github.com/dedis/protobuf"


/******************************** BLOCK *******************************/

type Block struct {
	PrevHash 		[32]byte
	Nonce			[32]byte
	Transactions 	[]TxPublish
}

func (b *Block) GetPrevHash() [32]byte{
	return b.PrevHash
}

func (b *Block) Hash() (out [32]byte) {
	h := sha256.New()
	h.Write(b.PrevHash[:])
	h.Write(b.Nonce[:])
	binary.Write(h, binary.LittleEndian, uint32(len(b.Transactions)))
	for _, t := range b.Transactions {
		th := t.Hash()
		h.Write(th[:])
	}
	copy(out[:], h.Sum(nil))
	return
}

func (b *Block) String() string{
	currentHash := b.Hash()
	str := fmt.Sprintf("%s", hex.EncodeToString(currentHash[:])) + ":" +
		fmt.Sprintf("%s", hex.EncodeToString(b.PrevHash[:])) + ":"

	for i, tx := range b.Transactions {
		str += tx.File.Name
		if i<len(b.Transactions)-1 {
			str += ","
		}
	}

	return str
}

func (b *Block) addTx(g *Gossiper) {
	for _, tx := range b.Transactions {
		g.txMap[tx.File.Name] = &tx
		for i, txTemp := range g.txPending {
			if txTemp.File.Name==tx.File.Name {
				g.txPending = append(g.txPending[:i], g.txPending[i+1:]...)
			}
		}
	}
}

func (b *Block) removeTx(g *Gossiper) {
	for _, tx := range b.Transactions {
		delete(g.txMap, tx.File.Name)
		g.txPending = append(g.txPending, tx)
	}
}

/**************************** BLOCKPUBLISH ****************************/

type BlockPublish struct {
	Block 		Block
	HopLimit 	uint32
}

func (msg *BlockPublish) GetContent() string {
	str := ""
	for _, tx := range msg.Block.Transactions {
		str += tx.GetContent()
	}
	return str
}

func (msg *BlockPublish) GetDestination() string {
	return ""
}

func (msg *BlockPublish) String() string{
	return msg.GetContent()
}

func (msg *BlockPublish) Display(origin string) string {
	return msg.GetContent()
}

/********************************** TREATMENT **********************************/

func (msg *BlockPublish) TreatMsg(g *Gossiper, origin string) {
	currentHash := msg.Block.Hash()
	if !BlockHashValid(currentHash) {
		return
	}

	_, ok := g.blockChain[currentHash]
	if ok {
		return
	}

	newNode := NewBlockchainNode(&msg.Block, nil)
	newNode.addNode(g)

	if !newNode.hasNext() {
		g.sendMessage(origin, &BlockRequest{g.lastBlock.Length})
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

func (msg *BlockPublish) WrapMsg(g *Gossiper) []byte{
	packetBytes, _ := protobuf.Encode(&GossipPacket{BlockPublish:msg})
	return packetBytes
}

func (msg *BlockPublish) sendWithDelay(g *Gossiper, d time.Duration) {
	currentHash := msg.Block.Hash()
	if !BlockHashValid(currentHash) {
		return
	}

	_, ok := g.blockChain[currentHash]
	if ok {
		return
	}

	NewBlockchainNode(&msg.Block, nil).addNode(g)

	end := false
	go Timeout(d, &end)
	for !end {
		runtime.Gosched()
	}

	msg.HopLimit -= 1

	if msg.HopLimit > 0 {
		for _, peer := range g.peers {
			g.sendMessage(peer, msg)
		}
	}
}