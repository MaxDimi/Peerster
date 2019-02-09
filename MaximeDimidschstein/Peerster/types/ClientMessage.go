package types

import (
	"github.com/dedis/protobuf"
)

/********************************** CLIENT MESSAGE **********************************/

type ClientMessage struct {
	Contents      string
}

func (msg *ClientMessage) GetContent() string{
	return msg.Contents
}

func (msg *ClientMessage) GetDestination() string {
	return ""
}

func (msg *ClientMessage) String() string{
	if msg==nil {
		return ""
	}
	return "CLIENT MESSAGE " + msg.Contents
}

func (msg *ClientMessage) Display(origin string) string {
	return msg.String()
}

/********************************** FILE LOADING **********************************/

type FileMessage struct {
	Filename string
}

func (msg *FileMessage) GetContent() string{
	return msg.Filename
}

func (msg *FileMessage) GetDestination() string {
	return ""
}

func (msg *FileMessage) String() string{
	return "FILE name " + msg.Filename
}

func (msg *FileMessage) Display(origin string) string {
	return msg.String()
}

func (msg *FileMessage) TreatMsg(g *Gossiper, origin string) {
	newFile := g.addFile(msg.Filename)
	if newFile != nil {
		toSend := &TxPublish{*newFile, TxHopLimit+1}
		toSend.TreatMsg(g, "")
	}

}

func (msg *FileMessage) WrapMsg(g *Gossiper) []byte{
	packetBytes, _ := protobuf.Encode(&ClientPacket{File:msg})
	return packetBytes
}

/********************************** TREATMENT **********************************/

func (msg *ClientMessage) TreatMsg(g *Gossiper, origin string) {
	g.printMsg(msg, "")
	if g.simple {
		toSend := &SimpleMessage{g.name, (*g.address).String(), msg.Contents}

		for _, p := range g.peers {
			if origin != p && p != ""{
				g.sendMessage(p, toSend)
			}
		}
	}
	if !g.simple{
		toSend := &RumorMessage{Origin:g.name, ID:g.id, Text:msg.Contents}
		g.addMessage(toSend)
		g.id+=1
		go g.propagate(toSend, "", false)
	}
}

func (msg *ClientMessage) WrapMsg(g *Gossiper) []byte{
	packetBytes, _ := protobuf.Encode(&ClientPacket{Simple:msg})
	return packetBytes
}