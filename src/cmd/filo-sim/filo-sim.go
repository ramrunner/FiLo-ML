package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ramrunner/FiLo-ML/connect"
	"github.com/ramrunner/FiLo-ML/node"

	"github.com/goccy/go-graphviz"
	gv "github.com/goccy/go-graphviz"
	gvc "github.com/goccy/go-graphviz/cgraph"
)

var (
	gtype   string
	nodes   int
	network gnetwork
	graph   *gvc.Graph
	niters  intslice
)

type statsCol struct {
	thits int
	hits  int
	hitc  chan measurement
}
type measurement struct {
	key string
	val int
}

func NewStatsCol() *statsCol {
	return &statsCol{
		thits: 0,
		hits:  0,
		hitc:  make(chan measurement, 0),
	}
}

func (s *statsCol) Hit(a string, val int) {
	s.hitc <- measurement{a, val}
}

type kval struct {
	k string
	v int
}

func maxmink(a map[string]int) (string, int, string, int) {
	kvarr := []kval(nil)
	for k, v := range a {
		kvarr = append(kvarr, kval{k, v})
	}
	sort.Slice(kvarr, func(i, j int) bool {
		return kvarr[i].v < kvarr[j].v
	})
	if len(kvarr) > 0 {
		return kvarr[0].k, kvarr[0].v, kvarr[len(kvarr)-1].k, kvarr[len(kvarr)-1].v
	}
	return "", 0, "", 0
}

func (s *statsCol) Print(freq int) {
	vals := make(map[string]int)
	tic := time.NewTicker(time.Duration(freq) * time.Second)
	iter := 0
	for {
		select {
		case <-tic.C:
			//log.Printf("total:%d rate:%v", s.thits, s.hits/freq)
			//tn := time.Now()
			_, v1, _, v2 := maxmink(vals)
			fmt.Printf("%v\t\t%s\t\t%d\t\t%d\n", iter, v1, v2, s.hits/freq)
			iter++
			s.hits = 0
		case m := <-s.hitc:
			vals[m.key] = m.val
			s.thits = s.thits + 1
			s.hits = s.hits + 1

		}
	}
}

type intslice []int

func (i *intslice) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *intslice) Set(val string) error {
	svals := strings.Split(val, ",")
	for p := range svals {
		nv, e := strconv.Atoi(svals[p])
		if e != nil {
			return e
		}
		*i = append(*i, nv)
	}
	return nil
}

func init() {
	flag.StringVar(&gtype, "gtype", "random", "type of string to support: random, parallel, linear, fattree")
	flag.IntVar(&nodes, "nodes", 20, "number of nodes")
	flag.Var(&niters, "niters", "comma separated list of iters to run experiment on")
}

func usage() {
	fmt.Printf("commands can be create, simulate\n")
}

type gnode struct {
	*node.Node
	gvn *gvc.Node
}

type gnetwork []*gnode

func (g gnetwork) getNodes() []*node.Node {
	ret := make([]*node.Node, len(g))
	for i := range g {
		ret[i] = g[i].Node
	}
	return ret
}

func (g gnetwork) findId(a string) *gnode {
	for i := range g {
		if g[i].Node.Id() == a {
			return g[i]
		}
	}
	return nil
}

func (g gnetwork) gConnect() {
	for _, i := range g {
		for _, j := range i.Node.Neighbors() {
			realj := g.findId(j.Id())
			graph.CreateEdge("e", i.gvn, realj.gvn)
		}

	}
}

func main() {
	var e error
	flag.Parse()
	if len(flag.Args()) == 0 || flag.Args()[0] != "create" {
		usage()
		return
	}
	g := gv.New()
	gv.UnDirected(g)
	mst := NewStatsCol()
	go mst.Print(10)
	graph, e = g.Graph()
	if e != nil {
		log.Fatal(e)
	}
	defer func() {
		if e := graph.Close(); e != nil {
			log.Fatal(e)
		}
		g.Close()
	}()
	network = make([]*gnode, nodes)
	for i := 0; i < nodes; i++ {
		name := fmt.Sprintf("n%d", i)
		var innode *node.Node
		if i%2 == 0 {
			innode = node.NewNode(name, rand.Intn(800)+200, 200)
		} else {
			innode = node.NewNode(name, rand.Intn(80)+20, 20)
		}
		delay, _ := innode.Delays()
		gnn, e := graph.CreateNode(fmt.Sprintf("%s (%d)", name, delay))
		if e != nil {
			log.Fatal(e)
		}
		network[i] = &gnode{innode, gnn}
	}
	switch gtype {
	case "random":
		e = connect.Random(network.getNodes(), 3)
	case "parallel":
		e = connect.Parallel(network[0].Node, network[nodes-1].Node, network[1:nodes-1].getNodes())
	case "linear":
		e = connect.Linear(network.getNodes())
	case "fattree":
		e = connect.FatTree(network.getNodes(), 3)
	}
	if e != nil {
		log.Fatalf("error in connect:%s", e)
	}
	network.gConnect()
	if err := g.RenderFilename(graph, graphviz.PNG, "output.png"); err != nil {
		log.Fatal(err)
	}
	log.Printf("====starting simulation=====\n")
	log.Printf("Will run experiments for times :%v\n", niters)
	for _, it := range niters {
		log.Printf("niters:%d\n", it)
		wg := &sync.WaitGroup{}
		wg1 := &sync.WaitGroup{}
		ctx, cf := context.WithCancel(context.Background())
		wg1.Add(1)
		go func() { // fire up safety goroutine in case they get stuck
			// since max is 1sec per node. n*nodes is max time
			tot := nodes * it
			log.Printf("setting deadlock check for %d seconds", tot)
			defer wg1.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(1 * time.Second)
					tot--
					if tot%10 == 0 {
						log.Printf("deadlock detector reaping in %d s\n", tot)
					}
					if tot > 0 {
						continue
					} else {
						cf()
						log.Fatal("goroutines got stuck and killed")
					}
				}
			}
		}()
		t1 := time.Now()
		for i, n := range network {
			n.SetIters(it)
			n.SetFirstTime(true)
			if i == len(network)-1 { // the last node will break symmetry
				//n.SetFirstTime(false)
			}
			n.Start(ctx, wg, mst)
		}
		log.Printf("started at:%v\n", t1)
		wg.Wait()
		cf()
		log.Printf("Done in:%v\n", time.Since(t1))

	}
	return
}
