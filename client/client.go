package main

import (
	"fmt"
	"log"

	"golang.org/x/net/websocket"
)

var origin = "http://localhost/"
var url = "ws://localhost:8080/register"

func main() {
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	message := []byte("hello, world!")
	_, err = ws.Write(message)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Send: %s\n", message)

	for {
		var msg = make([]byte, 512)
		_, err = ws.Read(msg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Receive: %s\n", msg)

		// reader := bufio.NewReader(os.Stdin)
		// text, _ := reader.ReadString('\n')
		//
		// _, err = ws.Write([]byte(text))
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// fmt.Printf("Send: %s\n", message)
	}
}
