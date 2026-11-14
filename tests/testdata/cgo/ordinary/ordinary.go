package ordinary

// static int number(void) { return 1; }
import "C"

func Number() int { return int(C.number()) }
