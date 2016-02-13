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
	"bytes"
	"flag"
	"io"
	"log"
	"net/url"
	"os/exec"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func populateStdin(str string) func(io.WriteCloser) {
	return func(stdin io.WriteCloser) {
		defer stdin.Close()
		io.Copy(stdin, bytes.NewBufferString(str))
	}
}

// receiveHandler handles all incoming data from server and runs as a goroutine
func jobHandler(serverConnection *websocket.Conn, wg *sync.WaitGroup) {
	// Ensures channel and waitgroup are closed and done
	defer wg.Done()

	// This is the main loop, in each loop a job is being processed
	for {
		// Send init signal to server
		serverConnection.WriteMessage(websocket.TextMessage, []byte("init"))

		// Waits on incoming messages and processes them, and if client is exiting, exits
		_, bytedata, err := serverConnection.ReadMessage()
		check(err)

		data := strings.Trim(string(bytedata), " ")
		// log.Println("RECV:", data)

		if data != "exit" {
			// Runs the OpenMatrix program that calculates matrix multiplication by OpenCL
			child := exec.Command("./OpenMatrix")

			stdin, err := child.StdinPipe()
			check(err)

			stdout, err := child.StdoutPipe()
			check(err)

			err = child.Start()
			check(err)

			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				defer stdin.Close()
				io.Copy(stdin, bytes.NewBuffer(bytedata))
			}()

			newBuf := bytes.NewBuffer(make([]byte, 0))

			go func() {
				defer wg.Done()
				io.Copy(newBuf, stdout)
			}()

			wg.Wait()
			// Ensures the child has exited
			err = child.Wait()
			check(err)

			// Sends data back to server
			err = serverConnection.WriteMessage(websocket.TextMessage, newBuf.Bytes())
			check(err)
			// log.Println("SENT: ", string(newBuf.Bytes()))
			log.Println("status[handler]: sent the results to server!")

			_, bytedata, err = serverConnection.ReadMessage()
			check(err)
			data = string(bytedata)
		}

		if data == "exit" {
			return
		} else if data != "done" {
			log.Println("Received unexpected data from server")
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

	// disables or enables log
	// log.SetOutput(ioutil.Discard)

	// Creates websocket url that connects to /echo on server
	serverURL := url.URL{Scheme: "ws", Host: *addr, Path: "/"}
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
	go jobHandler(serverConnection, &wg)

	// Waits on all goroutines
	log.Printf("Waiting on goroutines")
	wg.Wait()
}
