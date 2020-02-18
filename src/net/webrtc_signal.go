package net

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"time"

	"github.com/pion/webrtc"
)

// Signal defines an interface for systems to exchange SDP offers and answers
// to establish WebRTC PeerConnections
type Signal interface {

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
	offerFile  string
	answerFile string
	consumer   chan RPC
	lastOffer  *webrtc.SessionDescription
	lastAnswer *webrtc.SessionDescription
}

// NewTestSignal instantiates a TestSignal from two file paths.
func NewTestSignal(offerFile, answerFile string) *TestSignal {
	return &TestSignal{
		offerFile:  offerFile,
		answerFile: answerFile,
		consumer:   make(chan RPC),
	}
}

// Listen implements the Signal interface. It tracks the offer file and submits
// new offers to the consumer
func (ts *TestSignal) Listen() error {
	for {
		offer, err := readSDP(ts.offerFile)
		if err != nil {
			return err
		}

		if offer != nil {

			fmt.Println("offer file found")

			if ts.lastOffer != nil && reflect.DeepEqual(ts.lastOffer, offer) {
				fmt.Println("offer file hasn't changed")
			} else {
				respCh := make(chan RPCResponse, 1)

				rpc := RPC{
					Command:  *offer,
					RespChan: respCh,
				}

				ts.consumer <- rpc

				// Wait for response
				select {
				case resp := <-respCh:
					fmt.Println("RPC responde. Writing answer file")
					writeSDP(resp.Response.(webrtc.SessionDescription), ts.answerFile)
					break
				}

				ts.lastOffer = offer

				fmt.Println("forwarded offer")
			}
		}

		time.Sleep(1 * time.Second)
	}
}

// Consumer implements the Signal interface
func (ts *TestSignal) Consumer() <-chan RPC {
	return ts.consumer
}

// Offer implements the Signal interface
func (ts *TestSignal) Offer(target string, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	err := writeSDP(offer, ts.offerFile)
	if err != nil {
		return nil, err
	}

	timeout := time.After(5 * time.Second)
	for {
		select {
		case <-timeout:
			err := fmt.Errorf("Timeout waiting for SDP answer")
			return nil, err
		default:
			answer, err := readSDP(ts.answerFile)
			if err != nil {
				fmt.Printf("Error reading answer file: %v\n", err)
				return nil, err
			}

			if answer != nil {

				if ts.lastAnswer != nil && reflect.DeepEqual(ts.lastAnswer, answer) {
					continue
				}

				ts.lastAnswer = answer

				return answer, nil
			}

			fmt.Println("Answer is empty")

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
