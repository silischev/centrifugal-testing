package common

import (
	"log"
	"strconv"
	"sync/atomic"

	"github.com/centrifugal/centrifuge-go"

	"github.com/golang-jwt/jwt"
)

var userID uint64

const tokenHmacSecret = "test" // see config centrifugo.json

type EventHandler struct{}

func (h *EventHandler) OnError(_ *centrifuge.Client, _ centrifuge.ErrorEvent) {}

func (h *EventHandler) OnDisconnect(_ *centrifuge.Client, e centrifuge.DisconnectEvent) {
	if e.Reason != "clean disconnect" {
		log.Printf("disconnect: %s", e.Reason)
	}
}

func connToken(user string, exp int64) string {
	claims := jwt.MapClaims{"sub": user}
	if exp > 0 {
		claims["exp"] = exp
	}

	t, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(tokenHmacSecret))
	if err != nil {
		log.Fatalln(err)
	}

	return t
}

func NewConnection(url, format string) *centrifuge.Client {
	var c *centrifuge.Client
	if format == "protobuf" {
		c = centrifuge.NewProtobufClient(url, centrifuge.DefaultConfig())
	} else {
		c = centrifuge.NewJsonClient(url, centrifuge.DefaultConfig())
	}

	id := atomic.AddUint64(&userID, 1)
	c.SetToken(connToken(strconv.FormatUint(id, 10), 0))

	events := &EventHandler{}
	c.OnError(events)
	c.OnDisconnect(events)
	return c
}
