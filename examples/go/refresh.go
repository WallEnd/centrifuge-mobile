package main

// Demonstrate how to resque from connection expiration.

import (
	"fmt"
	"log"

	"github.com/centrifugal/centrifuge-mobile"
)

// In production you need to receive credentials from application backend.
func credentials() *centrifuge.Credentials {
	// Never show secret to client of your application. Keep it on your application backend only.
	secret := "secret"
	// Application user ID.
	user := "42"
	// Exp as string.
	exp := centrifuge.Exp(60)
	// Empty info.
	info := ""
	// Generate sign so Centrifugo server can trust connection parameters received from client.
	sign := centrifuge.GenerateClientSign(secret, user, exp, info)

	return &centrifuge.Credentials{
		User: user,
		Exp:  exp,
		Info: info,
		Sign: sign,
	}
}

type eventHandler struct{}

func (h *eventHandler) OnConnect(c *centrifuge.Client, ctx *centrifuge.ConnectContext) {
	log.Println("Connected")
}

func (h *eventHandler) OnDisconnect(c *centrifuge.Client, ctx *centrifuge.DisconnectContext) {
	log.Println("Disconnected")
}

func (h *eventHandler) OnRefresh(c *centrifuge.Client) (*centrifuge.Credentials, error) {
	log.Println("Refresh")
	return credentials(), nil
}

type subEventHandler struct{}

func (h *subEventHandler) OnMessage(sub *centrifuge.Sub, msg *centrifuge.Message) {
	log.Println(fmt.Sprintf("New message received in channel %s: %#v", sub.Channel(), msg))
}

func newConnection() *centrifuge.Client {
	creds := credentials()
	wsURL := "ws://localhost:8000/connection/websocket"

	handler := &eventHandler{}

	events := centrifuge.NewEventHandler()
	events.OnDisconnect(handler)
	events.OnRefresh(handler)
	events.OnConnect(handler)

	c := centrifuge.New(wsURL, creds, events, centrifuge.DefaultConfig())

	err := c.Connect()
	if err != nil {
		log.Fatalln(err)
	}

	subEvents := centrifuge.NewSubEventHandler()
	subEvents.OnMessage(&subEventHandler{})

	_, err = c.Subscribe("public:chat", subEvents)
	if err != nil {
		log.Fatalln(err)
	}

	return c
}

func main() {
	log.Println("Start program")
	newConnection()
	select {}
}
