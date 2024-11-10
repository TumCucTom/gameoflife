package gol

import (
	"fmt"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

var wg, pause, executingKeyPress, calcN, combineN, calcA sync.WaitGroup

// var mu sync.Mutex
var turn IntContainer

var worldGlobal []pixel

//var worldGlobal, neighboursGlobal WorldContainer

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
	world []pixel
}

//func (c *WorldContainer) setup(worldS [][]uint8) {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//
//	c.world = worldS
//}
//
//func (c *WorldContainer) inc(x, y int) {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//
//	c.world[x][y]++
//}
//
//func (c *WorldContainer) read(x, y int) uint8 {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//
//	return c.world[x][y]
//}
//
//func (c *WorldContainer) giveWhole() [][]uint8 {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//
//	return c.world
//}
//
//func (c *WorldContainer) write(x, y int, val uint8) {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//
//	c.world[x][y] = val
//}

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

type suint8 struct {
	x   int
	val []uint8
}

type neighbourPixel struct {
	X    int
	Y    int
	quit bool
}

func getAliveCells(p Params, world []pixel) []util.Cell {
	// make the slice
	aliveCells := make([]util.Cell, 0)

	for _, item := range world {
		aliveCells = append(aliveCells, util.Cell{X: item.X, Y: item.Y})
	}

	return aliveCells
}

func getNumAliveCells(p Params, world []pixel) int {
	return len(world)
}

func makeOutput(p Params, c distributorChannels, world []pixel) {

	// add a get output to the command channel
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, p.Turns)

	size := p.ImageWidth
	count := 0
	for i := range world {
		current := world[i]
		for {
			if current.Y == count%size && current.X == count/size {
				c.ioOutput <- 255
				break
			} else {
				c.ioOutput <- 0
			}
			count++
		}
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
}

func makeOutputTurnWithTurnNum(p Params, c distributorChannels, turns int, world []pixel) {

	// add a get output to the command channel
	c.ioCommand <- ioOutput
	filename := fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, turns)
	c.ioFilename <- filename

	size := p.ImageWidth
	count := 0
	for i := range world {
		current := world[i]
		for {
			if current.Y == count%size && current.X == count/size {
				c.ioOutput <- 255
				break
			} else {
				c.ioOutput <- 0
			}
			count++
		}
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- ImageOutputComplete{turns, filename}
}

//func combineChannelDataNum(data chan pixelVal, c distributorChannels, workerNum int) {
//	for {
//		select {
//		case <-data:
//			item := <-data
//			if item.Value == 1 {
//				count.inc()
//
//			} else {
//				x := item.X
//				y := item.Y
//				if item.Value != worldGlobal.read(x, y) {
//					worldGlobal.write(x, y, item.Value)
//					c.events <- CellFlipped{turn.get(), util.Cell{X: x, Y: y}}
//				}
//			}
//		default:
//			if count.get() >= workerNum*2 {
//				return
//			}
//		}
//	}
//}

//func combineChannelData(data chan pixelVal) {
//	length := len(data)
//	for i := 0; i < length; i++ {
//		item := <-data
//		worldGlobal.write(item.X, item.Y, item.Value)
//	}
//}

//func combineChannelDataN(data chan neighbourPixel) {
//	length := len(data)
//	for i := 0; i < length; i++ {
//		item := <-data
//		neighboursGlobal.inc(item.X, item.Y)
//	}
//}

//func combineChannelDataNNum(data chan neighbourPixel, numWorkers int) {
//	for {
//		select {
//		case <-data:
//			item := <-data
//			if item.quit {
//				count.inc()
//				fmt.Print(count.get(), " ")
//			} else {
//				neighboursGlobal.inc(item.X, item.Y)
//			}
//		default:
//			if count.get() >= numWorkers {
//				wg.Done()
//				return
//			}
//		}
//	}
//}

//func startWaits(workers int) {
//	calcN.Add(workers)
//	calcA.Add(workers)
//	combineN.Add(1)
//}

//func startWorkers(workerNum, numRows int, p Params, c chan pixelVal, n chan neighbourPixel, chans distributorChannels) {
//	startWaits(workerNum)
//	i := 0
//	for i < workerNum-1 {
//		go updateWorldWorker(i*numRows, numRows*(i+1), workerNum, c, p, n, chans)
//		i++
//	}
//
//	// final worker does the remaining rows
//	go updateWorldWorker(i*numRows, p.ImageHeight, workerNum, c, p, n, chans)
//
//}

//func startWorkersNeighbours(workerNum, numRows int, p Params, neighbourChan chan neighbourPixel) {
//	// if there is only one worker
//	if workerNum == 1 {
//		// add one to wait for this worker
//		wg.Add(1)
//		go calculateNewAlive(p, 0, numRows, neighbourChan)
//		return
//	}
//
//	// if there is more than one worker
//	// add one to wait for this worker
//	wg.Add(1)
//	go calculateNewAlive(p, 0, numRows, neighbourChan)
//
//	// spread work between workers up to the last whole multiple
//	finishRow := numRows
//	for i := numRows; i < p.ImageHeight-numRows; i += numRows {
//		// add one to wait for this worker
//		wg.Add(1)
//		go calculateNewAlive(p, i, i+numRows, neighbourChan)
//		finishRow += numRows
//	}
//	// final worker does the remaining rows
//	// add one to wait for this worker
//	wg.Add(1)
//	go calculateNewAlive(p, finishRow, p.ImageHeight, neighbourChan)
//}

//func startWorkersCombineN(neighbours WorldContainer, workerNum, numRows int, p Params, neighbourChan chan neighbourPixel) {
//
//	// if there is only one worker
//	if workerNum == 1 {
//		// add one to wait for this worker
//		wg.Add(1)
//		go combineChannelDataNNum(neighbours, neighbourChan)
//		return
//	}
//
//	// if there is more than one worker
//	// add one to wait for this worker
//	wg.Add(1)
//	go combineChannelDataNNum(neighbours, neighbourChan)
//
//	// spread work between workers up to the last whole multiple
//	finishRow := numRows
//	for i := numRows; i < p.ImageHeight-numRows; i += numRows {
//		// add one to wait for this worker
//		wg.Add(1)
//		go combineChannelDataNNum(neighbours, neighbourChan)
//		finishRow += numRows
//	}
//
//	// final worker does the remaining rows
//	// add one to wait for this worker
//	wg.Add(1)
//	go combineChannelDataNNum(neighbours, neighbourChan)
//}

//func startWorkersCombine(workerNum, numRows int, p Params, data chan pixelVal, c distributorChannels) {
//
//	// if there is only one worker
//	if workerNum == 1 {
//		// add one to wait for this worker
//		wg.Add(1)
//		go combineChannelData(data, c, 16-workerNum)
//		return
//	}
//
//	// if there is more than one worker
//	// add one to wait for this worker
//	wg.Add(1)
//	go combineChannelData(data, c, 16-workerNum)
//
//	// spread work between workers up to the last whole multiple
//	finishRow := numRows
//	for i := numRows; i < p.ImageHeight-numRows; i += numRows {
//		// add one to wait for this worker
//		wg.Add(1)
//		go combineChannelData(data, c, 16-workerNum)
//		finishRow += numRows
//	}
//
//	// final worker does the remaining rows
//	// add one to wait for this worker
//	wg.Add(1)
//	go combineChannelData(data, c, 16-workerNum)
//}

func calculateNewAliveParallel(p Params, workerNum int, c distributorChannels, world []pixel) []pixel {
	//numRows := p.ImageHeight / workerNum

	// make channels for the world data and neighbour data
	// needs to be the size of the board
	//dataChan := make(chan pixelVal, p.ImageWidth*p.ImageHeight)
	//// needs to be the size of the board 8 times as we may send 8 neighbours for each pixelVal
	//neighbourChan := make(chan neighbourPixel, 8*p.ImageWidth*p.ImageHeight)
	//// close these channels after calculation
	//defer close(dataChan)
	//defer close(neighbourChan)
	//
	//neighbours := make([][]uint8, p.ImageHeight)
	//for i := range neighbours {
	//	neighbours[i] = make([]uint8, p.ImageWidth)
	//}
	//
	//neighboursGlobal.setup(neighbours)

	//// start workers for calculating neighbours
	//startWorkersNeighbours(14, numRows, p, neighbourChan)
	//var neighbours WorldContainer
	//neighbour := make([][]uint8, p.ImageHeight)
	//for i := range neighbour {
	//	neighbour[i] = make([]uint8, p.ImageWidth)
	//}
	//neighbours.setup(neighbour)
	////startWorkersCombineN(neighbours, amount, numRows, p, neighbourChan)
	//wg.Add(2)
	//go combineChannelDataNNum(neighbours, neighbourChan)
	//go combineChannelDataNNum(neighbours, neighbourChan)
	//
	//// wait for neighbours to be calculated
	//wg.Wait()
	//neighbour = neighbours.giveWhole()

	var newWorld []pixel

	splitSegments := make([]chan []pixel, workerNum)
	for i := range splitSegments {
		splitSegments[i] = make(chan []pixel, p.ImageWidth*p.ImageWidth)
	}
	// start workers to make the world
	//startWorkers(workerNum, numRows, p, dataChan, neighbourChan, c)
	setupWorkers(p.ImageHeight, workerNum, splitSegments, world)

	cells := make([]util.Cell, p.ImageWidth*p.ImageHeight)

	wg.Wait()
	for i := 0; i < workerNum; i++ {
		newWorld = append(newWorld, <-splitSegments[i]...)
	}
	c.events <- CellsFlipped{turn.get() + 1, cells}
	//for i := 0; i < workerNum; i++ {
	//	//temp := <-splitSegments[i]
	//	//for _, item := range temp {
	//	//	fmt.Println(item)
	//	//}
	//	newWorld = append(newWorld, <-splitSegments[i]...)
	//}

	return newWorld
}
func setupWorkers(size, workerNum int, splitSegments []chan []pixel, world []pixel) {
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
func runWorker(size, start, end int, splitSegment chan []pixel, world []pixel) {
	defer wg.Done()
	calculateNextWorld(start, end, size, world, splitSegment)
}

func calculateNextWorld(start, end, width int, world []pixel, c chan []pixel) {
	//newWorld := make([][]uint8, end-start)
	//for i := 0; i < end-start; i++ {
	//	newWorld[i] = world[i]
	//}

	newWorldSegment := worldGlobal[start:end]
	neighboursWorld := calculateNeighbours(start, end, width, world)
	//for _, item := range neighboursWorld {
	//	fmt.Println(item)
	//}

	for _, item := range world[start:end] {
		neighbours := neighboursWorld[item.Y][item.X]
		if neighbours < 2 || neighbours > 3 {
			newWorldSegment
		} else if neighbours == 3 && dead {
			c <- pixelVal{y, x, 255}
		}
	}
}

//func calculateNeighbours(start, end int, world []pixel) ([]pixel, []pixel) {
//	var aliveNeighbours = world[start:end]
//	var deadNeighbours []pixel
//
//	for k, item := range world[start:end] {
//		x := item.X
//		y := item.Y
//		i := k + start
//
//		//reverse
//		count := 1
//		if world[i-1].X == x-1 && world[i-1].Y == y {
//			aliveNeighbours[i].Value++
//		}
//		else{
//			deadNeighbours = deadNeighbours
//		}
//		count = 2
//		for {
//			y1 := world[i-count].Y
//			if y1 != y {
//				break
//			}
//			count++
//		}
//		for {
//			y1 := world[i-count].Y
//			if y1 == y-1 {
//				x1 := world[i-count].X
//				if x1 == x || x1 == x+1 || x1 == x-1 {
//					aliveNeighbours[i].Value++
//				}
//			} else {
//				break
//			}
//			count++
//		}
//
//		//forwards
//		count = 1
//		if world[i+1].X == x-1 && world[i+1].Y == y {
//			aliveNeighbours[i].Value++
//		}
//		count = 2
//		for {
//			y1 := world[i+count].Y
//			if y1 != y {
//				break
//			}
//			count++
//		}
//		for {
//			y1 := world[i+count].Y
//			if y1 == y+1 {
//				x1 := world[i+count].X
//				if x1 == x || x1 == x+1 || x1 == x-1 {
//					aliveNeighbours[i].Value++
//				}
//			} else {
//				break
//			}
//			count++
//		}
//	}
//	return aliveNeighbours, deadNeighbours
//}

func calculateNeighbours(start, end, width int, world []pixel) [][]int {
	neighbours := make([][]int, width)
	for i := range neighbours {
		neighbours[i] = make([]int, width)
	}

	if !(start == 0 && end == width) {
		if start == 0 {
			for i := 0; i >= len(world)-1; i++ {
				if world[i].Y == 0 {
					x := world[i].X
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
				} else {
					break
				}
			}
		} else {
			start--
		}

		if end == width {
			for i := len(world) - 1; i >= 0; i-- {
				if world[i].Y == width-1 {
					x := world[i].X
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
				} else {
					break
				}
			}
		} else {
			end++
		}
	}

	for _, item := range world[start:end] {
		x := item.X
		y := item.Y
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

	return neighbours
}

//func calculateNewAlive(p Params, start, end int, n chan neighbourPixel) {
//	// state that this worker is done once the functions completes
//
//	//get neighbours
//	// for all cells, calculate how many neighbours it has
//	for y := start; y < end; y++ {
//		for x := 0; x < p.ImageWidth; x++ {
//
//			// if a cell is CellAlive
//			if worldGlobal.read(x, y) == CellAlive {
//				// add 1 to all neighbours
//				// i and j are the offset
//				for i := -1; i <= 1; i++ {
//					for j := -1; j <= 1; j++ {
//
//						//for image wrap around
//						xCoord := x + i
//						if xCoord < 0 {
//							xCoord = p.ImageWidth - 1
//						} else if xCoord >= p.ImageWidth {
//							xCoord = 0
//						}
//
//						// for image wrap around
//						yCoord := y + j
//						if yCoord < 0 {
//							yCoord = p.ImageHeight - 1
//						} else if yCoord >= p.ImageWidth {
//							yCoord = 0
//						}
//
//						// if you are not offset, do not add one. This is yourself
//						if !(i == 0 && j == 0) {
//							n <- neighbourPixel{xCoord, yCoord, false}
//						}
//					}
//				}
//			}
//		}
//	}
//	//n <- neighbourPixel{-1, -1, true}
//}

//func updateWorldWorker(start, end, workerNum int, c chan pixelVal, p Params, n chan neighbourPixel, chans distributorChannels) {
//	calculateNewAlive(p, start, end, n)
//	calcN.Done()
//	calcN.Wait()
//
//	//combineChannelDataNNum(n, workerNum)
//	//combineN.Done()
//	combineN.Wait()
//	//for _, item := range neighbours {
//	//	fmt.Println(item)
//	//}
//
//	// for all cells in your region
//	for y := start; y < end; y++ {
//		for x := 0; x < p.ImageWidth; x++ {
//
//			numNeighbours := neighboursGlobal.read(x, y)
//			// you die with less than two or more than 3 neighbours (or stay dead)
//			if numNeighbours < 2 || numNeighbours > 3 {
//				c <- pixelVal{x, y, CellDead}
//			} else if numNeighbours == 3 {
//				// you become alive if you are dead and have exactly 3
//				c <- pixelVal{x, y, CellAlive}
//			}
//			// stay the same
//		}
//	}
//	//c <- pixelVal{-1, -1, 1}
//	calcA.Done()
//}

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

func paused(c distributorChannels, p Params, world []pixel) {
	c.events <- StateChange{turn.get(), Paused}
	for keyNew := range c.keyPresses {
		switch keyNew {
		case 's':
			makeOutputTurnWithTurnNum(p, c, turn.get(), world)
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
			paused(c, p, worldGlobal)
			executingKeyPress.Done()
		}
	}
}

func executeTurns(p Params, c distributorChannels, world []pixel) []pixel {
	for turn.get() < p.Turns {
		// call the function to calculate new CellAlive cells from old CellAlive cells
		world = calculateNewAliveParallel(p, p.Threads, c, world)
		worldGlobal = world

		// increase the number of turns passed
		turn.inc()
		//runKeyPressController(c, p)

		pause.Wait()
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
	}
	return world
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	// Create a 2D slice to store the world.
	var world []pixel

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
				world = append(world, pixel{x, y, 0})
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
