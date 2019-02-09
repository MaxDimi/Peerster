package types

import (
	"encoding/hex"
	"fmt"
	"strings"
)
import "github.com/dedis/protobuf"


/******************************** SearchResult *******************************/

type SearchResult struct {
	FileName     string
	MetafileHash []byte
	ChunkMap     []uint64
	ChunkCount   uint64
}

/********************************** OBJECT **********************************/

type SearchReply struct {
	Origin string
	Destination string
	HopLimit uint32
	Results []*SearchResult
}

func (msg *SearchReply) GetContent() string{
	return ""
}

func (msg *SearchReply) GetDestination() string {
	return msg.Destination
}

func (msg *SearchReply) String() string{
	return "REQUEST origin " + msg.Origin + " hop-limit " + fmt.Sprint(msg.HopLimit) + " contents " + msg.GetContent()
}

func (msg *SearchReply) Display(origin string) string {
	toPrint := ""
	for _, result := range msg.Results {
		toPrint += "FOUND match " + result.FileName + " at " + msg.Origin + " metafile=" +
			fmt.Sprintf("%s", hex.EncodeToString(result.MetafileHash) + " chunks=")
		for i, chunk := range result.ChunkMap {
			toPrint += fmt.Sprint(chunk)
			if i < len(result.ChunkMap)-1 {
				toPrint +=","
			}
		}
	}
	return toPrint
}

/********************************** TREATMENT **********************************/

func (msg *SearchReply) TreatMsg(g *Gossiper, origin string) {
	if msg.Destination == g.name {
		fmt.Println(msg.Display(origin))

		for request := range g.mySearchReq {
			if msg.corresponds(request) {
				for _, result := range msg.Results {
					toStore := &SearchReply{Origin:msg.Origin, Destination:g.name, HopLimit:0, Results:append(make([]*SearchResult, 0, 1), result)}
					g.mySearchReq[request][result.FileName] = append(g.mySearchReq[request][result.FileName], toStore)

					for _, chunk := range result.ChunkMap {
						var tempHash [32]byte
						copy(tempHash[:], result.MetafileHash)
						_, ok := g.chunksFound[tempHash]
						if !ok {
							g.chunksFound[tempHash] = make(map[uint64] string)
						}
						_, ok = g.chunksFound[tempHash][chunk]
						if !ok {
							g.chunksFound[tempHash][chunk] = msg.Origin
						}
					}
				}
			}
		}

	} else {
		g.sendMessage("", msg)
	}
}

func (msg *SearchReply) WrapMsg(g *Gossiper) []byte{
	var packetBytes []byte
	if g==nil {
	} else {
		msg.HopLimit -= 1
		if msg.HopLimit <= 0 {
			return nil
		}
		packetBytes, _ = protobuf.Encode(&GossipPacket{SearchReply:msg})
	}
	return packetBytes
}

func (msg *SearchReply) corresponds(request *SearchRequest) bool {
	for _, result := range msg.Results {
		found := false
		for i:=0 ; i<len(request.Keywords) && !found ; i++ {
			if strings.Contains(result.FileName, request.Keywords[i]) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}