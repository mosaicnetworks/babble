package runtime

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mosaicnetworks/babble/src/babble"
)

type Node struct {
	id     int
	babble *babble.Babble
}

type Runtime struct {
	config       babble.BabbleConfig
	nbNodes      int
	sendTx       int
	runningNodes []*Node
}

func New(babbleConfig babble.BabbleConfig, nbNodes int, sendTx int) *Runtime {
	return &Runtime{
		config:       babbleConfig,
		sendTx:       sendTx,
		nbNodes:      nbNodes,
		runningNodes: []*Node{},
	}
}

func (r *Runtime) buildConfig() error {
	babblePort := 1337

	peersJSON := `[`

	for i := 0; i < r.nbNodes; i++ {
		nb := strconv.Itoa(i)

		babblePortStr := strconv.Itoa(babblePort + (i * 10))

		babbleNode := exec.Command("babble", "keygen", "--pem=/tmp/babble_configs/.babble"+nb+"/priv_key.pem", "--pub=/tmp/babble_configs/.babble"+nb+"/key.pub")

		res, err := babbleNode.CombinedOutput()
		if err != nil {
			log.Fatal(err, res)
		}

		pubKey, err := ioutil.ReadFile("/tmp/babble_configs/.babble" + nb + "/key.pub")
		if err != nil {
			log.Fatal(err, res)
		}

		peersJSON += `	{
		"NetAddr":"127.0.0.1:` + babblePortStr + `",
		"PubKeyHex":"` + string(pubKey) + `"
	},
`
	}

	peersJSON = peersJSON[:len(peersJSON)-2]
	peersJSON += `
]
`

	if r.nbNodes == 1 {
		return nil
	}

	for i := 0; i < r.nbNodes; i++ {
		nb := strconv.Itoa(i)

		err := ioutil.WriteFile("/tmp/babble_configs/.babble"+nb+"/peers.json", []byte(peersJSON), 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func (r *Runtime) sendTxs(babbleNode *exec.Cmd, i int) {
	ticker := time.NewTicker(1 * time.Second)
	nb := strconv.Itoa(i)

	txNb := 0

	for range ticker.C {
		if txNb == r.sendTx {
			ticker.Stop()

			break
		}

		network := exec.Command("network", "proxy", "--node="+nb, "--submit="+nb+"_"+strconv.Itoa(txNb))

		err := network.Run()
		if err != nil {
			continue
		}

		txNb++
	}
}

func (r *Runtime) runBabbles() error {
	os.RemoveAll("/tmp/babble_configs")

	if err := r.buildConfig(); err != nil {
		log.Fatal(err)
	}

	babblePort := 1337
	servicePort := 8080

	wg := sync.WaitGroup{}

	var processes = make([]*os.Process, r.nbNodes)

	for i := 0; i < r.nbNodes; i++ {
		wg.Add(1)

		go func(i int) {
			nb := strconv.Itoa(i)
			babblePortStr := strconv.Itoa(babblePort + (i * 10))
			proxyServPortStr := strconv.Itoa(babblePort + (i * 10) + 1)
			proxyCliPortStr := strconv.Itoa(babblePort + (i * 10) + 2)

			servicePort := strconv.Itoa(servicePort + i)

			defer wg.Done()

			read, write, err := os.Pipe()
			defer write.Close()

			if err != nil {
				fmt.Println("Cannot create pipe", err)

				return
			}

			babbleNode := exec.Command("babble", "run", "-l=127.0.0.1:"+babblePortStr, "--datadir=/tmp/babble_configs/.babble"+nb, "--proxy-listen=127.0.0.1:"+proxyServPortStr, "--client-connect=127.0.0.1:"+proxyCliPortStr, "-s=127.0.0.1:"+servicePort, "--heartbeat="+r.config.NodeConfig.HeartbeatTimeout.String())

			babbleNode.Stdout = write
			babbleNode.Stderr = write

			out, err := os.OpenFile("/tmp/babble_configs/.babble"+nb+"/out.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)

			if err != nil {
				fmt.Println("Cannot open log file", err)

				return
			}

			go func() {
				defer read.Close()
				// copy the data written to the PipeReader via the cmd to stdout
				if _, err := io.Copy(out, read); err != nil {
					log.Fatal(err)
				}
			}()

			err = babbleNode.Start()

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Running", i)

			if r.sendTx > 0 {
				go r.sendTxs(babbleNode, i)
			}

			processes[i] = babbleNode.Process

			babbleNode.Wait()

			fmt.Println("Terminated", i)

		}(i)
	}

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		for range c {
			for _, proc := range processes {
				proc.Kill()
			}
		}
	}()

	wg.Wait()

	return nil
}

func (r *Runtime) Start() error {
	running := true

	fmt.Println("Type 'h' to get help")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	for running {
		fmt.Print("$> ")

		if !scanner.Scan() {
			return nil
		}

		ln := scanner.Text()

		splited := strings.Split(ln, " ")

		switch splited[0] {
		case "h":
			help()
		case "r":
			fallthrough
		case "run":
			if len(splited) >= 2 {
				r.nbNodes, _ = strconv.Atoi(splited[1])
			}

			go r.runBabbles()

		case "l":
			fallthrough
		case "log":
			nb := "0"
			if len(splited) >= 2 {
				nb = splited[1]
			}

			ReadLog(nb)
		case "q":
			running = false

			break
		case "":
		default:
			fmt.Println("Unknown command", splited[0])
		}
	}

	return nil
}

func ReadLog(nb string) {
	test := exec.Command("tail", "-f", "/tmp/babble_configs/.babble"+nb+"/out.log")

	test.Stdin = os.Stdin

	stdin := os.Stdin

	var err error

	os.Stdin, err = os.Open("/dev/null")

	if err != nil {
		fmt.Println("ERROR", err)

		return
	}

	// This is crucial - otherwise it will write to a null device.
	test.Stdout = os.Stdout

	test.Run()

	os.Stdin = stdin
}

func help() {
	fmt.Println("Commands:")
	fmt.Println("  r | run [nb=4]   - Run `nb` babble nodes")
	fmt.Println("  l | log [node=0] - Show logs for a node")
	fmt.Println("  h                - This help")
	fmt.Println("  q                - Quit")
}
