package net

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	webrtc "github.com/pion/webrtc/v2"
)

// Signal defines an interface for systems to exchange SDP offers and answers
// to establish WebRTC PeerConnections
type Signal interface {
	// Addr returns the local address used to identify this end of a connection
	Addr() string

	// Listen is called to listen for incoming SDP offers, and forward them to
	// to the Consumer channel
	Listen() error

	// Consumer is the channel through which incoming SDP offers are passed to
	// the WebRTCStreamLayer. SDP offers are wrapped around an RPC object which
	// offers a response mechanism.
	Consumer() <-chan RPC

	// Offer sends an SDP offer and waits for an answer
	Offer(target string, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error)
}

// TestSignal implements the Signal interface by reading and writing files on
// disk. It is only used for testing.
type TestSignal struct {
	addr     string
	consumer chan RPC
	dir      string
}

// NewTestSignal instantiates a TestSignal
func NewTestSignal(addr string, dir string) *TestSignal {
	return &TestSignal{
		addr:     addr,
		consumer: make(chan RPC),
		dir:      dir,
	}
}

// Addr implements the Signal interface. It returns the local address used to
// identify this end of the connection.
func (ts *TestSignal) Addr() string {
	return ts.addr
}

// Listen implements the Signal interface. It scans the test directory for
// offers and submits new offers to the consumer channel. Filenames are of the
// for <offerer>_<answerer>_offer.sdp or <offerer>_<answerer>_answer.sdp. So for
// example if alice makes an offer to bob, she will write the offer in a file
// called alice_bob_offer.sdp and bob will answer in alice_bob_answer.sdp
func (ts *TestSignal) Listen() error {

	// processedOffers keeps track of the offers that have already been
	// processed
	processedOffers := make(map[string]bool)

	for {

		// open the directory where sdp files are dumped
		sdpDir, err := os.Open(ts.dir)
		if err != nil {
			return err
		}

		//scan directory
		fileNames, err := sdpDir.Readdirnames(0)
		if err != nil {
			return err
		}

		for _, fileName := range fileNames {
			s := strings.Split(fileName, "_")

			if len(s) != 3 ||
				s[1] != ts.addr ||
				s[2] != "offer.sdp" {
				continue
			}

			if _, ok := processedOffers[s[0]]; ok {
				continue
			}

			offer, err := readSDP(filepath.Join(ts.dir, fileName))
			if err != nil {
				return err
			}

			if offer != nil {
				respCh := make(chan RPCResponse, 1)

				rpc := RPC{
					Command:  *offer,
					RespChan: respCh,
				}

				ts.consumer <- rpc

				// Wait for response
				select {
				case resp := <-respCh:
					fmt.Println("Signal writing answer file")
					answerFilename := fmt.Sprintf("%s_%s_answer.sdp", s[0], s[1])
					writeSDP(resp.Response.(webrtc.SessionDescription), filepath.Join(ts.dir, answerFilename))
					break
				}

				processedOffers[s[0]] = true
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// Consumer implements the Signal interface
func (ts *TestSignal) Consumer() <-chan RPC {
	return ts.consumer
}

// Offer implements the Signal interface
func (ts *TestSignal) Offer(target string, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {

	offerFilename := fmt.Sprintf("%s_%s_offer.sdp", ts.addr, target)
	err := writeSDP(offer, filepath.Join(ts.dir, offerFilename))
	if err != nil {
		return nil, err
	}

	answerFilename := fmt.Sprintf("%s_%s_answer.sdp", ts.addr, target)
	answerFile := filepath.Join(ts.dir, answerFilename)

	timeout := time.After(5 * time.Second)
	for {
		select {
		case <-timeout:
			err := fmt.Errorf("Timeout waiting for SDP answer")
			return nil, err
		default:
			answer, err := readSDP(answerFile)
			if err != nil {
				fmt.Printf("Error reading answer file: %v\n", err)
				return nil, err
			}

			if answer != nil {
				return answer, nil
			}

			time.Sleep(100 * time.Millisecond)
		}
	}
}

func readSDP(file string) (*webrtc.SessionDescription, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, nil
	}

	fileContent, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	res := webrtc.SessionDescription{}

	err = json.Unmarshal(fileContent, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func writeSDP(sdp webrtc.SessionDescription, file string) error {
	raw, err := json.Marshal(sdp)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(file, raw, 0644)
	if err != nil {
		return err
	}

	return nil
}
