package hashgraph

import (
	"reflect"
	"testing"

	"github.com/mosaicnetworks/babble/src/crypto/keys"
)

func createDummyEventBody() EventBody {
	body := EventBody{}
	body.Transactions = [][]byte{[]byte("abc"), []byte("def")}
	body.InternalTransactions = []InternalTransaction{}
	body.Parents = []string{"self", "other"}
	body.Creator = []byte("public key")
	body.BlockSignatures = []BlockSignature{
		{
			Validator: body.Creator,
			Index:     0,
			Signature: "r|s",
		},
	}
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
		t.Fatalf("Transactions do not match. Expected %#v, got %#v", body.Transactions, newBody.Transactions)
	}
	if !reflect.DeepEqual(body.InternalTransactions, newBody.InternalTransactions) {
		t.Fatalf("Internal Transactions do not match. Expected %#v, got %#v", body.InternalTransactions, newBody.InternalTransactions)
	}
	if !reflect.DeepEqual(body.BlockSignatures, newBody.BlockSignatures) {
		t.Fatalf("BlockSignatures do not match. Expected %#v, got %#v", body.BlockSignatures, newBody.BlockSignatures)
	}
	if !reflect.DeepEqual(body.Parents, newBody.Parents) {
		t.Fatalf("Parents do not match. Expected %#v, got %#v", body.Parents, newBody.Parents)
	}
	if !reflect.DeepEqual(body.Creator, newBody.Creator) {
		t.Fatalf("Creators do not match. Expected %#v, got %#v", body.Creator, newBody.Creator)
	}

}

func TestSignEvent(t *testing.T) {
	privateKey, _ := keys.GenerateECDSAKey()
	publicKeyBytes := keys.FromPublicKey(&privateKey.PublicKey)

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
	privateKey, _ := keys.GenerateECDSAKey()
	publicKeyBytes := keys.FromPublicKey(&privateKey.PublicKey)

	body := createDummyEventBody()
	body.Creator = publicKeyBytes

	event := Event{Body: body}
	if err := event.Sign(privateKey); err != nil {
		t.Fatalf("Error signing Event: %s", err)
	}

	raw, err := event.MarshalDB()
	if err != nil {
		t.Fatalf("Error marshalling Event: %s", err)
	}

	newEvent := new(Event)
	if err := newEvent.UnmarshalDB(raw); err != nil {
		t.Fatalf("Error unmarshalling Event: %s", err)
	}

	if !reflect.DeepEqual(*newEvent, event) {
		t.Fatalf("Events are not deeply equal")
	}
}

func TestWireEvent(t *testing.T) {
	privateKey, _ := keys.GenerateECDSAKey()
	publicKeyBytes := keys.FromPublicKey(&privateKey.PublicKey)

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
			InternalTransactions: event.Body.InternalTransactions,
			SelfParentIndex:      1,
			OtherParentCreatorID: 66,
			OtherParentIndex:     2,
			CreatorID:            67,
			Index:                event.Body.Index,
			BlockSignatures:      event.WireBlockSignatures(),
		},
		Signature: event.Signature,
	}

	wireEvent := event.ToWire()

	if !reflect.DeepEqual(expectedWireEvent, wireEvent) {
		t.Fatalf("WireEvent should be %#v, not %#v", expectedWireEvent, wireEvent)
	}
}

func TestIsLoaded(t *testing.T) {
	//nil payload
	event := NewEvent(nil, nil, nil, []string{"p1", "p2"}, []byte("creator"), 1)
	if event.IsLoaded() {
		t.Fatalf("IsLoaded() should return false for nil Body.Transactions and Body.BlockSignatures")
	}

	//empty payload
	event.Body.Transactions = [][]byte{}
	if event.IsLoaded() {
		t.Fatalf("IsLoaded() should return false for empty Body.Transactions")
	}

	event.Body.BlockSignatures = []BlockSignature{}
	if event.IsLoaded() {
		t.Fatalf("IsLoaded() should return false for empty Body.BlockSignatures")
	}

	//initial event
	event.Body.Index = 0
	if !event.IsLoaded() {
		t.Fatalf("IsLoaded() should return true for initial event")
	}

	//non-empty tx payload
	event.Body.Transactions = [][]byte{[]byte("abc")}
	if !event.IsLoaded() {
		t.Fatalf("IsLoaded() should return true for non-empty transaction payload")
	}

	//non-empy signature payload
	event.Body.Transactions = nil
	event.Body.BlockSignatures = []BlockSignature{{Validator: []byte("validator"), Index: 0, Signature: "r|s"}}
	if !event.IsLoaded() {
		t.Fatalf("IsLoaded() should return true for non-empty signature payload")
	}
}
