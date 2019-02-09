package types

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)
import "github.com/dedis/protobuf"

type MoneyTxPublish struct {
	Sender		string
	Receiver	string
	Amount 		float64
	HopLimit 	uint32
	ID			uint32
	Condition	TxCondition
	R			string
	S			string
}

func (t *MoneyTxPublish) Hash() (out [32]byte) {
	h := sha256.New()
	binary.Write(h, binary.LittleEndian,
		t.Amount)
	binary.Write(h, binary.LittleEndian,
		t.ID)
	h.Write([]byte(t.Sender))
	h.Write([]byte(t.Receiver))
	copy(out[:], h.Sum(nil))
	return
}

func (t *MoneyTxPublish) Identifier() string {
	return t.Sender + "_" + fmt.Sprint(t.ID)
}

func (msg *MoneyTxPublish) GetContent() string{
	return fmt.Sprint(msg.Amount)
}

func (msg *MoneyTxPublish) GetDestination() string {
	return ""
}

func (msg *MoneyTxPublish) String() string{
	return "MONEY-TRANSFER from " + msg.Sender + " to " + msg.Receiver + " amount " + msg.GetContent()
}

func (msg *MoneyTxPublish) Display(origin string) string {
	return msg.GetContent()
}

/********************************** TREATMENT **********************************/

func (msg *MoneyTxPublish) TreatMsg(g *Gossiper, origin string) {
	if origin=="" {
		msg.Sender = g.name
		msg.ID = g.transferID
		myHash := msg.Hash()
		r, s, err := ecdsa.Sign(rand.Reader, g.privKey, myHash[:])
		if err != nil {
			return
		}
		msg.R = r.Text(16)
		msg.S = s.Text(16)
	}

	lastID, ok := g.transfers[msg.Sender]
	if (ok && msg.ID<=lastID) || (!ok && msg.ID!=1) || msg.Amount<=0.0 { // Not in order or invalid
		return
	}

	balance, ok := g.wallets[msg.Sender]
	if !ok {
		g.wallets[msg.Sender] = InitialBalance
		balance = InitialBalance
	}

	_, ok = g.wallets[msg.Receiver]
	if !ok {
		g.wallets[msg.Receiver] = InitialBalance
	}

	for _, tx := range g.transfersPending {
		if msg.Sender == tx.Sender {
			balance -= tx.Amount
		} else if msg.Sender == tx.Receiver {
			balance += tx.Amount
		}
	}

	if balance<msg.Amount { // Insufficient balance
		return
	}

	g.transfersPending = append(g.transfersPending, *msg)

	if origin=="" {
		g.transferID++
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

func (msg *MoneyTxPublish) WrapMsg(g *Gossiper) []byte{
	var packetBytes []byte
	if g==nil {
		packetBytes, _ = protobuf.Encode(&ClientPacket{MoneyTxPublish:msg})
	} else {
		packetBytes, _ = protobuf.Encode(&GossipPacket{MoneyTxPublish:msg})
	}
	return packetBytes
}