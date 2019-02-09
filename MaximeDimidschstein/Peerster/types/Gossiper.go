package types

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/dedis/protobuf"
	"log"
	"math/rand"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)


/*****************************************************************************
******************************** GOSSIPER type *******************************
*****************************************************************************/

type Gossiper struct {
	address 			*net.UDPAddr
	conn 				*net.UDPConn
	name 				string
	peers 				[]string
	simple 				bool
	id 					uint32
	want				[]PeerStatus
	messages			map[string][]*RumorMessage
	peerWant			map[string][]PeerStatus
	routingTable 		map[string]string
	metafiles 			map[[32]byte]Metafile
	chunks 				map[[32]byte][]byte
	currentReq 			map[[32]byte][32]byte // Pending requests chunk->metafile
	cRMutex				sync.Mutex
	nextReq 			map[[32]byte]int // Pending requests metafile->nextchunk
	pWMutex 			sync.Mutex
	pMutex				sync.Mutex
	msgMutex			sync.Mutex
	wMutex				sync.Mutex
	recReqMutex			sync.Mutex
	rtimer				int
	recentReq			map[string][]struct{req *SearchRequest; end *bool}
	mySearchReq 		map[*SearchRequest]map[string][]*SearchReply
	chunksFound 		map[[32]byte]map[uint64]string // hash -> chunknumber -> dest
	txMap				map[string]*TxPublish
	txPending			[]TxPublish
	txMining			[]TxPublish
	blockChain			map[[32]byte]*BlockchainNode//Block
	lastBlock			*BlockchainNode
	chainHeads			map[[32]byte]*BlockchainNode // Blocks that are at the head of their own chains
	blockTimer			time.Duration
	wallet				float64 // Current own balance
	wallets				map[string]float64 // Balance of each known node in the network
	transfers			map[string]uint32 // Last transfer ID received from each peer
	transferID			uint32
	transfersMap		map[string]*MoneyTxPublish
	transfersCond		map[string]*MoneyTxPublish // Transfers which condition is not true yet
	transfersPending	[]MoneyTxPublish
	transfersMining		[]MoneyTxPublish
	privKey				*ecdsa.PrivateKey
	pubKey				ecdsa.PublicKey
	pubKeys				map[string]ecdsa.PublicKey
}

func NewGossiper(address, name string, peersStr string, rtimer int, simple bool) *Gossiper {
	udpAddr, err := net.ResolveUDPAddr("udp4", address)
	if err!=nil {
		return nil
	}
	udpConn, err := net.ListenUDP("udp4", udpAddr)
	if err!=nil {
		return nil
	}

	peersList := strings.Split(peersStr, ",")
	if peersList==nil || len(peersList)<=0 || (len(peersList)==1 && peersList[0]=="") {
		peersList = make([]string, 0, 5)
	}

	lastBlock := NewBlockchainNode(&Block{}, nil)
	var zeroHash [32]byte
	lastBlock.Hash = zeroHash
	lastBlock.Length = 1

	blockChain := make(map[[32]byte]*BlockchainNode)
	blockChain[lastBlock.Hash] = lastBlock

	chainHeads := make(map[[32]byte]*BlockchainNode)
	chainHeads[lastBlock.Hash] = lastBlock

	wallets := make(map[string]float64)
	wallets[name] = InitialBalance

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
	if err != nil {
		return nil
	}


	return &Gossiper{
		address: 		udpAddr,
		conn:    		udpConn,
		name:    		name,
		peers:	 		peersList,
		simple:	 		simple,
		id:		 		1,
		peerWant:		make(map[string][]PeerStatus),
		messages:		make(map[string][]*RumorMessage),
		routingTable:	make(map[string]string),
		metafiles:		make(map[[32]byte]Metafile),
		chunks: 		make(map[[32]byte][]byte),
		currentReq: 	make(map[[32]byte][32]byte),
		nextReq: 		make(map[[32]byte]int),
		rtimer:			rtimer,
		recentReq: 		make(map[string][]struct{req *SearchRequest; end *bool}),
		mySearchReq:	make(map[*SearchRequest]map[string][]*SearchReply),
		chunksFound:	make(map[[32]byte]map[uint64]string),
		txMap:			make(map[string]*TxPublish),
		blockChain:		blockChain,
		chainHeads:		chainHeads,
		lastBlock:		lastBlock,
		blockTimer:		initialBlockTimeout,
		wallets:		wallets,
		transfers:		make(map[string]uint32),
		transfersMap:	make(map[string]*MoneyTxPublish),
		transfersCond:	make(map[string]*MoneyTxPublish),
		transferID:		1,
		privKey:		privKey,
		pubKey:			privKey.PublicKey,
		pubKeys:		make(map[string]ecdsa.PublicKey),
	}
}

/*****************************************************************************
****************************** GOSSIP FUNCTIONS ******************************
*****************************************************************************/

/*********************************** LOOP ***********************************/

func (g *Gossiper) GossipLoop() {


	if !g.simple {
		go g.antiEntropy(AntiEntropyTimeout)
		go g.miningMain()
		go g.transfersMiningMain()
		go g.checkCond()
	}

	if g.rtimer>0 {
		go g.routeRumor()
	}

	for {

		buf := make([]byte, ChunkSize+1000)
		_, source, _ := g.conn.ReadFromUDP(buf)
		origin := source.String()

		g.pMutex.Lock()
		if !IsIn(origin, g.peers) && origin != g.address.String() && origin != "" {
			g.peers = append(g.peers, origin)
			g.cleanPeers()
		}
		g.pMutex.Unlock()

		pkt := GossipPacket{}
		protobuf.Decode(buf, &pkt)

		if g.simple && pkt.Simple != nil {
			pkt.Simple.TreatMsg(g, origin)
		} else if pkt.Rumor != nil {
			pkt.Rumor.TreatMsg(g, origin)
		} else if pkt.Status != nil {
			pkt.Status.TreatMsg(g, origin)
		} else if pkt.Private != nil {
			pkt.Private.TreatMsg(g, origin)
		} else if pkt.DataRequest!=nil {
			pkt.DataRequest.TreatMsg(g, origin)
		} else if pkt.DataReply!=nil {
			pkt.DataReply.TreatMsg(g, origin)
		} else if pkt.SearchRequest!=nil {
			pkt.SearchRequest.TreatMsg(g, origin)
		} else if pkt.SearchReply!=nil {
			pkt.SearchReply.TreatMsg(g, origin)
		} else if pkt.TxPublish!=nil {
			pkt.TxPublish.TreatMsg(g, origin)
		} else if pkt.BlockPublish!=nil {
			pkt.BlockPublish.TreatMsg(g, origin)
		} else if pkt.BlockRequest!=nil {
			pkt.BlockRequest.TreatMsg(g, origin)
		} else if pkt.BlockReply!=nil {
			pkt.BlockReply.TreatMsg(g, origin)
		} else if pkt.MoneyTxPublish!=nil {
			pkt.MoneyTxPublish.TreatMsg(g, origin)
		} else if pkt.MoneyBlockPublish!=nil {
			pkt.MoneyBlockPublish.TreatMsg(g, origin)
		} else if pkt.PubKeyPublish!=nil {
			pkt.PubKeyPublish.TreatMsg(g, origin)
		}

	}
}


/********************************* ROUTING ********************************/

func (g *Gossiper) updateRoutingTable(origin, address string) {
	g.routingTable[origin] = address
	fmt.Println("DSDV " + origin + " " + address)
}

func (g *Gossiper) routeRumor() {

	for {
		if len(g.peers) <= 0 {
			end := false
			timeout := 5 * time.Second
			go Timeout(timeout, &end)
			for !end {
				runtime.Gosched()
			}
		} else {
			g.sendRouteRumor()

			end := false
			timeout, _ := time.ParseDuration(fmt.Sprint(g.rtimer) + "s")

			go Timeout(timeout, &end)

			for !end {
				runtime.Gosched()
			}
		}
	}
}

func (g *Gossiper) sendRouteRumor() {
	toSend := &RumorMessage{Origin:g.name, ID:g.id, Text:""}
	g.addMessage(toSend)
	g.id+=1
	g.propagate(toSend, "", false)
}


/********************************** SENDING *********************************/

func (g *Gossiper) sendAck(origin string){
	g.sendMessage(origin, &StatusPacket{g.want})
}

func (g *Gossiper) propagateRecursive(msg *RumorMessage, except string, coin bool){
	peersToSend := make([]string, len(g.peers))
	var peer string
	copy(peersToSend, g.peers)

	if except!="" {
		peersToSend = RemoveFromList(except, peersToSend, 0)
	}

	if len(g.peers)<=0 || len(peersToSend) <= 0 {
		return
	}

	i := rand.Int() % len(peersToSend)
	peer = peersToSend[i]

	if coin {
		fmt.Println("FLIPPED COIN sending rumor to " + peer)
	}
	fmt.Println("MONGERING with " + peer)

	g.sendMessage(peer, msg)

	end := false
	go Timeout(RumorTimeout, &end)

	acked := false
	for !end && !acked {
		g.pWMutex.Lock()
		pWant, ok := g.peerWant[peer]
		if ok {
			i := IndexOf(msg.Origin, pWant)
			if i >= 0 && pWant[i].NextID>msg.ID {
				acked = true
			}
		}
		g.pWMutex.Unlock()
	}

	if len(peersToSend)>1 && rand.Int() % 2 == 1 {
		g.propagateRecursive(msg, peer, true)
	}
}

func (g *Gossiper) propagate(msg *RumorMessage, except string, coin bool){
	peersToSend := make([]string, len(g.peers))
	coinFlip := 1

	for len(peersToSend)>1 && coinFlip == 1 {
		var peer string
		copy(peersToSend, g.peers)

		if except!="" {
			peersToSend = RemoveFromList(except, peersToSend, 0)
		}

		if len(g.peers)<=0 || len(peersToSend) <= 0 {
			return
		}

		i := rand.Int() % len(peersToSend)
		peer = peersToSend[i]

		if coin {
			fmt.Println("FLIPPED COIN sending rumor to " + peer)
		}
		fmt.Println("MONGERING with " + peer)

		g.sendMessage(peer, msg)

		end := false
		go Timeout(RumorTimeout, &end)

		acked := false
		for !end && !acked {
			g.pWMutex.Lock()
			pWant, ok := g.peerWant[peer]
			if ok {
				i := IndexOf(msg.Origin, pWant)
				if i >= 0 && pWant[i].NextID>msg.ID {
					acked = true
				}
			}
			g.pWMutex.Unlock()
		}

		except = peer
		coinFlip = rand.Int() % 2
		if coinFlip==1 {
			coin = true
		} else {
			coin = false
		}
	}
}

func (g *Gossiper) antiEntropy(d time.Duration) {
	for {
		if len(g.peers) <= 0 {
			end := false
			timeout := 5 * time.Second
			go Timeout(timeout, &end)
			for !end {
				runtime.Gosched()
			}
		} else {
			end := false
			go Timeout(d, &end)
			for !end {
				runtime.Gosched()
			}
			peer := g.peers[rand.Int()%len(g.peers)]
			g.sendAck(peer)
		}
	}
}

func (g *Gossiper) transmitPubKey(k crypto.PublicKey) {
	for {
		if len(g.peers) <= 0 {
			end := false
			timeout := 10 * time.Second
			go Timeout(timeout, &end)
			for !end {
				runtime.Gosched()
			}
		} else {
			pkPublish := &PubKeyPublish{g.name, g.pubKey, BlockHopLimit}
			pkPublish.TreatMsg(g, "")
			end := false
			go Timeout(30 * time.Second, &end)
			for !end {
				runtime.Gosched()
			}
		}
	}
}

func (g *Gossiper) sendMessage(address string, msg Message) {
	var rAddr *net.UDPAddr
	var err error

	if address!="" {
		rAddr, err = net.ResolveUDPAddr("udp", address)

		if err!=nil {
			fmt.Println(err)
		}
	} else {
		dest, ok := g.routingTable[msg.GetDestination()]
		if !ok {
			fmt.Println("PRIVATE Unknown destination!")
			return
		}

		rAddr, err = net.ResolveUDPAddr("udp", dest)
		if err!=nil {
			fmt.Println(err)
		}
	}

	packetBytes := msg.WrapMsg(g)
	g.conn.WriteToUDP(packetBytes, rAddr)
}


/*****************************************************************************
****************************** CLIENT FUNCTIONS ******************************
*****************************************************************************/

/*********************************** LOOP ***********************************/

func (g *Gossiper) ClientLoop(UIAddr *net.UDPAddr) {
	udpConn, err := net.ListenUDP("udp4", UIAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer udpConn.Close()
	for {
		buf := make([]byte, ChunkSize+1000)

		udpConn.ReadFromUDP(buf)
		pkt := ClientPacket{}
		protobuf.Decode(buf, &pkt)

		if pkt.Simple != nil {
			pkt.Simple.TreatMsg(g, "")
		} else if pkt.Private != nil {
			pkt.Private.TreatMsg(g, "")
		} else if pkt.File != nil {
			pkt.File.TreatMsg(g, "")
		} else if pkt.DataRequest != nil {
			pkt.DataRequest.TreatMsg(g, "")
		} else if pkt.SearchRequest != nil {
			pkt.SearchRequest.TreatMsg(g, "")
		} else if pkt.MoneyTxPublish != nil {
			pkt.MoneyTxPublish.TreatMsg(g, "")
		}
	}
}


/*****************************************************************************
****************************** MESSAGE TREATMENT *****************************
*****************************************************************************/

func (g *Gossiper) addMessage(msg *RumorMessage) {
	g.msgMutex.Lock()
	source, ok := g.messages[msg.Origin]
	if ok {
		g.messages[msg.Origin] = append(source, msg)
		index := IndexOf(msg.Origin, g.want)
		g.wMutex.Lock()
		g.want[index].NextID+=1
		g.wMutex.Unlock()
	} else {
		g.messages[msg.Origin] = append(g.messages[msg.Origin], msg)
		g.wMutex.Lock()
		g.want = append(g.want, PeerStatus{Identifier:msg.Origin, NextID:msg.ID+1})
		g.wMutex.Unlock()
	}
	g.msgMutex.Unlock()
}

func (g *Gossiper) getMessageToSend(want []PeerStatus) *RumorMessage{

	for _, st := range g.want {
		i := IndexOf(st.Identifier, want)
		if i<0 && st.NextID>1 {
			g.msgMutex.Lock()
			defer g.msgMutex.Unlock()
			return g.messages[st.Identifier][0]
		} else if want[i].NextID < st.NextID && int(want[i].NextID) <= len(g.messages[st.Identifier]) {
			g.msgMutex.Lock()
			defer g.msgMutex.Unlock()
			return g.messages[st.Identifier][want[i].NextID-1]
		}
	}
	return nil
}

func (g *Gossiper) askMessage(want []PeerStatus) bool{
	for _, st := range want {
		i := IndexOf(st.Identifier, g.want)
		if i<0 || g.want[i].NextID < st.NextID {
			return true
		}
	}
	return false
}


/*****************************************************************************
******************************* FILE FUNCTIONS *******************************
*****************************************************************************/

func (g *Gossiper) addFile(filename string) *File {
	f, _ := os.Open(SharedDir + "/" + filename)
	defer f.Close()
	chunk := make([]byte, ChunkSize)
	n := ChunkSize
	var chunkHashes []byte
	size := 0

	for n == ChunkSize {
		n, _ = f.Read(chunk)
		size += n
		currHash := sha256.Sum256(chunk[:n])
		chunkHashes = append(chunkHashes, currHash[:]...)
		g.chunks[currHash] = chunk
	}

	metaHash := sha256.Sum256(chunkHashes)

	g.metafiles[metaHash] = Metafile{Filename:filename, Filesize:size, ChunkHashes:chunkHashes, MetaHash:metaHash}

	return &File{filename, int64(size), metaHash[:]}
}

func (g *Gossiper) saveFile(metaHash [32]byte) {
	filename := g.metafiles[metaHash].Filename
	f, err := os.Create(Downloads + "/" + filename)
	if err!=nil {
		fmt.Println("\x1b[31;1m ERROR CREATING FILE \x1b[0m")
	}
	defer f.Close() //TODO EOF problem

	chunks := g.metafiles[metaHash].ChunkHashes
	var current [32]byte

	for i := 0 ; i<len(chunks) ; i+=32 {
		copy(current[:], chunks[i:i+32])
		_, err = f.Write(g.chunks[current])
		if err!=nil {
			fmt.Println("\x1b[31;1m ERROR WRITING FILE \x1b[0m")
		}
	}
}

func (g *Gossiper) buildChunkMap(metafile Metafile) []uint64{
	var result []uint64

	chunks := metafile.ChunkHashes
	var current [32]byte
	for i := 0 ; i<len(chunks)/32 ; i++ {
		copy(current[:], chunks[32*i:32*(i+1)])
		_,ok := g.chunks[current]
		if ok {
			result = append(result, uint64(i+1))
		}
	}

	return result
}

func (g *Gossiper) searchFile(keyword string) []*SearchResult{
	var result []*SearchResult
	for _, f := range g.metafiles {
		if strings.Contains(f.Filename, keyword) {
			res := &SearchResult{f.Filename, f.MetaHash[:], g.buildChunkMap(f), uint64(len(f.ChunkHashes)/32)}
			result = append(result, res)
		}
	}
	return result
}


/*****************************************************************************
***************************** REQUESTS FUNCTIONS *****************************
*****************************************************************************/

func (g *Gossiper) treatRequest(request *DataRequest) []byte {
	var hash [32]byte
	copy(hash[:], request.HashValue)

	metafile, ok := g.metafiles[hash]
	if ok {
		toReturn, _ := protobuf.Encode(&metafile)
		fmt.Println("SENDING METAFILE")
		return toReturn
	}

	chunk, ok := g.chunks[hash]
	if ok {
		fmt.Println("SENDING CHUNK")
		return chunk
	}

	fmt.Println("File requested not found!")

	return nil
}

func (g *Gossiper) treatReply(reply *DataReply) ([32]byte, *RequestError) {
	stopRequest := &RequestError{-1, ""}
	var hash, toReturn [32]byte
	copy(hash[:], reply.HashValue)

	metafile := Metafile{}
	err := protobuf.Decode(reply.Data, &metafile)
	if err == nil { // MetaFile request answered
		g.nextReq[metafile.MetaHash] = 1
		g.metafiles[metafile.MetaHash] = metafile

		destination := reply.Origin
		chunkMap, ok := g.chunksFound[metafile.MetaHash]
		if ok {
			dest, ok := chunkMap[uint64(1)]
			if ok {
				destination = dest
			}
		}
		stopRequest.Dest = destination

		fmt.Println("DOWNLOADING metafile of " + metafile.Filename + " from " + reply.Origin)

		if len(metafile.ChunkHashes)>=32 { // Ask for first chunk
			copy(toReturn[:], metafile.ChunkHashes[0:32])
			g.cRMutex.Lock()
			g.currentReq[toReturn] = metafile.MetaHash
			g.cRMutex.Unlock()
			g.nextReq[metafile.MetaHash] = 1
			return toReturn, nil
		}
		stopRequest.Code=1 //Empty file
		toReturn = metafile.MetaHash
		return toReturn, stopRequest
	}

	current, ok := g.currentReq[hash]
	if ok { // Chunk request
		g.cRMutex.Lock()
		delete(g.currentReq, hash)
		g.cRMutex.Unlock()
		index := g.nextReq[current]
		metafile = g.metafiles[current]
		g.chunks[hash] = reply.Data

		destination := reply.Origin
		chunkMap, ok := g.chunksFound[metafile.MetaHash]
		if ok {
			dest, ok := chunkMap[uint64(index+1)]
			if ok {
				destination = dest
			}
		}
		stopRequest.Dest = destination

		fmt.Println("DOWNLOADING " + metafile.Filename + " chunk " + fmt.Sprint(index) + " from " + reply.Origin)

		if (index+1)*32 <= len(metafile.ChunkHashes) { // Ask for next chunk
			copy(toReturn[:], metafile.ChunkHashes[index*32 : (index+1)*32])
			g.cRMutex.Lock()
			g.currentReq[toReturn] = metafile.MetaHash
			g.cRMutex.Unlock()
			g.nextReq[metafile.MetaHash] += 1
			return toReturn, nil
		} else {
			delete(g.nextReq, metafile.MetaHash)
			toReturn = current
			stopRequest.Code=1
		}
	}
	return toReturn, stopRequest
}

func (g *Gossiper) requestLoopRecursive(request *DataRequest) {

	var hash [32]byte
	copy(hash[:], request.HashValue[:])
	g.cRMutex.Lock()
	_, isChunk := g.currentReq[hash]
	g.cRMutex.Unlock()

	g.sendMessage("", request)

	end := false
	go Timeout(RequestTimeout, &end)

	acked := false
	for !end && !acked {
		if isChunk {
			g.cRMutex.Lock()
			_, temp := g.currentReq[hash]
			g.cRMutex.Unlock()
			acked = !temp
		} else {
			next, _ := g.nextReq[hash]
			acked = next>1
		}
	}

	if !acked {
		g.requestLoopRecursive(request)
	}
}

func (g *Gossiper) requestLoop(request *DataRequest) {

	acked := false

	for !acked {
		var hash [32]byte
		copy(hash[:], request.HashValue[:])
		g.cRMutex.Lock()
		_, isChunk := g.currentReq[hash]
		g.cRMutex.Unlock()

		g.sendMessage("", request)

		end := false
		go Timeout(RequestTimeout, &end)

		for !end && !acked {
			if isChunk {
				g.cRMutex.Lock()
				_, temp := g.currentReq[hash]
				g.cRMutex.Unlock()
				acked = !temp
			} else {
				next, _ := g.nextReq[hash]
				acked = next>1
			}
		}
	}
}

func (g *Gossiper) timeRequest(request *SearchRequest) {
	end := false

	go Timeout(DupRequestTimeout, &end)

	newReq := struct{req *SearchRequest; end *bool}{request, &end,}

	g.recReqMutex.Lock()
	g.recentReq[request.Origin] = append(g.recentReq[request.Origin], newReq)
	g.recReqMutex.Unlock()

	for !end {
		runtime.Gosched()
	}

	g.recReqMutex.Lock()

	reqFromOrigin := g.recentReq[request.Origin]
	for i := len(reqFromOrigin) - 1; i >= 0; i-- {
		if reqFromOrigin[i].req.Equals(newReq.req) {
			g.recentReq[request.Origin] = append(reqFromOrigin[:i], reqFromOrigin[i+1:]...)
			break
		}
	}
	g.recReqMutex.Unlock()
}

func (g *Gossiper) duplicateRequests(request *SearchRequest) bool {
	g.recReqMutex.Lock()
	reqs, ok := g.recentReq[request.Origin]
	g.recReqMutex.Unlock()

	dup := false

	if ok {
		for _, e := range reqs {
			if e.req.Equals(request) {

				if !*(e.end) {
					dup = true
				}

				break
			}
		}
	}

	go g.timeRequest(request) // Add request or update timer

	return dup
}

func (g *Gossiper) searchLoop(msg *SearchRequest) {
	msg.Budget=2

	for msg.Budget <= MaxBudget && g.totalMatches(msg) < MatchThreshold {
		msg.ForwardSearchRequest(g, "")
		end := false
		go Timeout(SearchRequestRepeatTimeout, &end)

		for !end {
			runtime.Gosched()
		}

		msg.Budget *= 2
	}
	if g.totalMatches(msg) >= MatchThreshold {
		fmt.Println("SEARCH FINISHED")
	}
}

func (g *Gossiper) totalMatches(msg *SearchRequest) int {
	resultsMap := g.mySearchReq[msg]
	completeFiles := 0
	for _, fileResults := range resultsMap {
		firstResult := fileResults[0].Results[0]
		totalCount := firstResult.ChunkCount
		chunkFound := make([]bool, totalCount)
		for _, results := range fileResults {
			for _, result := range results.Results {
				for _, chunkNumber := range result.ChunkMap {
					chunkFound[chunkNumber-1] = true
				}
			}
		}
		if IsComplete(chunkFound) {
			completeFiles+=1
		}
	}

	return completeFiles
}


/*****************************************************************************
********************************* BLOCKCHAIN *********************************
*****************************************************************************/

func (g *Gossiper) miningMain() {
	for {
		for len(g.txPending)<=0 || len(g.txMining)>0  {
			runtime.Gosched()
		}

		prevTimeout := g.blockTimer

		start := time.Now()
		newBlock := g.mine()
		elapsed := time.Since(start)
		g.txMining = make([]TxPublish, 0, 5)

		if newBlock != nil {
			toSend := &BlockPublish{*newBlock, BlockHopLimit+1}
			go toSend.sendWithDelay(g, prevTimeout)
			g.blockTimer = elapsed*2
		}
	}
}

func (g *Gossiper) mine() *Block{
	var lastBlock [32] byte
	if g.lastBlock!=nil {
		lastBlock = g.lastBlock.Hash
	}
	g.txMining = make([]TxPublish, len(g.txPending))
	copy(g.txMining, g.txPending)

	valid := false
	var blockToMine Block
	var currentHash [32]byte

	for !valid {

		blockToMine = Block{lastBlock, GenerateNonce(), g.txMining}
		currentHash = blockToMine.Hash()
		valid = BlockHashValid(currentHash)

		if Obsolete(g.txMining, g.txPending) {
			g.txMining = make([]TxPublish, 0)
			return nil
		}
	}

	fmt.Println("FOUND-BLOCK " + fmt.Sprintf("%s", hex.EncodeToString(currentHash[:])))

	return &blockToMine
}


func (g *Gossiper) transfersMiningMain() {
	for {
		for len(g.transfersPending)<=0 || len(g.transfersMining)>0  {
			runtime.Gosched()
		}

		prevTimeout := g.blockTimer

		start := time.Now()
		newBlock := g.transfersMine()
		elapsed := time.Since(start)
		g.transfersMining = make([]MoneyTxPublish, 0, 5)

		if newBlock != nil {
			toSend := &MoneyBlockPublish{*newBlock, BlockHopLimit+1}
			go toSend.sendWithDelay(g, prevTimeout)
			g.blockTimer = elapsed*2
		}
	}
}

func (g *Gossiper) transfersMine() *MoneyBlock{
	var lastBlock [32] byte
	if g.lastBlock!=nil {
		lastBlock = g.lastBlock.Hash
	}
	g.transfersMining = make([]MoneyTxPublish, len(g.transfersPending))
	copy(g.transfersMining, g.transfersPending)

	valid := false
	var blockToMine MoneyBlock
	var currentHash [32]byte

	for !valid {

		blockToMine = MoneyBlock{lastBlock, GenerateNonce(), g.transfersMining}
		currentHash = blockToMine.Hash()
		valid = BlockHashValid(currentHash)

		if ObsoleteTransfer(g.transfersMining, g.transfersPending) {
			g.transfersMining = make([]MoneyTxPublish, 0)
			return nil
		}
	}

	fmt.Println("FOUND-BLOCK " + fmt.Sprintf("%s", hex.EncodeToString(currentHash[:])))

	return &blockToMine
}

func (g *Gossiper) checkCond() {
	for {
		unchanged := true
		var toRemove []*MoneyTxPublish
		for _, tx := range g.transfersCond {
			if tx.Condition.Check() {

				unchanged = false
				toRemove = append(toRemove, tx)
				g.transfersMap[tx.Identifier()] = tx
				g.wallets[tx.Sender] = g.wallets[tx.Sender] - tx.Amount
				g.wallets[tx.Receiver] = g.wallets[tx.Receiver] + tx.Amount

				if g.wallets[tx.Sender] < 0.0 {
					g.wallets[tx.Sender] = g.wallets[tx.Sender] + tx.Amount
					g.wallets[tx.Receiver] = g.wallets[tx.Receiver] - tx.Amount
				}
			}
		}
		for _, tx := range toRemove {
			delete(g.transfersCond, tx.Identifier())
		}
		if unchanged {
			end := false
			go Timeout(TxChangesTimeout, &end)
			for !end {
				runtime.Gosched()
			}
		} else {
			g.printWallets()
		}
	}
}


/*****************************************************************************
******************************* TOOL FUNCTIONS *******************************
*****************************************************************************/

func (g *Gossiper) printMsg(msg Message, origin string){
	fmt.Println(msg.Display(origin))
	g.printPeers()
}

func (g *Gossiper) printPeers(){
	s := "PEERS "
	for i,p := range g.peers {
		if p != "" {
			s += p
			if i < len(g.peers)-1 {
				s += ","
			}
		}
	}
	fmt.Println(s)
}

func (g *Gossiper) cleanPeers() {
	g.peers = RemoveFromList("", g.peers, 0)
	deleted := 0
	for i := range g.peers { // Removing duplicates
		j := i-deleted
		if j < len(g.peers)-1{
			g.peers = RemoveFromList(g.peers[j], g.peers, j+1)
		} else {
			return
		}
	}
}

func (g *Gossiper) printBlockChain() {
	fmt.Println(g.lastBlock.ChainString())
}

func (g *Gossiper) printWallets() {
	keys := make([]string, 0, len(g.wallets))
	for key := range g.wallets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	s := "BALANCES "
	for ind, node := range keys {
		balance := g.wallets[node]
		s += node + ":" + fmt.Sprintf("%.2f", balance)
		if ind < len(keys)-1 {
			s +=","
		}
		ind++
	}
	fmt.Println(s)
}