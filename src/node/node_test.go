package node

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func makeNodes(hm int) []*Node {
	ret := make([]*Node, hm)
	for i := 0; i < hm; i++ {
		ret[i] = NewNode(fmt.Sprintf("test-%d", i), 1000, 10)
		ret[i].SetIters(10)
	}
	return ret
}

// will make sure that the fired goroutines will stop when out of iters.
func TestStopping(t *testing.T) {
	nods := makeNodes(10)
	// each will do approx 10 iters of max 1s+10ms
	// therefore if we wait for 11 s all should have terminated
	wg := &sync.WaitGroup{}
	wg1 := &sync.WaitGroup{}
	ctx, cf := context.WithCancel(context.Background())
	wg1.Add(1)
	go func() {
		tot := 15
		defer wg1.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(1 * time.Second)
				tot--
				if tot > 0 {
					continue
				} else {
					t.Logf("exiting due to cf\n")
					cf()
					t.Fail()
				}
			}
		}
	}()
	t1 := time.Now()
	for _, n := range nods {
		n.Start(ctx, wg)
		t.Logf("starting!")
	}
	wg.Wait()
	cf()
	wg1.Wait()
	t.Logf("exited in :%v\n", time.Since(t1))
}
