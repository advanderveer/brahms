package brahms

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/advanderveer/go-test"
)

func TestMiniNetCore(t *testing.T) {
	n1 := N("127.0.0.1", 1)
	n2 := N("127.0.0.1", 2)
	n3 := N("127.0.0.1", 3)

	rnd := rand.New(rand.NewSource(1))
	prm, _ := NewParams(0.45, 0.45, 0.1, 100, 10)

	//create a mini network with three cores
	tr := NewMemNetTransport()
	c1 := NewCore(rnd, n1, NewView(n2), prm, tr)
	tr.AddCore(c1)
	c2 := NewCore(rnd, n2, NewView(n3), prm, tr)
	tr.AddCore(c2)
	c3 := NewCore(rnd, n3, NewView(n1), prm, tr)
	tr.AddCore(c3)

	// after two iterations we should have a connected graph
	for i := 0; i < 10; i++ {
		c1.UpdateView(time.Millisecond)
		c2.UpdateView(time.Millisecond)
		c3.UpdateView(time.Millisecond)
	}

	// view and sampler should show a connected graph
	test.Equals(t, NewView(n2, n3), c1.view)
	test.Equals(t, NewView(n2, n3), c1.sampler.Sample())
	test.Equals(t, NewView(n1, n3), c2.view)
	test.Equals(t, NewView(n1, n3), c2.sampler.Sample())
	test.Equals(t, NewView(n1, n2), c3.view)
	test.Equals(t, NewView(n1, n2), c3.sampler.Sample())
}

func TestNetworkJoin(t *testing.T) {
	//@TODO test how a member would join an existing network by knowing just one
	//other node in the network. A single push without a pull would not be taken
	//into account currently. Is that a real problem in an actual network?
}

func TestLargeNetwork(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	n := uint16(100)
	q := 40

	td := 20
	d := 0.05
	nd := int(math.Round(float64(n) * d))

	m := 1.0
	l := int(math.Round(m * math.Pow(float64(n), 1.0/3)))
	p, _ := NewParams(
		0.45,
		0.45,
		0.1,
		l, l,
	)

	tr := NewMemNetTransport()
	cores := make([]*Core, 0, n)
	for i := uint16(1); i <= n; i++ {
		self := N("127.0.0.1", i)
		other := N("127.0.0.1", i+1)
		if i == n {
			other = N("127.0.0.1", 1)
		}

		c := NewCore(r, self, NewView(other), p, tr)
		tr.AddCore(c)
		cores = append(cores, c)
	}

	var wg sync.WaitGroup
	for i := 0; i < q; i++ {

		// if not short test: draw graphs
		if !testing.Short() && (5&i == 0 || i == td || i == td+1) {
			views := make(map[*Node]View, len(cores))
			dead := make(map[NID]struct{})
			for _, c := range cores {
				views[c.Self()] = c.view.Copy()

				if !c.alive {
					dead[c.Self().Hash()] = struct{}{}
				}
			}

			wg.Add(1)
			go func(i int, views map[*Node]View) {
				defer wg.Done()

				buf := bytes.NewBuffer(nil)
				draw(t, buf, views, dead)
				drawPNG(t, buf, fmt.Sprintf(filepath.Join("draws", "network_%d.png"), i))
				fmt.Println("drawing step '", i, "'...")

			}(i, views)
		}

		// move the cores ahead in time
		for _, c := range cores {
			if !c.alive {
				continue
			}

			c.UpdateView(100 * time.Microsecond)
			c.ValidateSample(100 * time.Microsecond)
		}

		// after some time turn off some cores
		if i == td {
			for i := 0; i < nd; i++ {
				idx := r.Intn(len(cores))
				cores[idx].alive = false
				cores[idx].view = View{}
			}
		}
	}

	var tot float64
	for _, c := range cores {
		tot += float64(len(c.view))
	}

	wg.Wait() //wait for drawings

	// @TODO assert that no-one connects to the in-active cores anymore
	// @TODO assert that the rest is still connected
	// test.Assert(t, tot/float64(len(cores)) > 3.1, "should be reasonably connected")
}
