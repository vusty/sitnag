package figure

import (
	"math"
	"slices"
	"testing"
	"time"
)

// The card gives the mannequin this much room around the spot it stands on,
// in units: see figureX, figureFloor and figureScale in the window.
const (
	roomSide = 118
	roomUp   = 250
)

func TestEveryMoveStaysInItsRoom(t *testing.T) {
	for n := range Count {
		for at := time.Duration(0); at < 10*time.Second; at += 40 * time.Millisecond {
			f := At(n, at)
			if len(f.Parts) != 6 {
				t.Fatalf("move %d: %d parts, want trunk, head, two arms and two legs", n, len(f.Parts))
			}
			low := math.Inf(1)
			for _, p := range f.Parts {
				for _, s := range p.Shapes {
					for _, e := range [][3]float64{{s.X1, s.Y1, s.R1}, {s.X2, s.Y2, s.R2}} {
						x, y, r := e[0], e[1], e[2]
						if math.IsNaN(x) || math.IsNaN(y) {
							t.Fatalf("move %d at %v: not a number", n, at)
						}
						if math.Abs(x)+r > roomSide || y+r > roomUp {
							t.Fatalf("move %d at %v: reaches (%.0f, %.0f), outside its room", n, at, x, y)
						}
						low = min(low, y-r)
					}
				}
			}
			// Seen from a little above, the floor under a foot is within a
			// few units of zero on screen.
			if math.Abs(low) > 6 {
				t.Fatalf("move %d at %v: lowest point at %.1f, want it on the floor", n, at, low)
			}
		}
	}
}

func TestMovesLoopWithoutAJump(t *testing.T) {
	for n, m := range moves {
		// Parts come ordered by depth, and two at the same depth may swap, so
		// each shape is looked up by where it is.
		var end []Shape
		for _, p := range m.pose(1 - 1e-9).frame().Parts {
			end = append(end, p.Shapes...)
		}
		for _, p := range m.pose(0).frame().Parts {
			for _, s := range p.Shapes {
				if !slices.ContainsFunc(end, func(o Shape) bool {
					return math.Hypot(s.X1-o.X1, s.Y1-o.Y1) < 0.01 && math.Hypot(s.X2-o.X2, s.Y2-o.Y2) < 0.01
				}) {
					t.Fatalf("move %d: the end of a repetition does not meet its start", n)
				}
			}
		}
	}
}

func TestOutlineIsWoundOneWay(t *testing.T) {
	shapes := []Shape{
		{0, 0, 5, 0, 0, 5},     // a circle
		{0, 0, 8, 30, 10, 5},   // tapering
		{0, 0, 5, -20, 40, 5},  // a plain bar
		{10, 10, 3, 12, 10, 9}, // one end inside the other
	}
	for _, s := range shapes {
		pts := s.Outline(1)
		var area float64
		for i := 0; i < len(pts); i += 2 {
			j := (i + 2) % len(pts)
			area += pts[i]*pts[j+1] - pts[j]*pts[i+1]
		}
		if area <= 0 {
			t.Fatalf("%+v: outline wound the other way (area %.1f)", s, area/2)
		}
	}
}
