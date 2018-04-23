package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	cn "golang.org/x/image/colornames"

	"github.com/babbleio/babble/mobile"
	"github.com/babbleio/babble/net"
)

type AppMessageHandler struct{}

type ConfigData struct {
	Peers           []net.Peer
	NodeID          int    //0 one of array indexes above
	NodePrivateKey  string //"-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIP7PIEnr/7RUSLc55XP44GTsAxsSg/AzekqHqXDQxfGKoAoGCCqGSM49AwEHoUQDQgAEpM2p+b0DxYTEJaPtGiaM2oVjtixMbx5S6BrVUDREuH8A4rNKrAWohJFZHHG6U5w15y9KaTYntoB5Cq/mS0x6ww==\n-----END EC PRIVATE KEY-----",
	NodePublicKey   string //"0x04A4CDA9F9BD03C584C425A3ED1A268CDA8563B62C4C6F1E52E81AD5503444B87F00E2B34AAC05A88491591C71BA539C35E72F4A693627B680790AAFE64B4C7AC3",
	Node_addr       string // "10.128.1.36:1331",
	Proxy_addr      string //"10.128.1.36:1332",
	Client_addr     string //"10.128.1.36:1333",
	Service_addr    string //"10.128.1.36:1334",
	StoreType       string //"inmem" or "badger"
	StorePath       string //"D:\\Projects\\go-work\\src\\github.com\\babbleio\\babble\\DB\\DB0" /if badger "Path to badger", If inmem - "" empty
	CircleBackColor string //"BLUE"  Back color
	CircleForeColor string //"YELLOW" Fore color
	CircleRadius    int    //[50, 150]
}

func (h *AppMessageHandler) OnError(ctx *mobile.ErrorContext) {
	fmt.Printf("Error: %s\n", ctx.Error)
}

func (h *AppMessageHandler) OnCommit(ctx *mobile.TxContext) {
	fmt.Printf("Msg: %s\n", ctx.Data)
}

func initNode(cnfgData *ConfigData) (node *mobile.Node) {

	nodeAddr := cnfgData.Node_addr
	var privKey = cnfgData.NodePrivateKey
	var config = mobile.DefaultConfig()
	config.SyncLimit = 100

	var msgHandler = &AppMessageHandler{}
	var events = mobile.NewEventHandler()

	events.SetHandlers(msgHandler, msgHandler)

	peers, _ := json.Marshal(cnfgData.Peers)

	node = mobile.New(nodeAddr, string(peers), privKey, events, config) //Ivan
	return
}

func toColor(b [4]byte) int32 {
	return (int32(b[0]) << 24) + (int32(b[1]) << 16) + (int32(b[2]) << 8) + int32(b[3])
}

func (cnfgData *ConfigData) getConfigData(fullPath string) error {

	file, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cnfgData)
	if err != nil {
		return err
	}

	return nil
}

func getColorByName(name string) int32 {

	colorId := cn.Map[strings.ToLower(name)]

	if colorId.A == 0 && colorId.R == 0 && colorId.G == 0 && colorId.B == 0 && strings.ToLower(name) != "black" {
		colorId = cn.Red
	}

	return (int32(colorId.A) << 24) + (int32(colorId.R) << 16) + (int32(colorId.G) << 8) + int32(colorId.B)
}

func main() {

	arg1 := os.Args[1]
	//arg1 := "D:\\Projects\\go-work\\src\\github.com\\babbleio\\babble\\config\\ConfigBabbleGo0.json"

	cnfgData := new(ConfigData)
	if err := cnfgData.getConfigData(arg1); err != nil {
		fmt.Printf("An error occur while reading config file '%s'\r\nError: %s", arg1, err.Error())
		return
	}

	var node = initNode(cnfgData)

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
			ball.Size = 50
			ball.CircleBackColor = getColorByName(cnfgData.CircleBackColor)
			ball.CircleForeColor = getColorByName(cnfgData.CircleForeColor)

			var binBuf bytes.Buffer
			binary.Write(&binBuf, binary.LittleEndian, ball)

			node.SubmitTx(binBuf.Bytes())
			time.Sleep(5000 * time.Millisecond)
		}
	})()

	node.Run(false)
}
