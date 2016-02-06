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
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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
			if count >= 100 {
				str = "exit"
			}
			log.Println("Count: ", count)
			err = c.WriteMessage(websocket.TextMessage, []byte(str))
			check(err)
			if count >= 100 {
				count = 0
				break
			}
			count = count + 1
		}
	}
}

// ParseMatrix reads ints from a reader
func ParseMatrix(r io.Reader) (int, [][]int, [][]int) {
	var size, tmp int
	a, b := [][]int{}, [][]int{}
	fmt.Fscanf(r, "%d", &size)
	for i := 0; i < size; i++ {
		row := []int{}
		for j := 0; j < size; j++ {
			fmt.Fscanf(r, "%d", &tmp)
			row = append(row, tmp)
		}
		a = append(a, row)
	}
	for i := 0; i < size; i++ {
		row := []int{}
		for j := 0; j < size; j++ {
			fmt.Fscanf(r, "%d", &tmp)
			row = append(row, tmp)
		}
		b = append(b, row)
	}
	return size, a, b
}

// MPlus addes two matrixes
func MPlus(n int, a, b [][]int) [][]int {
	res := [][]int{}
	for i := 0; i < n; i++ {
		row := []int{}
		for j := 0; j < n; j++ {
			row = append(row, a[i][j]+b[i][j])
		}
		res = append(res, row)
	}
	return res
}

func divide(m [][]int) ([][]int, [][]int, [][]int, [][]int) {
	n := len(m)

	m11 := [][]int{}
	m12 := [][]int{}
	m21 := [][]int{}
	m22 := [][]int{}

	for i := 0; i < n/2; i++ {
		m11 = append(m11, m[i][:n/2])
		m12 = append(m12, m[i][n/2:])
	}

	for i := n / 2; i < n; i++ {
		m21 = append(m21, m[i][:n/2])
		m22 = append(m22, m[i][n/2:])
	}
	return m11, m12, m21, m22
}

func reconstruct(a11, a12, a21, a22 [][]int) [][]int {
	res := [][]int{}
	n := len(a11)
	for i := 0; i < n; i++ {
		row := []int{}
		row = append(row, a11[i]...)
		row = append(row, a12[i]...)
		res = append(res, row)
	}

	for i := 0; i < n; i++ {
		row := []int{}
		row = append(row, a21[i]...)
		row = append(row, a22[i]...)
		res = append(res, row)
	}
	return res
}

// MMult Function, recursively computes matrix multiplication
func MMult(n int, a, b [][]int) [][]int {
	if n == 1 {
		return [][]int{{a[0][0] * b[0][0]}}
	}

	a11, a12, a21, a22 := divide(a)
	// fmt.Println("A is")
	// fmt.Println(a11)
	// fmt.Println(a12)
	// fmt.Println(a21)
	// fmt.Println(a22)

	b11, b12, b21, b22 := divide(b)
	// fmt.Println("B is")
	// fmt.Println(b11)
	// fmt.Println(b12)
	// fmt.Println(b21)
	// fmt.Println(b22)

	x1 := MMult(n/2, a11, b11)
	x2 := MMult(n/2, a12, b21)
	x3 := MMult(n/2, a11, b12)
	x4 := MMult(n/2, a12, b22)
	x5 := MMult(n/2, a21, b11)
	x6 := MMult(n/2, a22, b21)
	x7 := MMult(n/2, a21, b12)
	x8 := MMult(n/2, a22, b22)

	c11 := MPlus(n/2, x1, x2)
	c12 := MPlus(n/2, x3, x4)
	c21 := MPlus(n/2, x5, x6)
	c22 := MPlus(n/2, x7, x8)

	// fmt.Println("C is")
	// fmt.Println(c11)
	// fmt.Println(c12)
	// fmt.Println(c21)
	// fmt.Println(c22)

	res := reconstruct(c11, c12, c21, c22)
	return res
}

func main() {
	var err error

	flag.Parse()
	log.SetFlags(1)

	file, err := os.Open("512.in")
	check(err)

	n, a, b := ParseMatrix(file)

	// fmt.Println("Size is:", n)
	// fmt.Println(a)
	// fmt.Println(b)

	_ = MMult(n, a, b)
	// fmt.Println(res)

	// data, err = ioutil.ReadFile("32.in")
	// check(err)

	// http.HandleFunc("/", clientHandler)
	// log.Fatal(http.ListenAndServe(*addr, nil))
}
