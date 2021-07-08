package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	startTime := time.Now()
	runTime := 1000
	var wg sync.WaitGroup
	p, _ := NewPool(100, func(i interface{}) {
		test(i)
		wg.Done()
	})
	defer p.Close()

	for i := 0; i < runTime; i++ {
		wg.Add(1)
		p.Submit(i)
	}
	wg.Wait()
	fmt.Printf("running goroutines: %d\n", p.runNum)
	fmt.Printf("time: %dms", time.Since(startTime)/time.Millisecond)
}

func test(i interface{}) {
	n := i.(int)
	n++
	fmt.Printf("%d \n", n)
}
