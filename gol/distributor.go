package gol

import (
	"fmt"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

var wg, pause, executingKeyPress, pauseReached sync.WaitGroup

// var mu sync.Mutex
var turn IntContainer

var worldGlobal, worldOld [][]uint8

type BoolContainer struct {
	mu      sync.Mutex
	boolean bool
}

type IntContainer struct {
	mu  sync.Mutex
	num int
}

type WorldContainer struct {
	mu    sync.Mutex
	world [][]uint8
}

func (c *IntContainer) inc() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.num++
}

func (c *IntContainer) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.num = 0
}

func (c *IntContainer) get() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.num
}

func (c *BoolContainer) setFalse() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.boolean = false
}

func (c *BoolContainer) setTrue() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.boolean = true
}

func (c *BoolContainer) get() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.boolean
}

var end, snapshot, getCount BoolContainer

const CellAlive = 255
const CellDead = 0

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- [][]uint8
	ioInput    <-chan [][]uint8
	keyPresses <-chan rune
}

type pixel struct {
	X     int
	Y     int
	Value uint8
}

type suint8 struct {
	x   int
	val []uint8
}

type neighbourPixel struct {
	X    int
	Y    int
	quit bool
}

func getAliveCells(p Params, world [][]uint8) []util.Cell {
	// make the slice
	aliveCells := make([]util.Cell, 0)

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			// append every CellAlive cells
			if world[x][y] == CellAlive {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}

	return aliveCells
}

func getNumAliveCells(p Params, world [][]uint8) int {
	num := 0
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			// append every CellAlive cells
			if world[x][y] == CellAlive {
				num++
			}
		}
	}

	return num
}

func makeOutput(p Params, c distributorChannels, world [][]uint8) {

	// add a get output to the command channel
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, p.Turns)

	c.ioOutput <- world

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
}

func makeOutputTurnWithTurnNum(p Params, c distributorChannels, turns int, world [][]uint8) {

	// add a get output to the command channel
	c.ioCommand <- ioOutput
	filename := fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, turns)
	c.ioFilename <- filename

	c.ioOutput <- world

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- ImageOutputComplete{turns, filename}
}

func calculateNewAliveParallel(p Params, workerNum int, c distributorChannels, world [][]uint8) [][]uint8 {
	//numRows := p.ImageHeight / workerNum

	splitSegments := make([]chan pixel, workerNum)
	for i := range splitSegments {
		splitSegments[i] = make(chan pixel, p.ImageWidth*p.ImageWidth)
	}
	// start workers to make the world
	//startWorkers(workerNum, numRows, p, dataChan, neighbourChan, c)
	setupWorkers(p.ImageHeight, workerNum, splitSegments, world)

	var cells []util.Cell

	wg.Wait()
	for i := 0; i < workerNum; i++ {
		length := len(splitSegments[i])
		for j := 0; j < length; j++ {
			item := <-splitSegments[i]
			world[item.X][item.Y] = item.Value
			cells = append(cells, util.Cell{X: item.X, Y: item.Y})
		}
	}
	c.events <- CellsFlipped{turn.get() + 1, cells}
	//for i := 0; i < workerNum; i++ {
	//	//temp := <-splitSegments[i]
	//	//for _, item := range temp {
	//	//	fmt.Println(item)
	//	//}
	//	newWorld = append(newWorld, <-splitSegments[i]...)
	//}

	return world
}
func setupWorkers(size, workerNum int, splitSegments []chan pixel, world [][]uint8) {
	numRows := size / workerNum

	i := 0
	for i < workerNum-1 {
		wg.Add(1)
		go runWorker(size, i*numRows, (i+1)*numRows, splitSegments[i], world)
		i++
	}

	// final worker does the remaining rows
	wg.Add(1)
	go runWorker(size, i*numRows, size, splitSegments[i], world)
}
func runWorker(size, start, end int, splitSegment chan pixel, world [][]uint8) {
	defer wg.Done()
	calculateNextWorld(start, end, size, world, splitSegment)
}

func calculateNextWorld(start, end, width int, world [][]uint8, c chan pixel) {
	//newWorld := make([][]uint8, end-start)
	//for i := 0; i < end-start; i++ {
	//	newWorld[i] = world[i]
	//}

	neighboursWorld := calculateNeighbours(start, end, width, world)
	//for _, item := range neighboursWorld {
	//	fmt.Println(item)
	//}

	for y := start; y < end; y++ {
		for x := 0; x < width; x++ {
			neighbors := neighboursWorld[y][x]
			if (neighbors < 2 || neighbors > 3) && world[y][x] == 255 {
				c <- pixel{y, x, 0}
			} else if neighbors == 3 && world[y][x] == 0 {
				c <- pixel{y, x, 255}
			}
		}
	}
}

func calculateNeighbours(start, end, width int, world [][]uint8) [][]int {
	neighbours := make([][]int, width)
	for i := range neighbours {
		neighbours[i] = make([]int, width)
	}

	if !(start == 0 && end == width) {
		if start == 0 {
			for x := 0; x < width; x++ {
				if world[width-1][x] == CellAlive {
					for i := -1; i <= 1; i++ {
						//for image wrap around
						xCoord := x + i
						if xCoord < 0 {
							xCoord = width - 1
						} else if xCoord >= width {
							xCoord = 0
						}
						neighbours[0][xCoord]++
					}
				}
			}
		} else {
			start--
		}

		if end == width {
			for x := 0; x < width; x++ {
				if world[0][x] == CellAlive {
					for i := -1; i <= 1; i++ {
						//for image wrap around
						xCoord := x + i
						if xCoord < 0 {
							xCoord = width - 1
						} else if xCoord >= width {
							xCoord = 0
						}
						neighbours[width-1][xCoord]++
					}
				}
			}
		} else {
			end++
		}
	}

	for y := start; y < end; y++ {
		for x := 0; x < width; x++ {
			// if a cell is CellAlive
			if world[y][x] == CellAlive {
				// add 1 to all neighbours
				// i and j are the offset
				for i := -1; i <= 1; i++ {
					for j := -1; j <= 1; j++ {

						// if you are not offset, do not add one. This is yourself
						if !(i == 0 && j == 0) {
							ny, nx := y+i, x+j
							if nx < 0 {
								nx = width - 1
							} else if nx == width {
								nx = 0
							} else {
								nx = nx % width
							}

							if ny < 0 {
								ny = width - 1
							} else if ny == width {
								ny = 0
							} else {
								ny = ny % width
							}

							neighbours[ny][nx]++
						}

					}
				}
			}
		}
	}

	return neighbours
}

func runAliveEvery2(done chan bool) {
	//make the ticker
	ticker := time.NewTicker(2 * time.Second)

	for {
		//see if we need to report the number of alive cells
		select {
		case <-done:
			return
		case _ = <-ticker.C:
			getCount.setTrue()
		}
	}
}

func paused(c distributorChannels, p Params, world [][]uint8, turns int) {
	c.events <- StateChange{turn.get(), Paused}
	for keyNew := range c.keyPresses {
		switch keyNew {
		case 's':
			makeOutputTurnWithTurnNum(p, c, turns, world)
		case 'p':
			c.events <- StateChange{turn.get(), Executing}
			pause.Done()
			return
		case 'q':
			end.setTrue()
			pause.Done()
			return
		}
	}
}

func runKeyPressController(c distributorChannels, p Params) {
	for key := range c.keyPresses {
		switch key {
		case 's':
			snapshot.setTrue()
		case 'q':
			end.setTrue()
			return
		case 'p':
			executingKeyPress.Add(1)
			pause.Add(1)
			pauseReached.Wait()
			paused(c, p, worldGlobal, turn.get())
			executingKeyPress.Done()
		}
	}
}

func executeTurns(p Params, c distributorChannels, world [][]uint8) [][]uint8 {
	pauseReached.Add(1)
	for turn.get() < p.Turns {
		// call the function to calculate new CellAlive cells from old CellAlive cells
		world = calculateNewAliveParallel(p, p.Threads, c, world)
		worldGlobal = world

		// increase the number of turns passed
		turn.inc()
		//runKeyPressController(c, p)

		pauseReached.Done()
		pause.Wait()
		pauseReached.Add(1)

		if end.get() {
			return world
		}
		if getCount.get() {
			getCount.setFalse()
			c.events <- AliveCellsCount{turn.get(), getNumAliveCells(p, world)}
		}
		if snapshot.get() {
			snapshot.setFalse()
			makeOutputTurnWithTurnNum(p, c, turn.get(), world)
		}

		c.events <- TurnComplete{turn.get()}
		worldOld = world

	}
	return world
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	// Create a 2D slice to store the world.
	world := make([][]uint8, p.ImageHeight)
	for i := range world {
		world[i] = make([]uint8, p.ImageWidth)
	}

	turn.reset()

	// add a get input to the command channel
	// give file name
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%vx%v", p.ImageWidth, p.ImageHeight)

	cells := make([]util.Cell, p.ImageWidth*p.ImageHeight)

	world = <-c.ioInput
	worldOld = world

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[x][y] == 255 {
				cells = append(cells, util.Cell{X: x, Y: y})
			}
		}
	}

	worldGlobal = world
	//worldGlobal.setup(world)
	//for _, n := range worldGlobal.giveWhole() {
	//	fmt.Println(n)
	//}
	c.events <- CellsFlipped{turn.get(), cells}
	c.events <- StateChange{turn.get(), Executing}

	// key press controller
	// create stop channel for quitting
	go runKeyPressController(c, p)

	// create a new ticket and a channel to stop it
	// run the ticker
	done := make(chan bool)
	go runAliveEvery2(done)

	// Make sure that the Io has finished any input before moving on to processing
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	// Execute all turns of the Game of Life.
	world = executeTurns(p, c, world)

	//output the slice to a pgm
	if end.get() {
		makeOutputTurnWithTurnNum(p, c, turn.get(), world)
	} else {
		makeOutput(p, c, world)
	}

	//Report the final state using FinalTurnCompleteEvent.
	aliveCells := getAliveCells(p, world)
	c.events <- FinalTurnComplete{turn.get(), aliveCells}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn.get(), Quitting}

	// stop the ticker
	done <- true

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	executingKeyPress.Wait()
	close(done)
	close(c.events)
}
