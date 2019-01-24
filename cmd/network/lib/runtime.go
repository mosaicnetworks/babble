package runtime

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/mosaicnetworks/babble/src/babble"
)

type Node struct {
	i    int
	proc *os.Process
}

type Runtime struct {
	config     babble.BabbleConfig
	nbNodes    int
	sendTx     int
	nextNodeId int
	processes  map[int]*os.Process
}

func New(babbleConfig babble.BabbleConfig, nbNodes int, sendTx int) *Runtime {
	return &Runtime{
		config:     babbleConfig,
		sendTx:     sendTx,
		nextNodeId: 0,
		nbNodes:    nbNodes,
		processes:  make(map[int]*os.Process),
	}
}

func (r *Runtime) buildConfig() error {
	babblePort := 1337

	peersJSON := `[`

	for i := r.nextNodeId; i < r.nbNodes+r.nextNodeId; i++ {
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
	if r.nextNodeId > 0 {
		// use the first node's peers.json
		for i := r.nextNodeId; i < r.nbNodes+r.nextNodeId; i++ {
			nb := strconv.Itoa(i)

			file, err := ioutil.ReadFile("/tmp/babble_configs/.babble0/peers.json")

			if err != nil {
				log.Fatal(err)
			}

			err = ioutil.WriteFile("/tmp/babble_configs/.babble"+nb+"/peers.json", file, 0644)

			if err != nil {
				log.Fatal(err)
			}
		}

		return nil
	}

	for i := r.nextNodeId; i < r.nbNodes+r.nextNodeId; i++ {
		nb := strconv.Itoa(i)

		err := ioutil.WriteFile("/tmp/babble_configs/.babble"+nb+"/peers.json", []byte(peersJSON), 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func (r *Runtime) sendTxs(i int) {
	nb := strconv.Itoa(i)

	network := exec.Command("network", "proxy", "--node="+nb, "--submit="+nb)

	err := network.Run()

	if err != nil {
		fmt.Println("Error: ", err)
	} else {
		fmt.Println("Ok")
	}
}

func (r *Runtime) runBabbles() error {
	if r.nextNodeId == 0 {
		os.RemoveAll("/tmp/babble_configs")
	}

	if err := r.buildConfig(); err != nil {
		log.Fatal(err)
	}

	babblePort := 1337
	servicePort := 8080

	wg := sync.WaitGroup{}

	// if existing network, join without creating peers.json

	for i := r.nextNodeId; i < r.nbNodes+r.nextNodeId; i++ {
		wg.Add(1)

		go func(i int) {
			nb := strconv.Itoa(i)

			babblePortStr := strconv.Itoa(babblePort + (i * 10))
			proxyServPortStr := strconv.Itoa(babblePort + (i * 10) + 1)
			proxyCliPortStr := strconv.Itoa(babblePort + (i * 10) + 2)

			servicePort := strconv.Itoa(servicePort + i)

			// defer wg.Done()

			read, write, err := os.Pipe()

			defer write.Close()

			if err != nil {
				fmt.Println("Cannot create pipe", err)

				return
			}

			var babbleNode *exec.Cmd

			// if r.nextNodeId == 0 {
			// 	babbleNode = exec.Command("babble", "run", "-l=127.0.0.1:"+babblePortStr, "--datadir=/tmp/babble_configs/.babble"+nb, "--proxy-listen=127.0.0.1:"+proxyServPortStr, "--client-connect=127.0.0.1:"+proxyCliPortStr, "-s=127.0.0.1:"+servicePort, "--heartbeat="+r.config.NodeConfig.HeartbeatTimeout.String())
			// } else {
			// 	babbleNode = exec.Command("babble", "run", "-l=127.0.0.1:"+babblePortStr, "--datadir=/tmp/babble_configs/.babble"+nb, "--proxy-listen=127.0.0.1:"+proxyServPortStr, "--client-connect=127.0.0.1:"+proxyCliPortStr, "-s=127.0.0.1:"+servicePort, "--heartbeat="+r.config.NodeConfig.HeartbeatTimeout.String(), "-c=127.0.0.1:1337")
			// }
			babbleNode = exec.Command("babble", "run", "-l=127.0.0.1:"+babblePortStr, "--datadir=/tmp/babble_configs/.babble"+nb, "--proxy-listen=127.0.0.1:"+proxyServPortStr, "--client-connect=127.0.0.1:"+proxyCliPortStr, "-s=127.0.0.1:"+servicePort, "--heartbeat="+r.config.NodeConfig.HeartbeatTimeout.String())

			babbleNode.Stdout = write
			babbleNode.Stderr = write

			babbleNode.SysProcAttr = &syscall.SysProcAttr{
				Setpgid: true,
			}

			out, err := os.OpenFile("/tmp/babble_configs/.babble"+nb+"/out.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)

			if err != nil {
				fmt.Println("Cannot open log file", err)

				return
			}

			go func() {
				defer read.Close()
				defer out.Close()

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

			wg.Done()

			if r.sendTx > 0 {
				go r.sendTxs(i)
			}

			r.processes[i] = babbleNode.Process

			babbleNode.Wait()

			fmt.Println("Terminated", i)

			delete(r.processes, i)
		}(i)
	}

	wg.Wait()

	r.nextNodeId += r.nbNodes

	return nil
}

func (r *Runtime) Kill(node int) error {
	if node < 0 {
		for _, proc := range r.processes {
			proc.Kill()

			r.processes = map[int]*os.Process{}
		}
	} else if proc, ok := r.processes[node]; ok {
		proc.Kill()

		delete(r.processes, node)
	} else {
		fmt.Println("Unknown process")
	}

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
			fallthrough

		case "help":
			help()

		case "p":
			fallthrough

		case "proxy":
			node := 0

			if len(splited) >= 2 {
				node, _ = strconv.Atoi(splited[1])
			}

			r.sendTxs(node)

		case "r":
			fallthrough

		case "run":
			if len(splited) >= 2 {
				r.nbNodes, _ = strconv.Atoi(splited[1])
			}

			r.runBabbles()

		case "l":
			fallthrough

		case "log":
			nb := "0"

			if len(splited) >= 2 {
				nb = splited[1]
			}

			ReadLog(nb)

		case "list":
			for i := range r.processes {
				fmt.Println(i, ": Babble")
			}

		case "k":
			fallthrough

		case "kill":
			nb := "0"

			if len(splited) >= 2 {
				nb = splited[1]
			}

			inb, _ := strconv.Atoi(nb)

			r.Kill(inb)

		case "killall":
			r.Kill(-1)

		case "q":
			fallthrough

		case "quit":
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
	logs := exec.Command("tail", "-f", "/tmp/babble_configs/.babble"+nb+"/out.log")

	// This is crucial - otherwise it will write to a null device.
	logs.Stdout = os.Stdout

	logs.Run()
}

func help() {
	fmt.Println("Commands:")
	fmt.Println("  r | run [nb=4]     - Run `nb` babble nodes")
	fmt.Println("  p | proxy [node=0] - Send a transaction to a node")
	fmt.Println("  l | log [node=0]   - Show logs for a node")
	fmt.Println("  h | help           - This help")
	fmt.Println("  k | kill [node=0]  - Kill given node")
	fmt.Println("      killall        - Kill all nodes")
	fmt.Println("      list           - List all running nodes")
	fmt.Println("  q | quit           - Quit")
}
