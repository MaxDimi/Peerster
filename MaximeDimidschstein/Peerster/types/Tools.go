package types

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"time"
)

var HopLimit = uint32(10)
var TxHopLimit = uint32(10)
var BlockHopLimit = uint32(20)
var ChunkSize = 8192
var SharedDir = "_SharedFiles"
var Downloads = "_Downloads"
var RumorTimeout = time.Second
var AntiEntropyTimeout = time.Second
var RequestTimeout = time.Second
var DupRequestTimeout = 500000*time.Nanosecond
var RequestRepeatTimeout = 500000*time.Nanosecond
var InitialBudget = uint64(2)
var MaxBudget = uint64(32)
var SearchRequestRepeatTimeout = time.Second
var MatchThreshold = 2
var ZeroBitsThreshold = 16
var initialBlockTimeout = 100*time.Nanosecond//5*time.Second
var InitialBalance = 20.0
var Loc, _ = time.LoadLocation("UTC")
var TxChangesTimeout = 1*time.Second


type RequestError struct {
	Code int
	Dest string
}

func (e *RequestError) Error() string {
	if e.Code==0 {
		return "Request next"
	} else if e.Code==1 {
		return "Transfer completed"
	}
	return "Stop requesting"
}

func IsIn(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func IndexOfStr(identifier string, list []string) int {
	for i, p := range list {
		if p == identifier {
			return i
		}
	}
	return -1
}

func IndexOf(identifier string, list []PeerStatus) int {
	for i, p := range list {
		if p.Identifier == identifier {
			return i
		}
	}
	return -1
}

func RemoveFrom(list []string, i int) []string {
	list = append(list[:i], list[i+1:]...)
	return list
}

func RemoveFromList(toRemove string, list []string, from int) []string {
	for i := len(list) - 1; i >= from; i-- {
		if list[i] == toRemove {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func Timeout(d time.Duration, end *bool){
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	*end = false
	for {
		select {
		case <-ticker.C:
			*end = true
			return
		}
	}
}

func AppendUnique(slice []*SearchResult, res *SearchResult) []*SearchResult {
	for _, e := range slice {
		if bytes.Equal(e.MetafileHash, res.MetafileHash) {
			return slice
		}
	}
	return append(slice, res)
}

func Merge(a []*SearchResult, b []*SearchResult) []*SearchResult {

	if b==nil || len(b)==0 {
		return a
	} else if a==nil || len(a)==0 {
		return b
	}

	for _, e := range b {
		a = AppendUnique(a, e)
	}
	return a
}

func IsComplete(a []bool) bool {
	for _, b := range a {
		if !b {
			return false
		}
	}
	return true
}

func BlockHashValid(blockHash [32]byte) bool {
	if blockHash[0] & 0xFF == 0 && blockHash[1] & 0xFF == 0 {
		return true
	}
	return false
}

func GenerateNonce() [32]byte {
	var nonce [32]byte
	n, err := rand.Read(nonce[:])
	if err != nil || n!=32 {
		fmt.Println("ERROR creating nonce: " + fmt.Sprint(err))
	}

	return nonce
}

func Obsolete(txMining, txPending []TxPublish) bool{
	for _, txM := range txMining {
		found := false
		for i:=0 ; i<len(txPending) && !found ; i++ {
			if txM.File.Name == txPending[i].File.Name {
				found = true
			}
		}
		if !found {
			return true
		}
	}
	return false
}

func ObsoleteTransfer(txMining, txPending []MoneyTxPublish) bool{
	for _, txM := range txMining {
		found := false
		for i:=0 ; i<len(txPending) && !found ; i++ {
			if txM.Identifier() == txPending[i].Identifier() {
				found = true
			}
		}
		if !found {
			return true
		}
	}
	return false
}

func GetLocalOffset() string{
	_, offset := time.Now().Local().Zone()
	hr := offset/3600
	offset_str := ""
	if hr<0 {
		offset_str = "-"
	} else {
		offset_str = "+"
	}
	if math.Abs(float64(hr))<10 {
		offset_str += "0" + fmt.Sprint(hr) + ":00"
	} else {
		offset_str += fmt.Sprint(hr) + ":00"
	}
	return offset_str
}