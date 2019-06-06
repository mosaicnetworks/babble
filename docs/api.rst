.. _api:

API
===

As mentioned in the :ref:`design` section, Babble communicates with the App 
through an ``AppProxy`` interface, which exposes three methods for Babble to 
call the App. Here we explain how to implement this API. 

**Note**: 
The Snapshot and Restore methods of the API are still work in progress. They are 
necessary for the :ref:`fastsync` protocol which is not completely ready yet. It 
is safe to just implement stubs for these methods.

Inmem
-----

The ``InmemProxy`` uses native callback handlers to enable Babble to call 
methods on the App directly. Applications need only implement the 
``ProxyHandler`` interface and pass that to an ``InmemProxy``.

Here is a quick example of how to use Babble as an in-memory engine (in the same 
process as your handler):

::

  package main
  
  import (
  	"github.com/mosaicnetworks/babble/src/babble"
  	"github.com/mosaicnetworks/babble/src/crypto"
  	"github.com/mosaicnetworks/babble/src/hashgraph"
  	"github.com/mosaicnetworks/babble/src/proxy/inmem"
  )
  
  // Implements proxy.ProxyHandler interface
  type Handler struct {
  	stateHash []byte
  }
  
  // Called when a new block is committed by Babble. This particular example 
  // just computes the stateHash incrementaly with incoming blocks, and accepts
  // all InternalTransactions
  func (h *Handler) CommitHandler(block hashgraph.Block) (stateHash []byte, err error) {
  	hash := h.stateHash
  
  	for _, tx := range block.Transactions() {
  		hash = crypto.SimpleHashFromTwoHashes(hash, crypto.SHA256(tx))
  	}
  
  	h.stateHash = hash
  
    receipts := []hashgraph.InternalTransactionReceipt{}
    for _, it := range block.InternalTransactions() {
    	r := it.AsAccepted()
    	receipts = append(receipts, r)
    }

  	response := proxy.CommitResponse{
  	  StateHash:                   h.stateHash,
  	  InternalTransactionReceipts: receipts,
  	}

    return response, nil
  }
  
  // Called when syncing with the network
  func (h *Handler) SnapshotHandler(blockIndex int) (snapshot []byte, err error) {
  	return []byte{}, nil
  }
  
  // Called when syncing with the network
  func (h *Handler) RestoreHandler(snapshot []byte) (stateHash []byte, err error) {
  	return []byte{}, nil
  }
  
  func NewHandler() *Handler {
  	return &Handler{}
  }
  
  func main() {
  	
  	config := babble.NewDefaultConfig()
  
  	// To use babble as an internal engine we use InmemProxy.
  	proxy := inmem.NewInmemProxy(NewHandler(), config.Logger)
  
  	config.Proxy = proxy
  
  	// Create the engine with the provided config
  	engine := babble.NewBabble(config)
  
  	// Initialize the engine
  	if err := engine.Init(); err != nil {
  		panic(err)
  	}
  
  	// Submit a transaction directly through the Proxy
  	go func() { proxy.SubmitTx([]byte("some content")) }()
  
  	// This is a blocking call
  	engine.Run()
  }

Socket
------

The ``SocketProxy`` is simply a TCP server that accepts `SubmitTx` requests, and 
calls remote methods on the App through a JSON-RPC interface. The App is 
therefore expected to implement its own component to send out SubmitTx 
requests through TCP, and receive JSON-RPC messages from the remote Babble node.

The advantage of using a TCP interface is that it provides the freedom to 
implement the application in any programming language. The specification of the
JSON-RPC interface is provided below, but here is an example of how to use our 
Go implementation, ``SocketBabbleProxy``, to connect to a remote Babble node.

Assuming there is a Babble node running with its proxy listening on 
``127.0.0.1:1338`` and configured to speak to an App at ``127.0.0.1:1339`` 
(these are the default values):

:: 

  package main
  
  import (
  	"time"
  
  	"github.com/mosaicnetworks/babble/src/crypto"
  	"github.com/mosaicnetworks/babble/src/hashgraph"
  	"github.com/mosaicnetworks/babble/src/proxy/socket/babble"
  )
  
  // Implements proxy.ProxyHandler interface
  type Handler struct {
  	stateHash []byte
  }
  
  // Called when a new block is comming. This particular example just computes 
  // the stateHash incrementaly with incoming blocks
  func (h *Handler) CommitHandler(block hashgraph.Block) (stateHash []byte, err error) {
  	hash := h.stateHash
  
  	for _, tx := range block.Transactions() {
  		hash = crypto.SimpleHashFromTwoHashes(hash, crypto.SHA256(tx))
  	}
  
  	h.stateHash = hash
  
  	receipts := []hashgraph.InternalTransactionReceipt{}
    for _, it := range block.InternalTransactions() {
    	r := it.AsAccepted()
    	receipts = append(receipts, r)
    }

  	response := proxy.CommitResponse{
  	  StateHash:                   h.stateHash,
  	  InternalTransactionReceipts: receipts,
  	}

    return response, nil
  }
  
  // Called when syncing with the network
  func (h *Handler) SnapshotHandler(blockIndex int) (snapshot []byte, err error) {
  	return []byte{}, nil
  }
  
  // Called when syncing with the network
  func (h *Handler) RestoreHandler(snapshot []byte) (stateHash []byte, err error) {
  	return []byte{}, nil
  }
  
  func NewHandler() *Handler {
  	return &Handler{}
  }
  
  func main() {
  	// Connect to the babble proxy at :1338 and listen on :1339.
  	// The Handler ties back to the application state.
  	proxy, err := babble.NewSocketBabbleProxy("127.0.0.1:1338", "127.0.0.1:1339", NewHandler(), 1*time.Second, nil)
  
  	// Verify that it can listen
  	if err != nil {
  		panic(err)
  	}
  
  	// Verify that it can connect and submit a transaction
  	if err := proxy.SubmitTx([]byte("some content")); err != nil {
  		panic(err)
  	}
  
  	// Wait indefinitly
  	for {
  		time.Sleep(time.Second)
  	}
  }

Example SubmitTx request (from App to Babble):

::

  request: {"method":"Babble.SubmitTx","params":["Y2xpZW50IDE6IGhlbGxv"],"id":0}
  response: {"id":0,"result":true,"error":null}


Note that the Proxy API is **not** over HTTP; It is raw JSON over TCP. Here is 
an example of how to make a SubmitTx request manually:  

::

  printf "{\"method\":\"Babble.SubmitTx\",\"params\":[\"Y2xpZW50IDE6IGhlbGxv\"],\"id\":0}" | nc -v  172.77.5.1 1338


Example CommitBlock request (from Babble to App):

::
    
  request:
        {
            "method": "State.CommitBlock",
            "params": [
                {
                "Body": {
                    "Index": 0,
                    "RoundReceived": 7,
                    "StateHash": null,
                    "FrameHash": "gdwRCdwxoyLUyzzRK6N31rlJFBJu5By/vDk5gSQHJHQ=",
                    "Transactions": [
                    "Tm9kZTEgVHg5",
                    "Tm9kZTEgVHgx",
                    "Tm9kZTEgVHgy",
                    "Tm9kZTEgVHgz",
                    "Tm9kZTEgVHg0",
                    "Tm9kZTEgVHg1",
                    "Tm9kZTEgVHg2",
                    "Tm9kZTEgVHg3",
                    "Tm9kZTEgVHg4",
                    "Tm9kZTEgVHgxMA=="
                    ]
                },
                "Signatures": {}
                }
            ],
            "id": 0
        }  
  
  response: {"id":0,"result":{"Hash":"6SKQataObI6oSY5n6mvf1swZR3T4Tek+C8yJmGijF00="},"error":null}

The content of the request's "params" is the JSON representation of a Block 
with a RoundReceived of 7 and 10 transactions. The transactions themselves are 
base64 string encodings.

The response's Hash value is the base64 representation of the application's 
State-hash resulting from processing the block's transaction sequentially.