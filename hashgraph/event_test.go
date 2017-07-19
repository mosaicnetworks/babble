/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package hashgraph

import (
	"reflect"
	"testing"
	"time"

	"github.com/babbleio/babble/crypto"
)

func createDummyEventBody() EventBody {
	body := EventBody{}
	body.Transactions = [][]byte{[]byte("abc"), []byte("def")}
	body.Parents = []string{"self", "other"}
	body.Creator = []byte("public key")
	body.Timestamp = time.Now()
	return body
}

func TestMarshallBody(t *testing.T) {
	body := createDummyEventBody()

	raw, err := body.Marshal()
	if err != nil {
		t.Fatalf("Error marshalling EventBody: %s", err)
	}

	newBody := new(EventBody)
	if err := newBody.Unmarshal(raw); err != nil {
		t.Fatalf("Error unmarshalling EventBody: %s", err)
	}

	if !reflect.DeepEqual(body.Transactions, newBody.Transactions) {
		t.Fatalf("Payloads do not match. Expected %#v, got %#v", body.Transactions, newBody.Transactions)
	}
	if !reflect.DeepEqual(body.Parents, newBody.Parents) {
		t.Fatalf("Parents do not match. Expected %#v, got %#v", body.Parents, newBody.Parents)
	}
	if !reflect.DeepEqual(body.Creator, newBody.Creator) {
		t.Fatalf("Creators do not match. Expected %#v, got %#v", body.Creator, newBody.Creator)
	}
	if body.Timestamp != newBody.Timestamp {
		t.Fatalf("Timestamps do not match. Expected %#v, got %#v", body.Timestamp, newBody.Timestamp)
	}

}

func TestSignEvent(t *testing.T) {
	privateKey, _ := crypto.GenerateECDSAKey()
	publicKeyBytes := crypto.FromECDSAPub(&privateKey.PublicKey)

	body := createDummyEventBody()
	body.Creator = publicKeyBytes

	event := Event{Body: body}
	if err := event.Sign(privateKey); err != nil {
		t.Fatalf("Error signing Event: %s", err)
	}

	res, err := event.Verify()
	if err != nil {
		t.Fatalf("Error verifying signature: %s", err)
	}
	if !res {
		t.Fatalf("Verify returned false")
	}
}

func TestMarshallEvent(t *testing.T) {
	privateKey, _ := crypto.GenerateECDSAKey()
	publicKeyBytes := crypto.FromECDSAPub(&privateKey.PublicKey)

	body := createDummyEventBody()
	body.Creator = publicKeyBytes

	event := Event{Body: body}
	if err := event.Sign(privateKey); err != nil {
		t.Fatalf("Error signing Event: %s", err)
	}

	raw, err := event.Marshal()
	if err != nil {
		t.Fatalf("Error marshalling Event: %s", err)
	}

	newEvent := new(Event)
	if err := newEvent.Unmarshal(raw); err != nil {
		t.Fatalf("Error unmarshalling Event: %s", err)
	}

	if !reflect.DeepEqual(*newEvent, event) {
		t.Fatalf("Events are not deeply equal")
	}
}

func TestWireEvent(t *testing.T) {
	privateKey, _ := crypto.GenerateECDSAKey()
	publicKeyBytes := crypto.FromECDSAPub(&privateKey.PublicKey)

	body := createDummyEventBody()
	body.Creator = publicKeyBytes

	event := Event{Body: body}
	if err := event.Sign(privateKey); err != nil {
		t.Fatalf("Error signing Event: %s", err)
	}

	event.SetWireInfo(1, 66, 2, 67)

	expectedWireEvent := WireEvent{
		Body: WireBody{
			Transactions:         event.Body.Transactions,
			SelfParentIndex:      1,
			OtherParentCreatorID: 66,
			OtherParentIndex:     2,
			CreatorID:            67,
			Timestamp:            event.Body.Timestamp,
			Index:                event.Body.Index,
		},
		R: event.R,
		S: event.S,
	}

	wireEvent := event.ToWire()

	if !reflect.DeepEqual(expectedWireEvent, wireEvent) {
		t.Fatalf("WireEvent should be %#v, not %#v", expectedWireEvent, wireEvent)
	}
}

func TestIsLoaded(t *testing.T) {
	//nil payload
	event := NewEvent(nil, []string{"p1", "p2"}, []byte("creator"), 1)
	if event.IsLoaded() {
		t.Fatalf("IsLoaded() should return false for nil Body.Transactions")
	}

	//empty payload
	event.Body.Transactions = [][]byte{}
	if event.IsLoaded() {
		t.Fatalf("IsLoaded() should return false for empty Body.Transactions")
	}

	//initial event
	event.Body.Index = 0
	if !event.IsLoaded() {
		t.Fatalf("IsLoaded() should return true for initial event")
	}

	//non-empty payload
	event.Body.Transactions = [][]byte{[]byte("abc")}
	if !event.IsLoaded() {
		t.Fatalf("IsLoaded() should return true for non-empty payload")
	}
}
