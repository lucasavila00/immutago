package main

// makePair returns an owned value with a scalar result.
func makePair() (Point, bool) { return Point{X: 1}, true }

// forwardPair forwards results without creating local bindings.
func forwardPair() (Point, bool) { return makePair() }

// takePair reads concrete value arguments.
func takePair(point Point, found bool) { println(point.X, found) }

// forwardArguments passes direct named results to concrete parameters.
func forwardArguments() { takePair(makePair()) }
