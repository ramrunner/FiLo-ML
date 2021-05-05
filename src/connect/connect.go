package connect

import (
	"math/rand"
	"sort"
	"time"

	"github.com/ramrunner/FiLo-ML/node"
)

var (
	r *rand.Rand // package level RNG
)

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func Linear(n []*node.Node) error {
	var e error
	for i := range n[:len(n)-1] {
		//fmt.Printf("COnnecting %s with %s\n", n[i].Id(), n[i+1].Id())
		e = n[i].Connect(n[i+1])
		if e != nil {
			return e
		}
	}
	return nil
}

func Parallel(nstart, nend *node.Node, npar []*node.Node) error {
	var e error
	for i := range npar {
		if nstart != nil {
			e = nstart.Connect(npar[i])
			if e != nil {
				return nil
			}
		}
		if nend != nil {
			e = nend.Connect(npar[i])
			if e != nil {
				return nil
			}
		}
	}
	return nil
}

func Random(n []*node.Node, maxdeg int) error {
	var e error
	r.Shuffle(len(n), func(i, j int) {
		n[i], n[j] = n[j], n[i]
	})
	done := []*node.Node(nil)
	for i := range n {
		if i < len(n)-1 {
			e = n[i].Connect(n[i+1])
			if e != nil {
				continue
			}
			done = append(done, n[i])
			possib := done[r.Intn(len(done))]
			if possib.Degree() < maxdeg {
				e = possib.Connect(n[i])
				if e != nil {
					continue
				}
			}
		}
	}
	return nil
}
func bfunc(done, still []*node.Node, maxdeg int) ([]*node.Node, error) {
	var e error
	if len(still) == 0 {
		return done, nil
	}
	if len(done) > len(still) {
		e = done[len(done)-1].Connect(still[0])
		e = Random(still, maxdeg)
		done[len(done)-2].Connect(still[1])
		done[len(done)-3].Connect(still[2])
		return done, e
	}
	//randomize half and connect with larger
	e = Random(still[:len(still)/2], maxdeg)
	if e != nil {
		return done, e
	}
	if len(done) > 0 {
		e = done[len(done)-1].Connect(still[0]) //connect smaller
		e = done[len(done)-2].Connect(still[1]) //connect smaller
		e = done[len(done)-3].Connect(still[2]) //connect smaller
		if e != nil {
			return done, e
		}
	}
	//larger should be connect by the next run.
	//recurse
	done, e = bfunc(append(done, still[:len(still)/2]...), still[len(still)/2:], maxdeg)
	return done, e

}

func FatTree(n []*node.Node, maxdeg int) error {
	var e error
	sort.Slice(n, func(i, j int) bool {
		n1, _ := n[i].Delays()
		n2, _ := n[j].Delays()
		return n1 < n2
	})
	done := []*node.Node(nil)
	done, e = bfunc(done, n, maxdeg)
	return e
}
