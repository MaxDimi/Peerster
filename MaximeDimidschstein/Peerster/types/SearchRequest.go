package types

import (
	"fmt"
	"math/rand"
)
import "github.com/dedis/protobuf"

/********************************** OBJECT **********************************/

type SearchRequest struct {
	Origin string
	Budget uint64
	Keywords []string
}

func (msg *SearchRequest) GetContent() string{
	return ""
}

func (msg *SearchRequest) GetDestination() string {
	return ""
}

func (msg *SearchRequest) String() string{
	return "REQUEST origin " + msg.Origin + " budget " + fmt.Sprint(msg.Budget) + " contents " + msg.GetContent()
}

func (msg *SearchRequest) Display(origin string) string {
	return msg.String()
}

func (msg *SearchRequest) Equals(other *SearchRequest) bool{
	if msg.Origin!=other.Origin || len(msg.Keywords)!=len(other.Keywords) {
		return false
	}

	keywords := other.Keywords

	for _, k := range msg.Keywords {
		index := IndexOfStr(k, keywords)
		if index<0 {
			return false
		}
		keywords = RemoveFrom(keywords, index)
	}

	return true
}

/********************************** TREATMENT **********************************/

func (msg *SearchRequest) TreatMsg(g *Gossiper, origin string) {
	if origin=="" {
		msg.Origin = g.name
		g.mySearchReq[msg] = make(map[string][]*SearchReply)
		if msg.Budget>0 {
			msg.Budget+=1
		} else {
			go g.searchLoop(msg)
			return
		}
	}

	if g.duplicateRequests(msg) {
		return
	}

	if msg.Origin != g.name {
		var result []*SearchResult
		for _, k := range msg.Keywords {
			temp := g.searchFile(k)
			if temp != nil && len(temp)>0 {
				result = Merge(result, temp)
			}
		}
		if result != nil && len(result)>0 {
			target := msg.Origin
			g.sendMessage("", &SearchReply{g.name, target, HopLimit, result})
		}
	}

	msg.Budget -= 1
	if msg.Budget>0 && ((len(g.peers)>0 && origin=="") || (len(g.peers)>1 && origin!="")) {
		msg.ForwardSearchRequest(g, origin)
	}
}

func (msg *SearchRequest) WrapMsg(g *Gossiper) []byte{
	var packetBytes []byte
	if g==nil {
		packetBytes, _ = protobuf.Encode(&ClientPacket{SearchRequest:msg})
	} else {
		packetBytes, _ = protobuf.Encode(&GossipPacket{SearchRequest:msg})
	}
	return packetBytes
}

func (msg *SearchRequest) ForwardSearchRequest(g *Gossiper, origin string) {
	budget := msg.Budget

	peers := make([]string, len(g.peers))
	copy(peers, g.peers)

	if origin!="" {
		for i, p := range peers {
			if p==origin {
				peers = append(peers[:i], peers[i+1:]...)
				break
			}
		}
	}

	lenPeers := uint64(len(peers))
	if budget<=0 {
		return
	} else if budget<lenPeers {
		msg.Budget=1
		for i, index := range rand.Perm(len(peers)) {
			if uint64(i)==budget {
				return
			}
			g.sendMessage(peers[index], msg)
		}
	} else {
		rest := budget % lenPeers
		msg.Budget = budget/lenPeers + 1
		index := uint64(0)
		for ; index<rest ; index++ {
			g.sendMessage(peers[index], msg)
		}
		msg.Budget -= 1
		for ; index<lenPeers ; index++ {
			g.sendMessage(peers[index], msg)
		}
	}
}