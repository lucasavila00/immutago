package cgolib

// static int add(int a, int b) { return a + b; }
import "C"
import (
	_ "example.com/consumer/bad"
	_ "github.com/lucasavila00/immutago/pkg/im"
	_ "unsafe"
)

// Add calls C so the test exercises real compiler-generated wrappers.
func Add(a, b int) int { return int(C.add(C.int(a), C.int(b))) }
