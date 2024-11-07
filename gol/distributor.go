package gol

import (
	"fmt"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

var wg, pause, executingKeyPress sync.WaitGroup

// var mu sync.Mutex
var turn IntContainer
var worldGlobal, oldWorld WorldContainer

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

func (c *WorldContainer) setup(worldS [][]uint8) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.world = worldS
}

func (c *WorldContainer) read(x, y int) uint8 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.world[x][y]
}

func (c *WorldContainer) giveWhole() [][]uint8 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.world
}

func (c *WorldContainer) write(x, y int, val uint8) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.world[x][y] = val
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
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

type pixel struct {
	X     int
	Y     int
	Value uint8
}

type neighbourPixel struct {
	X int
	Y int
}

func getAliveCells(p Params) []util.Cell {
	// make the slice
	aliveCells := make([]util.Cell, 0)

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			// append every CellAlive cells
			if worldGlobal.read(x, y) == CellAlive {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}

	return aliveCells
}

func getNumAliveCells(p Params) int {
	num := 0
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			// append every CellAlive cells
			if worldGlobal.read(x, y) == CellAlive {
				num++
			}
		}
	}

	return num
}

func makeOutput(p Params, c distributorChannels) {

	// add a get output to the command channel
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, p.Turns)

	// add one pixel at a time to the output channel
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- worldGlobal.read(x, y)
		}
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
}

func makeOutputTurnWithTurnNum(p Params, c distributorChannels, turns int) {

	// add a get output to the command channel
	c.ioCommand <- ioOutput
	filename := fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, turns)
	c.ioFilename <- filename

	// add one pixel at a time to the output channel
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- worldGlobal.read(x, y)
		}
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- ImageOutputComplete{turns, filename}
}

func makeOutputOld(world [][]uint8, p Params, c distributorChannels, turns int) {

	// add a get output to the command channel
	c.ioCommand <- ioOutput
	filename := fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, turns)
	c.ioFilename <- filename

	// add one pixel at a time to the output channel
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[x][y]
		}
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- ImageOutputComplete{turns, filename}
}

func combineChannelData(data chan pixel, c distributorChannels) {
	length := len(data)

	for i := 0; i < length; i++ {

		item := <-data
		x := item.X
		y := item.Y

		if item.Value != worldGlobal.read(x, y) {
			worldGlobal.write(x, y, item.Value)
			c.events <- CellFlipped{turn.get(), util.Cell{X: x, Y: y}}
		}
	}
}

func combineChannelDataN(data chan neighbourPixel, p Params) [][]int {
	// Create a 2D slice to store the neighbour count
	neighbours := make([][]int, p.ImageHeight)
	for i := range neighbours {
		neighbours[i] = make([]int, p.ImageWidth)
	}

	length := len(data)
	for i := 0; i < length; i++ {

		item := <-data
		neighbours[item.X][item.Y] += 1
	}
	return neighbours
}

func startWorkers(workerNum, numRows int, p Params, c chan pixel, neighbours [][]int) {
	// if there is only one worker
	if workerNum == 1 {
		wg.Add(1)
		go updateWorldWorker(0, numRows, neighbours, c, p)
		return
	}

	// if there is more than one worker
	//first worker
	wg.Add(1)
	go updateWorldWorker(0, numRows, neighbours, c, p)

	// spread work between workers up to the last whole multiple
	num := 1
	finishRow := numRows
	for i := numRows; i < p.ImageHeight-numRows; i += numRows {
		wg.Add(1)
		go updateWorldWorker(i, i+numRows, neighbours, c, p)
		num++
		finishRow += numRows
	}
	// final worker does the remaining rows
	wg.Add(1)
	go updateWorldWorker(finishRow, p.ImageHeight, neighbours, c, p)

}

func startWorkersNeighbours(workerNum, numRows int, p Params, neighbourChan chan neighbourPixel) {
	// if there is only one worker
	if workerNum == 1 {
		// add one to wait for this worker
		wg.Add(1)
		go calculateNewAlive(p, 0, numRows, neighbourChan)
		return
	}

	// if there is more than one worker
	// add one to wait for this worker
	wg.Add(1)
	go calculateNewAlive(p, 0, numRows, neighbourChan)

	// spread work between workers up to the last whole multiple
	finishRow := numRows
	for i := numRows; i < p.ImageHeight-numRows; i += numRows {
		// add one to wait for this worker
		wg.Add(1)
		go calculateNewAlive(p, i, i+numRows, neighbourChan)
		finishRow += numRows
	}

	// final worker does the remaining rows
	// add one to wait for this worker
	wg.Add(1)
	go calculateNewAlive(p, finishRow, p.ImageHeight, neighbourChan)
}

func calculateNewAliveParallel(p Params, workerNum int, c distributorChannels) {
	// this says RoundUp(p.imageHeight / workNum)
	numRows := (p.ImageHeight + workerNum - 1) / workerNum

	// make channels for the world data and neighbour data
	// needs to be the size of the board
	dataChan := make(chan pixel, p.ImageWidth*p.ImageHeight)
	// needs to be the size of the board 8 times as we may send 8 neighbours for each pixel
	neighbourChan := make(chan neighbourPixel, 8*p.ImageWidth*p.ImageHeight)
	// close these channels after calculation
	defer close(dataChan)
	defer close(neighbourChan)

	// start workers for calculating neighbours
	startWorkersNeighbours(workerNum, numRows, p, neighbourChan)

	// wait for neighbours to be calculated, then recombine neighbours
	wg.Wait()
	neighbours := combineChannelDataN(neighbourChan, p)

	// start workers to make the world
	startWorkers(workerNum, numRows, p, dataChan, neighbours)

	//wait for neighbours to be calculated, then recombine world
	wg.Wait()
	combineChannelData(dataChan, c)
}

func calculateNewAlive(p Params, start, end int, n chan neighbourPixel) {
	// state that this worker is done once the functions completes
	defer wg.Done()

	//get neighbours
	// for all cells, calculate how many neighbours it has
	for y := start; y < end; y++ {
		for x := 0; x < p.ImageWidth; x++ {

			// if a cell is CellAlive
			if worldGlobal.read(x, y) == CellAlive {
				// add 1 to all neighbours
				// i and j are the offset
				for i := -1; i <= 1; i++ {
					for j := -1; j <= 1; j++ {

						//for image wrap around
						xCoord := x + i
						if xCoord < 0 {
							xCoord = p.ImageWidth - 1
						} else {
							xCoord = xCoord % p.ImageWidth
						}

						// for image wrap around
						yCoord := y + j
						if yCoord < 0 {
							yCoord = p.ImageHeight - 1
						} else {
							yCoord = yCoord % p.ImageHeight
						}

						// if you are not offset, do not add one as this is yourself
						if !(i == 0 && j == 0) {
							n <- neighbourPixel{xCoord, yCoord}
						}
					}
				}
			}
		}
	}
}

func updateWorldWorker(start, end int, neighbours [][]int, c chan pixel, p Params) {
	// state that you are done once the function finishes
	defer wg.Done()

	// for all cells in your region
	for y := start; y < end; y++ {
		for x := 0; x < p.ImageWidth; x++ {

			numNeighbours := neighbours[x][y]
			// you die with less than two or more than 3 neighbours (or stay dead)
			if numNeighbours < 2 || numNeighbours > 3 {
				c <- pixel{x, y, CellDead}
			} else if numNeighbours == 3 {
				// you become alive if you are dead and have exactly 3
				c <- pixel{x, y, CellAlive}
			}
		}
	}
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

func paused(c distributorChannels, p Params) {
	c.events <- StateChange{turn.get(), Paused}
	for keyNew := range c.keyPresses {
		switch keyNew {
		case 's':
			makeOutputOld(worldGlobal.giveWhole(), p, c, turn.get())
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
			paused(c, p)
			executingKeyPress.Done()
		}
	}
}

func executeTurns(p Params, c distributorChannels) {
	for turn.get() < p.Turns {

		// call the function to calculate new CellAlive cells from old CellAlive cells
		calculateNewAliveParallel(p, p.Threads, c)

		oldWorld.setup(worldGlobal.giveWhole())

		// increase the number of turns passed
		turn.inc()

		//runKeyPressController(c, p)

		pause.Wait()
		if end.get() {
			return
		}
		if getCount.get() {
			getCount.setFalse()
			c.events <- AliveCellsCount{turn.get(), getNumAliveCells(p)}
		}
		if snapshot.get() {
			snapshot.setFalse()
			makeOutputTurnWithTurnNum(p, c, turn.get())
		}

		c.events <- TurnComplete{turn.get()}
	}
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
	// read each uint8 data off of the input channel
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			val := <-c.ioInput
			if val == 255 {
				world[x][y] = 255
				cells = append(cells, util.Cell{X: x, Y: y})
			}
		}
	}

	oldWorld.setup(world)
	worldGlobal.setup(world)
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
	executeTurns(p, c)

	//output the slice to a pgm
	if end.get() {
		makeOutputTurnWithTurnNum(p, c, turn.get())
	} else {
		makeOutput(p, c)
	}

	//Report the final state using FinalTurnCompleteEvent.
	aliveCells := getAliveCells(p)
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
