package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"

	"github.com/babbleio/babble/mobile"
)

type AppMessageHandler struct{}

func (h *AppMessageHandler) OnError(ctx *mobile.ErrorContext) {
	fmt.Printf("Error: %s\n", ctx.Error)
}

func (h *AppMessageHandler) OnCommit(ctx *mobile.TxContext) {
	fmt.Printf("Msg: %s\n", ctx.Data)
}

func initNode() (node *mobile.Node) {

	nodeAddr := "10.128.1.26:1337"

	var peers = "[\n" +
		"    {\n" +
		"            \"NetAddr\":\"" + nodeAddr + "\",\n" +
		"            \"PubKeyHex\":\"0x04DB3C17CCE6F68B6A78137B5A39D427D2E42A787AEC6EFA749446CB9E9E6EB4C1D486DE94CD5EEA2ABE21C54E59A31045A4BF389E2E9A585260B4BBDB3EDB4440\"\n" +
		"    },\n" +
		"\t{\n" +
		"            \"NetAddr\":\"10.128.1.136:1337\",\n" +
		"            \"PubKeyHex\":\"0x043DE6FB7ED40017FB04B72BD8B99BE4A75851D8EFD1D4B4A1EAB7A9C0285DA135525D44930A69DCD227FCA78640D7C2DCF9AB1C23BC3E5B880296F4B878923F6F\"\n" +
		"    }\n" +
		"]"

	var privKey = "-----BEGIN EC PRIVATE KEY-----\n" +
		"MHcCAQEEIPqlPITl5LFwhbbjJaa3wahx/8Imk0P1AWU5VLRpbz30oAoGCCqGSM49\n" +
		"AwEHoUQDQgAE2zwXzOb2i2p4E3taOdQn0uQqeHrsbvp0lEbLnp5utMHUht6UzV7q\n" +
		"Kr4hxU5ZoxBFpL84ni6aWFJgtLvbPttEQA==\n" +
		"-----END EC PRIVATE KEY-----"

	var config = mobile.DefaultConfig()
	config.SyncLimit = 100

	var msgHandler = &AppMessageHandler{}
	var events = mobile.NewEventHandler()

	events.SetHandlers(msgHandler, msgHandler)

	node = mobile.New(nodeAddr, peers, privKey, events, config)
	return
}

func toColor(b [4]byte) int32 {
	return (int32(b[0]) << 24) + (int32(b[1]) << 16) + (int32(b[2]) << 8) + int32(b[3])
}

func main() {

	var node = initNode()

	if node == nil {
		fmt.Println("Fail to create node")
		return
	}

	go (func() {

		time.Sleep(1 * time.Second)
		fmt.Println("Start sending txs")

		for true {
			ball := mobile.Ball{}
			ball.X = rand.Int31n(700)
			ball.Y = rand.Int31n(1000)
			ball.Size = 60
			ball.Color = toColor([4]byte{0xFF, 250, 186, 37}) // ARGB

			var binBuf bytes.Buffer
			binary.Write(&binBuf, binary.LittleEndian, ball)

			node.SubmitTx(binBuf.Bytes())
			time.Sleep(5000 * time.Millisecond)
		}
	})()

	node.Run(false)
}
