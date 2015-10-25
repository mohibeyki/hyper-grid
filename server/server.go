package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

var clients = make([]*websocket.Conn, 0)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func printBinary(s []byte) {
	fmt.Printf("Received b:")
	for n := 0; n < len(s); n++ {
		fmt.Printf("%d,", s[n])
	}
	fmt.Printf("\n")
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		printBinary(p)

		err = conn.WriteMessage(messageType, p)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	messageType, p, err := conn.ReadMessage()
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("A client has connected!")
	// log.Println(messageType)

	clients = append(clients, conn)

	err = conn.WriteMessage(messageType, p)
	if err != nil {
		log.Println(err)
		return
	}
	i := 0
	for {
		xar(i)
		i = i + 1
	}
}

func xar(i int) {
	log.Println("In XAR")
	for c := range clients {
		// log.Println(c)
		s := "Do You GET THIS? " + strconv.Itoa(i)
		clients[c].WriteMessage(1, []byte(s))
	}
}

// func (h *types.Hub) run() {
// 	for {
// 		select {
// 		case c := <-h.register:
// 			h.clients[c] = true
// 			c.send <- []byte(h.content)
// 			break
//
// 		case c := <-h.unregister:
// 			_, ok := h.clients[c]
// 			if ok {
// 				delete(h.clients, c)
// 				close(c.send)
// 			}
// 			break
//
// 		case m := <-h.broadcast:
// 			h.content = m
// 			h.broadcastMessage()
// 			break
// 		}
// 	}
// }

func main() {
	http.HandleFunc("/echo", echoHandler)
	http.HandleFunc("/register", registerHandler)
	http.Handle("/", http.FileServer(http.Dir(".")))
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic("Error: " + err.Error())
	}
}
