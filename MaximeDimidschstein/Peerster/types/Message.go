package types

import (
	"fmt"
)

/********************************** PACKET **********************************/

type GossipPacket struct {
	Simple				*SimpleMessage
	Rumor 				*RumorMessage
	Status 				*StatusPacket
	Private		 		*PrivateMessage
	DataRequest 		*DataRequest
	DataReply 			*DataReply
	SearchRequest	 	*SearchRequest
	SearchReply 		*SearchReply
	TxPublish			*TxPublish
	BlockPublish 		*BlockPublish
	BlockRequest 		*BlockRequest
	BlockReply 			*BlockReply
	MoneyTxPublish		*MoneyTxPublish
	MoneyBlockPublish 	*MoneyBlockPublish
	PubKeyPublish 		*PubKeyPublish
}

type ClientPacket struct {
	Simple			*ClientMessage
	Private 		*PrivateMessage
	File			*FileMessage
	DataRequest 	*DataRequest
	SearchRequest 	*SearchRequest
	MoneyTxPublish	*MoneyTxPublish
}

/********************************** MESSAGE *********************************/

type Message interface {
	GetContent() string
	GetDestination() string
	String()	 string
	Display(origin string)	 string
	TreatMsg(g *Gossiper, origin string)
	WrapMsg(g *Gossiper) []byte
}

/******************************* METAFILE *****************************/

type Metafile struct {
	Filename string
	Filesize int
	ChunkHashes []byte
	MetaHash [32]byte
}

func (mf *Metafile) String() string{
	return "FILE name " + mf.Filename + " size " + fmt.Sprint(mf.Filesize) + " MetaHash " + fmt.Sprint(mf.MetaHash)
}

/******************************* BLOCKS *******************************/

type Block_t interface {
	String()	string
	Hash()		[32]byte
	addTx(g *Gossiper)
	removeTx(g *Gossiper)
	GetPrevHash()	[32]byte
}