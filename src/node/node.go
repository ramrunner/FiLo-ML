package node

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

var (
	r *rand.Rand // package level RNG
)

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type StatCollector interface {
	Hit(string, int)
}

type Node struct {
	id          string          // node name
	paceMillis  int             // center of probability for delay
	deltaMillis int             // delta of delay
	iters       int             // how many iters should be completed
	ineighc     []chan struct{} // chans to neighbors
	oneighc     []chan struct{} // chans to neighbors
	neighp      []*Node         // neighbor list
	firstTime   bool            // marks if it's the first iter of the node
}

func NewNode(id string, pace, delta int) *Node {
	return &Node{
		id:          id,
		paceMillis:  pace,
		deltaMillis: delta,
		iters:       0,
		ineighc:     []chan struct{}(nil),
		oneighc:     []chan struct{}(nil),
		neighp:      []*Node(nil),
		firstTime:   true,
	}

}

func (n *Node) Delays() (int, int) {
	return n.paceMillis, n.deltaMillis
}

func (n *Node) Neighbors() []*Node {
	return n.neighp
}

func (n *Node) Degree() int {
	return len(n.neighp)
}

func (n *Node) Id() string {
	return n.id
}

func (n *Node) SetIters(its int) {
	n.iters = its
}

func (n *Node) SetFirstTime(b bool) {
	n.firstTime = b
}

func (n *Node) Connect(on *Node) error {
	//check if its ourselves
	if on.id == n.id {
		return fmt.Errorf("self connection attempt")
	}
	//check if we are already connected.
	for _, np := range n.neighp {
		//fmt.Printf("comparing %s with %s\n")
		if np.id == on.id {
			fmt.Printf("ERROR!\n")
			return fmt.Errorf("attempted reconnection on %s with %s", n.id, on.id)
		}
	}
	n.neighp = append(n.neighp, on)
	on.neighp = append(on.neighp, n)
	inc := make(chan struct{}, 0)
	onc := make(chan struct{}, 0)
	n.ineighc = append(n.ineighc, inc)
	n.oneighc = append(n.oneighc, onc)
	on.oneighc = append(on.oneighc, inc)
	on.ineighc = append(on.ineighc, onc)
	return nil
}

func (n *Node) Start(ctx context.Context, wg *sync.WaitGroup, statc StatCollector) {
	wg.Add(1)
	go n.Work(ctx, wg, statc)
}

func (n *Node) getRandInRange() int {
	return r.Intn(2*n.deltaMillis) + (n.paceMillis - n.deltaMillis)
}

func (n *Node) Work(ctx context.Context, wg *sync.WaitGroup, statc StatCollector) {
	defer wg.Done()
	iwg := &sync.WaitGroup{}
	inch := make(chan struct{})
	inagg := make(chan struct{})
	for _, ic := range n.ineighc {
		iwg.Add(1)
		go func(lic chan struct{}) {
			for x := range lic {
				inagg <- x
			}
			iwg.Done()
		}(ic)
	}
	iwg.Add(1)
	go func() {
		sum := 0
		for range inagg {
			sum++
			if sum >= len(n.oneighc) {
				inch <- struct{}{}
				sum = 0
			}
		}
		iwg.Done()
	}()
	go func() {
		iwg.Wait()
		close(inagg)
		close(inch)
	}()
	outch := make(chan struct{})
	go func() {
		defer owg.Done()
		for x := range outch {
			for _, oc := range n.oneighc {
				oc <- x
			}
		}
		for _, oc := range n.oneighc {
			close(oc)
		}
	}()
	/*for _, oc := range n.oneighc {
		//owg.Add(1)
		go func(loc chan struct{}) {
			for x := range outch {
				loc <- x
			}
			close(loc)
		}(oc)
	}*/
	if n.firstTime {
		n.iters--
		time.Sleep(time.Duration(n.getRandInRange()) * time.Millisecond)
		n.firstTime = false
		//log.Printf("WRITE w[%s] it:%d inc:%d onc:%d", n.id, n.iters, len(n.ineighc), len(n.oneighc))
		outch <- struct{}{}
	}

	for range inch {
		statc.Hit(n.id, n.iters)
		if n.iters <= 0 {
			return
		}
		//log.Printf("READ w[%s] it:%d inc:%d onc:%d", n.id, n.iters, len(n.ineighc), len(n.oneighc))
		time.Sleep(time.Duration(n.getRandInRange()) * time.Millisecond)
		outch <- struct{}{}
		n.iters--
		if n.iters <= 0 {
			log.Printf("%s DONE!", n.id)
			break
		}
	}
	close(outch)
}
