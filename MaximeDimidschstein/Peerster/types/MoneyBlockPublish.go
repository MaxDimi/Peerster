package types

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"runtime"
	"time"
)
import "github.com/dedis/protobuf"

type MoneyBlock struct {
	PrevHash 		[32]byte
	Nonce			[32]byte
	Transactions 	[]MoneyTxPublish
}

func (b *MoneyBlock) GetPrevHash() [32]byte{
	return b.PrevHash
}

func (b *MoneyBlock) Hash() (out [32]byte) {
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

func (b *MoneyBlock) String() string{
	currentHash := b.Hash()
	str := fmt.Sprintf("%s", hex.EncodeToString(currentHash[:])) + ":" +
			fmt.Sprintf("%s", hex.EncodeToString(b.PrevHash[:])) + ":"

	for i, tx := range b.Transactions {
		str += tx.String()
		if i<len(b.Transactions)-1 {
			str += ", "
		}
	}

	return str
}

func (b *MoneyBlock) addTx(g *Gossiper) {
	rollback := false
	ind := 0
	changes := make(map[string]float64)
	for i, tx := range b.Transactions {
		//g.transfersMap[tx.Identifier()] = &tx
		pubKey, ok := g.pubKeys[tx.Sender]
		//valid := false
		if ok {
			myHash := tx.Hash()
			r, ok_r := new(big.Int).SetString(tx.R, 16)
			s, ok_s := new(big.Int).SetString(tx.S, 16)
			if !ok_r || !ok_s {
				rollback = true
				ind = i
				break;
			}
			valid := ecdsa.Verify(&pubKey, myHash[:], r, s)
			if !valid {
				rollback = true
				ind = i
				break;
			}
		}

		if tx.Condition.Check() {
			g.transfersMap[tx.Identifier()] = &tx
			changes[tx.Sender] -= tx.Amount
			changes[tx.Receiver] += tx.Amount
			_, ok := g.wallets[tx.Sender]
			if !ok {
				g.wallets[tx.Sender] = InitialBalance
			}
			_, ok = g.wallets[tx.Receiver]
			if !ok {
				g.wallets[tx.Receiver] = InitialBalance
			}
			g.wallets[tx.Sender] = g.wallets[tx.Sender] - tx.Amount
			g.wallets[tx.Receiver] = g.wallets[tx.Receiver] + tx.Amount
		} else {
			g.transfersCond[tx.Identifier()] = &tx
		}

		lastID, ok := g.transfers[tx.Sender]
		if g.wallets[tx.Sender] < 0.0 || tx.Amount < 0.0 || (ok && tx.ID!=lastID+1) || (!ok && tx.ID!=1) {
			rollback = true
			ind = i
			break;
		}
		g.transfers[tx.Sender] = tx.ID
	}

	if rollback {
		for node, amount := range changes {
			g.wallets[node] -= amount
		}
		for i, tx := range b.Transactions {
			delete(g.transfersMap, tx.Identifier())
			delete(g.transfersCond, tx.Identifier())

			if i==ind {
				break;
			}
			g.transfers[tx.Sender] -= 1
		}
	} else {
		for _, tx := range b.Transactions {
			for i, txTemp := range g.transfersPending {
				if txTemp.Identifier() == tx.Identifier() {
					g.transfersPending = append(g.transfersPending[:i], g.transfersPending[i+1:]...)
				}
			}
		}
	}
}

func (b *MoneyBlock) removeTx(g *Gossiper) {
	for _, tx := range b.Transactions {
		delete(g.transfersMap, tx.Identifier())
		_, ok := g.transfersCond[tx.Identifier()]
		if !ok {
			g.wallets[tx.Sender] = g.wallets[tx.Sender] + tx.Amount
			g.wallets[tx.Receiver] = g.wallets[tx.Receiver] - tx.Amount
		} else {
			delete(g.transfersCond, tx.Identifier())
		}
		g.transfers[tx.Sender] -= 1
		g.transfersPending = append(g.transfersPending, tx)
	}
}

/**************************** BLOCKPUBLISH ****************************/

type MoneyBlockPublish struct {
	Block 		MoneyBlock
	HopLimit 	uint32
}

func (msg *MoneyBlockPublish) GetContent() string {
	str := ""
	for _, tx := range msg.Block.Transactions {
		str += tx.GetContent()
	}
	return str
}

func (msg *MoneyBlockPublish) GetDestination() string {
	return ""
}

func (msg *MoneyBlockPublish) String() string{
	return msg.GetContent()
}

func (msg *MoneyBlockPublish) Display(origin string) string {
	return msg.GetContent()
}

/********************************** TREATMENT **********************************/

func (msg *MoneyBlockPublish) TreatMsg(g *Gossiper, origin string) {
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

func (msg *MoneyBlockPublish) WrapMsg(g *Gossiper) []byte{
	packetBytes, _ := protobuf.Encode(&GossipPacket{MoneyBlockPublish:msg})
	return packetBytes
}

func (msg *MoneyBlockPublish) sendWithDelay(g *Gossiper, d time.Duration) {
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