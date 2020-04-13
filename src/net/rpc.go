package net

// RPCResponse captures both a response and a potential error.
type RPCResponse struct {
	Response interface{}
	Error    error
}

// RPC encapsulates an RPC request and provides a response mechanism.
type RPC struct {
	Command  interface{}
	RespChan chan<- RPCResponse
}

// Respond is used to respond with a response, error or both.
func (r *RPC) Respond(resp interface{}, err error) {
	r.RespChan <- RPCResponse{resp, err}
}
