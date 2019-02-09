package types

import (
	"encoding/hex"
	"fmt"
)

type BlockchainNode struct {
	Hash	[32]byte
	Block	Block_t
	Next	*BlockchainNode
	Length	int
	IsFirst bool
}

func NewBlockchainNode(block Block_t, next *BlockchainNode) *BlockchainNode {
	b := &BlockchainNode{
		Hash: block.Hash(),
		Block: block,
		Next: nil,
		Length: 1,
		IsFirst: true,
	}

	if next!=nil {
		b.setNext(next)
	}

	return b
}

func (b *BlockchainNode) isNext(next *BlockchainNode) bool {
	return next!=nil && b.Block.GetPrevHash()==next.Hash
}

func (b *BlockchainNode) setNext(next *BlockchainNode) bool {
	if next==nil {
		b.Next = nil
		b.Length = 1
	}

	if !b.isNext(next) {
		return false
	}
	b.Next = next
	next.IsFirst = false
	b.Length = next.Length+1
	return true
}

func (b *BlockchainNode) hasNext() bool {
	return b.Next != nil
}

func (b *BlockchainNode) computeLength() int {
	count := 1

	current := b
	for current.hasNext() {
		current = current.Next
		count += 1
	}

	b.Length = count
	return count
}

func (b *BlockchainNode) ancestors() []*BlockchainNode {
	ancestorsList := make([]*BlockchainNode, 0, 5)
	current := b
	ancestorsList = append(ancestorsList, current)

	for current.hasNext() {
		current = current.Next
		ancestorsList = append(ancestorsList, current)
	}

	return ancestorsList
}

func (b *BlockchainNode) commonAncestor(node *BlockchainNode) *BlockchainNode {
	myAncestors := b.ancestors()
	otherAncestors := node.ancestors()

	for i := len(myAncestors)-1 ; i>=0 ; i-- {
		for j := len(otherAncestors)-1 ; j>=0 ; j-- {
			if myAncestors[i]==otherAncestors[j] {
				return myAncestors[i]
			}
		}
	}

	return nil
}

func (b *BlockchainNode) String() string{
	return b.Block.String()
}

func (b *BlockchainNode) ChainString() string{
	str := "CHAIN"

	currentBlock := b
	var zeroHash [32]byte

	for currentBlock != nil && currentBlock.Hash!=zeroHash{
		str += " "
		str += currentBlock.String()
		currentBlock = currentBlock.Next
	}
	return str
}


func (b *BlockchainNode) addNode(g *Gossiper) {
	for _, node := range g.blockChain {
		if b.isNext(node) {
			if node.IsFirst {
				delete(g.chainHeads, node.Hash)
				g.chainHeads[b.Hash] = b
			}
			b.setNext(node)
			if b.Length > g.lastBlock.Length {
				if g.lastBlock == node {
					b.addTx(g)
				} else {
					b.rollbackTx(g)
				}
				g.lastBlock = b
				g.printBlockChain()
				g.printWallets()
			} else {
				fmt.Println("FORK-SHORTER " + fmt.Sprintf("%s", hex.EncodeToString(b.Hash[:])))
			}
		}
	}
	g.blockChain[b.Hash] = b
}

func (b *BlockchainNode) addTx(g *Gossiper) {
	b.Block.addTx(g)
}

func (b *BlockchainNode) rollbackTx(g *Gossiper) {
	currentNode := g.lastBlock
	ancestor := b.commonAncestor(currentNode)

	count := 0

	for currentNode!=ancestor {
		currentNode.Block.removeTx(g)
		currentNode = currentNode.Next
		count += 1
	}

	fmt.Println("FORK-LONGER rewind " + fmt.Sprint(count) + " blocks")

	currentNode = b
	for currentNode!=ancestor {
		currentNode.addTx(g)
		currentNode = currentNode.Next
	}
}