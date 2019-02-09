package main

import (
	"encoding/hex"
	"flag"
	"github.com/MaximeDimidschstein/Peerster/types"
	"net"
	"strings"
	"time"
)

func main() {

	UIPort := flag.String("UIPort", "8080", "port for the UI client")
	dest := flag.String("dest", "", "destination for the private message")
	file := flag.String("file", "", "file to be indexed by the gossiper")
	msg := flag.String("msg", "", "message to be sent")
	request := flag.String("request", "", "request a chunk or metafile of this hash")
	keywords := flag.String("keywords", "", "search for files according to the list of keywords")
	budget := flag.Uint64("budget", types.InitialBudget, "budget for the file search")
	amount := flag.Float64("amount", 0.0, "amount of money to send to dest")
	delay := flag.String("delay", "", "delay to make the transaction effective (format as a Duration string, e.g. '4d3h2m1s'")
	date := flag.String("date", "", "date on which the transaction becomes effective (format as '2019-01-014T23:59:59'")

	flag.Parse()

	flagset := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { flagset[f.Name]=true } )

	lAddr, _ := net.ResolveUDPAddr("udp", ":0")
	rAddr, _ := net.ResolveUDPAddr("udp", ":"+*UIPort)

	conn, _ := net.DialUDP("udp", lAddr, rAddr)
	var packetBytes []byte

	if *request!="" && *file != "" {
		req, _ := hex.DecodeString(*request)
		packetBytes = (&types.DataRequest{"", *dest, types.HopLimit, req}).WrapMsg(nil)
	} else if *file != "" {
		packetBytes = (&types.FileMessage{*file}).WrapMsg(nil)
	} else if *keywords!=""{
		keywordsList := strings.Split(*keywords, ",")
		budgetIsSet, ok := flagset["budget"]
		if !ok || !budgetIsSet {
			*budget = 0
		}
		packetBytes = (&types.SearchRequest{"", uint64(*budget), keywordsList}).WrapMsg(nil)
	} else if *amount > 0.0 {
		if *delay!="" {
			delay_D, err := time.ParseDuration(*delay)
			if err==nil {
				date_T := time.Now().In(types.Loc).Add(delay_D)
				packetBytes = (&types.MoneyTxPublish{"", *dest, *amount, types.HopLimit, 0, types.TxCondition{date_T}, "0", "0"}).WrapMsg(nil)
			}
		} else if *date!=""{
			date_T, err := time.ParseInLocation(time.RFC3339, *date+types.GetLocalOffset(), time.Local)
			if err==nil {
				packetBytes = (&types.MoneyTxPublish{"", *dest, *amount, types.HopLimit, 0, types.TxCondition{date_T.In(types.Loc)}, "0", "0"}).WrapMsg(nil)
			}
		}
		if len(packetBytes)<=0 {
			packetBytes = (&types.MoneyTxPublish{"", *dest, *amount, types.HopLimit, 0, types.DefaultCondition, "0", "0"}).WrapMsg(nil)
		}
	} else if *dest!="" {
		packetBytes = (&types.PrivateMessage{"", 0, *msg, *dest, types.HopLimit}).WrapMsg(nil)
	} else if *dest=="" {
		packetBytes = (&types.ClientMessage{*msg}).WrapMsg(nil)
	}

	conn.Write(packetBytes)

	defer conn.Close()
}