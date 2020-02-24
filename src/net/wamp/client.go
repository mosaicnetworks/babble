package wamp

import (
	"context"
	"fmt"
	"log"

	"github.com/gammazero/nexus/v3/client"
	"github.com/gammazero/nexus/v3/wamp"
)

type Client struct {
	name   string
	client *client.Client
}

func NewClient(server string, realm string, name string) (*Client, error) {
	cfg := client.Config{
		Realm: realm,
	}

	cli, err := client.ConnectNet(context.Background(), fmt.Sprintf("ws://%s", server), cfg)
	if err != nil {
		return nil, err
	}

	res := &Client{
		name:   name,
		client: cli,
	}

	res.Listen()

	return res, nil
}

func (c *Client) Listen() {
	proc := fmt.Sprintf("foo_%s", c.name)

	go func() {
		// Register procedure "foo"
		if err := c.client.Register(proc, foo, nil); err != nil {
			log.Fatal("Failed to register procedure:", err)
		}
		log.Println("Registered procedure foo with router")

	}()
}

func (c *Client) Call(to string, args ...string) error {

	callArgs := make(wamp.List, len(args))
	for i, a := range args {
		callArgs[i] = a
	}

	proc := fmt.Sprintf("foo_%s", to)

	log.Println("Call remote procedure", proc)

	result, err := c.client.Call(context.Background(), proc, nil, callArgs, nil, nil)
	if err != nil {
		log.Fatal(err)
		return err
	}

	for i := range result.Arguments {
		txt, _ := wamp.AsString(result.Arguments[i])
		log.Println(txt)
	}

	return nil

}

func (c *Client) Close() error {
	return c.client.Close()
}

func foo(ctx context.Context, inv *wamp.Invocation) client.InvokeResult {
	log.Println("In foo")

	msg := "bar"

	for _, arg := range inv.Arguments {
		s, ok := wamp.AsString(arg)
		if ok {
			msg = fmt.Sprintf("%s %s", msg, s)
		}
	}

	log.Println(msg)

	return client.InvokeResult{
		Args: wamp.List{msg},
	}
}
