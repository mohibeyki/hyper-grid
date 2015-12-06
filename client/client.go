// This file is part of Hyper-Grid.
//
// Hyper-Grid is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// Hyper-Grid is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Hyper-Grid.  If not, see <http://www.gnu.org/licenses/>.
//
// This file is created by Mohi Beyki <mohibeyki@gmail.com>
//

package main

import (
	"bufio"
	"flag"
	"log"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var running = true

// receiveHandler handles all incoming data from server and runs as a goroutine
func receiveHandler(connection *websocket.Conn, wg *sync.WaitGroup) {
	// Ensures channel and waitgroup are closed and done
	defer wg.Done()

	// Waits on incoming messages and processes them, and if client is exiting, exits
	for running {
		_, message, err := connection.ReadMessage()
		if err != nil {
			log.Printf("error[read]: %s", err)
			break
		}
		log.Printf("received: %s", message)
		if string(message) == "exit" {
			break
		}
	}
}

// Creates a ticker and sends a dummy msg on each tick
func dummySender(connection *websocket.Conn, wg *sync.WaitGroup) {
	// Creates a ticker
	ticker := time.NewTicker(time.Second)

	// Ensures ticker and waitgroup are stoped and done
	defer ticker.Stop()
	defer wg.Done()

	// For each tick (every second)
	for t := range ticker.C {
		if !running {
			break
		}

		msg := "dummySender @ " + t.String()
		err := connection.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			log.Printf("error[dummySender]: %s", err)
			break
		}
	}
}

func sendMessage(connection *websocket.Conn, msg string) {
	err := connection.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		log.Printf("error[sendMessage]: %s", err)
	}
}

func inputHandler(connection *websocket.Conn, wg *sync.WaitGroup) {
	// Ensures waitgroup is done
	defer wg.Done()

	// Handles input
	scanner := bufio.NewScanner(os.Stdin)

	// While it had input on scanner
	for scanner.Scan() {
		sendMessage(connection, scanner.Text())

		// Should terminate the loop on exit command
		if scanner.Text() == "exit" {
			// Sets running to false, not sure if its a good thing yet, used in dummySender
			running = false
			log.Printf("Exiting...")
			break
		}
	}
}

// Main function :)
func main() {
	// Create a WaitGroup to wait on goroutines
	var wg sync.WaitGroup

	// Getting data from flag
	flag.Parse()
	log.SetFlags(0)
	// log.SetOutput(ioutil.Discard)

	// Creates websocket url that connects to /echo on server
	serverURL := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	log.Printf("status[main]: connecting to %s", serverURL.String())

	// Dials server with serverURL and handles errors
	serverConnection, _, err := websocket.DefaultDialer.Dial(serverURL.String(), nil)
	if err != nil {
		log.Fatalf("error[main]: dial %s", err)
	}
	// Ensures that the server connection is closed when the program exits
	defer serverConnection.Close()

	// Calls a receiveHandler and adds it to the waitgroup
	wg.Add(1)
	go receiveHandler(serverConnection, &wg)

	// Calls a dummySender and adds it to the waitgroup
	wg.Add(1)
	go dummySender(serverConnection, &wg)

	wg.Add(1)
	go inputHandler(serverConnection, &wg)

	log.Printf("Waiting on goroutines")

	// Waits on all goroutines
	wg.Wait()
}
