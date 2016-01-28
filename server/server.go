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
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var data []byte
var upgrader = websocket.Upgrader{} // use default options
var count = 0

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func clientHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	check(err)
	defer c.Close()

	// sends data to the client
	for {
		_, message, err := c.ReadMessage()
		check(err)

		cmd := string(message)
		// log.Println("Received: ", cmd)

		if cmd == "init" {
			err = c.WriteMessage(websocket.TextMessage, data)
			check(err)
		} else {
			str := "done"
			if count >= 0 {
				str = "exit"
			}
			log.Println("Count: ", count)
			err = c.WriteMessage(websocket.TextMessage, []byte(str))
			check(err)
			if count >= 0 {
				count = 0
				break
			}
			count = count + 1
		}
	}
}

func main() {
	var err error

	flag.Parse()
	log.SetFlags(1)

	data, err = ioutil.ReadFile("2048.in")
	check(err)

	http.HandleFunc("/", clientHandler)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
