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
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Job type
type Job struct {
	matrixA, matrixB [][]float64
	jobID            int
}

var addr = flag.String("addr", "localhost:8080", "http service address")
var data []byte
var ga, gb [][]float64
var gn int
var upgrader = websocket.Upgrader{} // use default options
var doneJobs = 0
var blockSize = 0
var blockCount = 0
var jobQueue []Job
var jobQueueMaxSize = 512
var totalJobs = 0
var jobResults [][][]float64

///////////////
// UTILITIES //
///////////////

// check function, panics if there is an error
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func pushJob(a, b [][]float64, id int) bool {
	if len(jobQueue) >= jobQueueMaxSize {
		return false
	}
	jobQueue = append(jobQueue, Job{matrixA: a, matrixB: b, jobID: id})
	return true
}

func popJob() (job Job) {
	job, jobQueue = jobQueue[0], jobQueue[1:]
	return job
}

// parseResult reads matrix with size of blockSize * blockSize from a reader
func parseResult(r io.Reader) (int, [][]float64) {
	var size int
	var tmp float64
	m := [][]float64{}
	for i := 0; i < blockSize; i++ {
		row := []float64{}
		for j := 0; j < blockSize; j++ {
			fmt.Fscanf(r, "%f", &tmp)
			row = append(row, tmp)
		}
		fmt.Fscanf(r, "\n")
		m = append(m, row)
	}
	return size, m
}

// parseMatrix reads matrix from a reader, is used to parse the matrix from file
func parseMatrix(r io.Reader) (int, [][]float64, [][]float64) {
	var size int
	var tmp float64
	a, b := [][]float64{}, [][]float64{}
	fmt.Fscanf(r, "%d", &size)
	for i := 0; i < size; i++ {
		row := []float64{}
		for j := 0; j < size; j++ {
			fmt.Fscanf(r, "%f", &tmp)
			row = append(row, tmp)
		}
		fmt.Fscanf(r, "\n")
		a = append(a, row)
	}
	for i := 0; i < size; i++ {
		row := []float64{}
		for j := 0; j < size; j++ {
			fmt.Fscanf(r, "%f", &tmp)
			row = append(row, tmp)
		}
		fmt.Fscanf(r, "\n")
		b = append(b, row)
	}
	return size, a, b
}

// mPlus addes two matrixes
func mPlus(n int, a, b [][]float64) [][]float64 {
	if n > len(a) || n > len(b) {
		log.Println("n:", n, "a:", a, "b:", b)
	}
	res := make([][]float64, n)
	for i := 0; i < n; i++ {
		row := make([]float64, n)
		for j := 0; j < n; j++ {
			row[j] = a[i][j] + b[i][j]
		}
		res[i] = row
	}
	return res
}

// mMinus subtracts two matrixes
func mMinus(n int, a, b [][]float64) [][]float64 {
	res := make([][]float64, n)
	for i := 0; i < n; i++ {
		row := make([]float64, n)
		for j := 0; j < n; j++ {
			row[j] = a[i][j] - b[i][j]
		}
		res[i] = row
	}
	return res
}

// subMatrix func, gets a matrix, and coordinates, and returns the wanted subBlock
func subMatrix(m [][]float64, x, y int) [][]float64 {
	res := make([][]float64, blockSize)
	for i := 0; i < blockSize; i++ {
		res[i] = m[i+y][x : x+blockSize]
	}
	return res
}

// divide function, divides a matrix into 4 pieces to be used in a D&C algorithm
func divide(m [][]float64) ([][]float64, [][]float64, [][]float64, [][]float64) {
	n := len(m)

	m11 := make([][]float64, n/2)
	m12 := make([][]float64, n/2)
	m21 := make([][]float64, n/2)
	m22 := make([][]float64, n/2)

	for i := 0; i < n/2; i++ {
		m11[i] = m[i][:n/2]
		m12[i] = m[i][n/2:]
	}

	for i := n / 2; i < n; i++ {
		m21[i-n/2] = m[i][:n/2]
		m22[i-n/2] = m[i][n/2:]
	}

	return m11, m12, m21, m22
}

// reconstruct function, merges the result of D&C algorithm
func reconstruct(a11, a12, a21, a22 [][]float64) [][]float64 {
	n := len(a11)
	size := n * 2
	res := make([][]float64, size)

	for i := 0; i < n; i++ {
		row := make([]float64, size)
		for j := 0; j < n; j++ {
			row[j] = a11[i][j]
		}
		for j := 0; j < n; j++ {
			row[j+n] = a12[i][j]
		}
		res[i] = row
	}

	for i := 0; i < n; i++ {
		row := make([]float64, size)
		for j := 0; j < n; j++ {
			row[j] = a21[i][j]
		}
		for j := 0; j < n; j++ {
			row[j+n] = a22[i][j]
		}
		res[i+n] = row
	}
	return res
}

// getString function converts 2 matrixes to OpenMatrix stdin data
func getString(a, b [][]float64) string {
	var res bytes.Buffer
	var n = len(a)
	res.WriteString(strconv.Itoa(n))
	res.WriteString(" ")
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			res.WriteString(strconv.FormatFloat(a[i][j], 'f', -1, 32))
			res.WriteString(" ")
		}
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			res.WriteString(strconv.FormatFloat(b[i][j], 'f', -1, 32))
			res.WriteString(" ")
		}
	}

	return res.String()
}

// clientHandler function, handles each client
func clientHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	check(err)
	defer c.Close()
	var job Job

	// sends data to the client
	for {
		_, message, err := c.ReadMessage()
		check(err)

		cmd := string(message)

		if cmd == "init" {
			// Breaks the loop if we don't have any job in the jobQueue
			if len(jobQueue) < 1 {
				log.Println("client connected but there are no jobs in jobQueue!")
				err = c.WriteMessage(websocket.TextMessage, []byte("exit"))
				check(err)
				break
			} else {
				// get suitable matrixes
				job = popJob()
				err = c.WriteMessage(websocket.TextMessage, []byte(getString(job.matrixA, job.matrixB)))
				check(err)
			}
		} else {
			// In this else client has sent something other than 'init'

			_, res := parseResult(strings.NewReader(string(message)))
			if job.jobID >= len(jobResults) {
				log.Println("JobID:", job.jobID, "Len:", len(jobResults))
				log.Println("totalJobs", totalJobs)
			}
			jobResults[job.jobID] = res
			doneJobs++

			log.Println("Total jobs done:", doneJobs)

			if len(jobQueue) < 1 {
				err = c.WriteMessage(websocket.TextMessage, []byte("exit"))
				check(err)
				log.Println("status: no job in jobQueue")
				if doneJobs >= totalJobs {
					asd := strassenMerger(gn, ga, gb, 0)
					for i := 0; i < len(asd); i++ {
						for j := 0; j < len(asd[i]); j++ {
							fmt.Print(asd[i][j], " ")
						}
						fmt.Println()
					}
				}
				break
			} else {
				err = c.WriteMessage(websocket.TextMessage, []byte("done"))
				check(err)
			}
		}
	}
}

// mMult func, simple D&C algorithm to do matrix multiplication
func mMult(n int, a, b [][]float64) [][]float64 {
	if n == 1 {
		return [][]float64{{a[0][0] * b[0][0]}}
	}

	a11, a12, a21, a22 := divide(a)
	b11, b12, b21, b22 := divide(b)

	x1 := mMult(n/2, a11, b11)
	x2 := mMult(n/2, a12, b21)
	x3 := mMult(n/2, a11, b12)
	x4 := mMult(n/2, a12, b22)
	x5 := mMult(n/2, a21, b11)
	x6 := mMult(n/2, a22, b21)
	x7 := mMult(n/2, a21, b12)
	x8 := mMult(n/2, a22, b22)

	c11 := mPlus(n/2, x1, x2)
	c12 := mPlus(n/2, x3, x4)
	c21 := mPlus(n/2, x5, x6)
	c22 := mPlus(n/2, x7, x8)

	res := reconstruct(c11, c12, c21, c22)
	return res
}

// strassen Function, recursively computes matrix multiplication
func strassen(n int, a, b [][]float64) [][]float64 {
	if n == blockSize {
		return [][]float64{{a[0][0] * b[0][0]}}
	}

	a11, a12, a21, a22 := divide(a)
	b11, b12, b21, b22 := divide(b)

	p1 := strassen(n/2, a11, mMinus(n/2, b12, b22))
	p2 := strassen(n/2, mPlus(n/2, a11, a12), b22)
	p3 := strassen(n/2, mPlus(n/2, a21, a22), b11)
	p4 := strassen(n/2, a22, mMinus(n/2, b21, b11))
	p5 := strassen(n/2, mPlus(n/2, a11, a22), mPlus(n/2, b11, b22))
	p6 := strassen(n/2, mMinus(n/2, a12, a22), mPlus(n/2, b21, b22))
	p7 := strassen(n/2, mMinus(n/2, a11, a21), mPlus(n/2, b11, b12))

	c11 := mPlus(n/2, mMinus(n/2, mPlus(n/2, p5, p4), p2), p6)
	c12 := mPlus(n/2, p1, p2)
	c21 := mPlus(n/2, p3, p4)
	c22 := mMinus(n/2, mMinus(n/2, mPlus(n/2, p1, p5), p3), p7)

	res := reconstruct(c11, c12, c21, c22)
	return res
}

// strassenJobAdder Function, recursively adds strassen jobs to jobQueue
func strassenJobAdder(n int, a, b [][]float64, id int) {
	if n == blockSize {
		for pushJob(a, b, id) == false {
			time.Sleep(2 * time.Millisecond)
		}
		return
	}

	id *= 7

	a11, a12, a21, a22 := divide(a)
	b11, b12, b21, b22 := divide(b)

	strassenJobAdder(n/2, a11, mMinus(n/2, b12, b22), id+1)
	strassenJobAdder(n/2, mPlus(n/2, a11, a12), b22, id+2)
	strassenJobAdder(n/2, mPlus(n/2, a21, a22), b11, id+3)
	strassenJobAdder(n/2, a22, mMinus(n/2, b21, b11), id+4)
	strassenJobAdder(n/2, mPlus(n/2, a11, a22), mPlus(n/2, b11, b22), id+5)
	strassenJobAdder(n/2, mMinus(n/2, a12, a22), mPlus(n/2, b21, b22), id+6)
	strassenJobAdder(n/2, mMinus(n/2, a11, a21), mPlus(n/2, b11, b12), id+7)
}

func strassenMerger(n int, a, b [][]float64, id int) [][]float64 {
	if n == blockSize {
		return jobResults[id]
	}

	id *= 7

	a11, a12, a21, a22 := divide(a)
	b11, b12, b21, b22 := divide(b)

	p1 := strassenMerger(n/2, a11, mMinus(n/2, b12, b22), id+1)
	p2 := strassenMerger(n/2, mPlus(n/2, a11, a12), b22, id+2)
	p3 := strassenMerger(n/2, mPlus(n/2, a21, a22), b11, id+3)
	p4 := strassenMerger(n/2, a22, mMinus(n/2, b21, b11), id+4)
	p5 := strassenMerger(n/2, mPlus(n/2, a11, a22), mPlus(n/2, b11, b22), id+5)
	p6 := strassenMerger(n/2, mMinus(n/2, a12, a22), mPlus(n/2, b21, b22), id+6)
	p7 := strassenMerger(n/2, mMinus(n/2, a11, a21), mPlus(n/2, b11, b12), id+7)

	c11 := mPlus(n/2, mMinus(n/2, mPlus(n/2, p5, p4), p2), p6)
	c12 := mPlus(n/2, p1, p2)
	c21 := mPlus(n/2, p3, p4)
	c22 := mMinus(n/2, mMinus(n/2, mPlus(n/2, p1, p5), p3), p7)

	res := reconstruct(c11, c12, c21, c22)
	return res
}

func main() {
	var err error

	flag.Parse()
	log.SetFlags(1)
	blockSize = 64

	file, err := os.Open("64.in")
	check(err)

	gn, ga, gb = parseMatrix(file)
	blockCount = gn / blockSize

	power := math.Log2(float64(blockCount))
	memCap := 1
	totalJobs = 1
	for i := 0; i < int(power); i++ {
		totalJobs *= 7
		memCap += totalJobs
	}
	jobResults = make([][][]float64, memCap)

	go strassenJobAdder(gn, ga, gb, 0)

	// res := strassen(n, a, b)
	// log.Println(res)

	// data, err = ioutil.ReadFile("32.in")
	// check(err)

	http.HandleFunc("/", clientHandler)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
