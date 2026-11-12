// Package figure poses a small mannequin and hands back the shapes to draw.
// The warm-up shows it doing each move, so nobody has to remember what a
// warm-up is made of. It knows nothing about Windows: the caller scales the
// shapes and fills them.
//
// The mannequin is a skeleton in three dimensions, posed by joint angles and
// looked at from a little above. A move is a function of time that returns a
// pose, so the motion is smooth at any frame rate and a new move is a dozen
// lines, not a set of drawings.
package figure

import (
	"math"
	"sort"
	"time"
)

// Height is how tall the mannequin stands, in the units of Shape.
const Height = 188

// Shape is a round-ended bar that may taper: a circle of radius R1 at (X1,
// Y1), one of radius R2 at (X2, Y2), and everything between them. The origin
// is on the floor between the feet, Y points up.
type Shape struct {
	X1, Y1, R1 float64
	X2, Y2, R2 float64
}

// Part is one piece of the body: an arm, a leg, the trunk, the head. Its
// shapes overlap and are meant to be filled as one. Tone runs from 0 for the
// side away from the viewer to 1 for the near side.
type Part struct {
	Shapes []Shape
	Tone   float64
}

// Frame is the mannequin at one moment, parts ordered back to front.
type Frame struct {
	Parts []Part
	// ShadowW and ShadowH are the size of the shadow on the floor, centred on
	// the origin.
	ShadowW, ShadowH float64
}

// At returns the mannequin doing move number n, t into the move. n wraps, so
// any count of moves can be cycled through.
func At(n int, t time.Duration) Frame {
	m := moves[((n%Count)+Count)%Count]
	phase := math.Mod(t.Seconds(), m.period) / m.period
	return m.pose(phase).frame()
}

// Body proportions, in units.
const (
	thighLen    = 46
	shinLen     = 44
	footLen     = 16
	ankleUp     = 5 // the ankle above the sole
	hipHalf     = 9
	trunkLen    = 58
	shoulderOut = 19
	shoulderDip = 5 // shoulders below the base of the neck
	upperArmLen = 31
	forearmLen  = 30
	headUp      = 20 // base of the neck to the centre of the head
	headR       = 12.5
)

// camPitch is how far above the horizon the viewer stands, in degrees. A
// little height gives the floor some depth and keeps a side view from going
// flat.
const camPitch = 9

type arm struct {
	side  float64 // raised sideways, away from the body
	front float64 // raised forward; past 180 it comes down behind
	open  float64 // swung outward around the vertical, from pointing forward
	twist float64 // turned around its own length; turns where the elbow bends
	elbow float64 // bend; with no twist the forearm comes forward
}

type leg struct {
	side  float64 // lifted sideways
	front float64 // thigh lifted forward
	knee  float64 // bend, the shin going back
	toe   float64 // the foot pointing down: on tiptoe
}

// pose is every joint angle, in degrees, plus where to look from. Left and
// right are the mannequin's own.
type pose struct {
	yaw float64 // the body turned to its left, away from facing the viewer

	bend, lean, twist float64 // trunk: forward, to the left, turned to the left
	nod, tilt         float64 // head, on top of the trunk: forward, to the left

	shrugL, shrugR float64 // shoulders lifted, in units
	rollL, rollR   float64 // shoulders brought forward, in units

	armL, armR arm
	legL, legR leg
}

type vec struct{ x, y, z float64 }

func (a vec) add(b vec) vec { return vec{a.x + b.x, a.y + b.y, a.z + b.z} }

type mat [3][3]float64

func (m mat) mul(n mat) mat {
	var out mat
	for i := range 3 {
		for j := range 3 {
			out[i][j] = m[i][0]*n[0][j] + m[i][1]*n[1][j] + m[i][2]*n[2][j]
		}
	}
	return out
}

func (m mat) apply(v vec) vec {
	return vec{
		m[0][0]*v.x + m[0][1]*v.y + m[0][2]*v.z,
		m[1][0]*v.x + m[1][1]*v.y + m[1][2]*v.z,
		m[2][0]*v.x + m[2][1]*v.y + m[2][2]*v.z,
	}
}

func sincos(deg float64) (float64, float64) { return math.Sincos(deg * math.Pi / 180) }

// The axes: x to the mannequin's left, y up, z the way it faces. rotX tips
// what points up towards the front, rotY turns what faces front to the left,
// rotZ tips what points up to the right.
func rotX(deg float64) mat { s, c := sincos(deg); return mat{{1, 0, 0}, {0, c, -s}, {0, s, c}} }
func rotY(deg float64) mat { s, c := sincos(deg); return mat{{c, 0, s}, {0, 1, 0}, {-s, 0, c}} }
func rotZ(deg float64) mat { s, c := sincos(deg); return mat{{c, -s, 0}, {s, c, 0}, {0, 0, 1}} }

// bone is one drawn segment between two joints.
type bone struct {
	a, b   vec
	ra, rb float64
}

// limb is the bones of one body part, in the mannequin's own space. side is
// where it joins the body, measured from the midline: +1 for the left arm and
// leg, -1 for the right, nothing for the trunk and the head.
type limb struct {
	bones []bone
	side  vec
}

// frame works out where every joint is, stands the mannequin on the floor,
// turns it to the viewer and flattens it.
func (p pose) frame() Frame {
	var pelvis vec
	trunk := rotX(p.bend).mul(rotZ(-p.lean)).mul(rotY(p.twist))
	at := func(v vec) vec { return pelvis.add(trunk.apply(v)) }
	neck := at(vec{0, trunkLen, 0})
	head := neck.add(trunk.mul(rotZ(-p.tilt)).mul(rotX(p.nod)).apply(vec{0, headUp, 0}))

	limbs := []limb{
		// Two bars side by side, so the trunk is wider than it is deep.
		{bones: []bone{
			{at(vec{5, 3, 0}), at(vec{8, trunkLen - 13, 0}), 9, 10.5},
			{at(vec{-5, 3, 0}), at(vec{-8, trunkLen - 13, 0}), 9, 10.5},
			{at(vec{shoulderOut - 4, trunkLen - shoulderDip + (p.shrugL+p.shrugR)/2, 0}),
				at(vec{-shoulderOut + 4, trunkLen - shoulderDip + (p.shrugL+p.shrugR)/2, 0}), 6, 6},
		}},
		// The neck is what shows which way the head leans.
		{bones: []bone{{head, head, headR, headR}, {neck, head, 4.4, 4.4}}},
	}

	var feet [2][2]vec // ankle and toe of each foot
	for i, sgn := range []float64{1, -1} {
		a, shrug, roll, l := p.armL, p.shrugL, p.rollL, p.legL
		if sgn < 0 {
			a, shrug, roll, l = p.armR, p.shrugR, p.rollR, p.legR
		}

		shoulder := at(vec{sgn * shoulderOut, trunkLen - shoulderDip + shrug, roll})
		upper := trunk.mul(rotY(sgn * a.open)).mul(rotX(-a.front)).mul(rotZ(sgn * a.side))
		elbow := shoulder.add(upper.apply(vec{0, -upperArmLen, 0}))
		fore := upper.mul(rotY(sgn * a.twist)).mul(rotX(-a.elbow))
		hand := elbow.add(fore.apply(vec{0, -forearmLen, 0}))
		limbs = append(limbs, limb{side: trunk.apply(vec{sgn, 0, 0}), bones: []bone{
			{shoulder, elbow, 6.4, 5.4},
			{elbow, hand, 5.4, 4.4},
			{hand, hand, 5.4, 5.4},
		}})

		hip := pelvis.add(vec{sgn * hipHalf, 0, 0})
		thigh := rotZ(sgn * l.side).mul(rotX(-l.front))
		knee := hip.add(thigh.apply(vec{0, -thighLen, 0}))
		ankle := knee.add(thigh.mul(rotX(l.knee)).apply(vec{0, -shinLen, 0}))
		// The foot keeps to the floor whatever the shin does, toes a little out.
		toe := ankle.add(rotY(sgn * 10).mul(rotX(l.toe)).apply(vec{0, -1.5, footLen}))
		feet[i] = [2]vec{ankle, toe}
		limbs = append(limbs, limb{side: vec{sgn, 0, 0}, bones: []bone{
			{hip, knee, 8.8, 6.8},
			{knee, ankle, 6.8, 5},
			{ankle, toe, ankleUp, 3.6},
		}})
	}

	// Stand it on the floor: the lowest point of either foot goes to zero, and
	// the foot that carries the weight stays where it is. A lifted foot counts
	// for less the higher it is, so marching shifts the body from one foot to
	// the other instead of sliding it about.
	low := func(f [2]vec) float64 { return min(f[0].y-ankleUp, f[1].y-3.6) }
	floor := min(low(feet[0]), low(feet[1]))
	var sx, sz, sw float64
	for _, f := range feet {
		h := (low(f) - floor) / 4
		w := 1 / (1 + h*h)
		sx += w * f[0].x
		sz += w * (f[0].z + footLen/3)
		sw += w
	}
	shift := vec{-sx / sw, -floor, -sz / sw}

	// Depth is taken before the viewer's height comes in: looking down makes
	// everything low seem further away, and the legs are not behind the trunk.
	turn := rotY(p.yaw)
	view := rotX(camPitch).mul(turn)
	type flat struct {
		part  Part
		depth float64
	}
	parts := make([]flat, 0, len(limbs))
	for _, l := range limbs {
		// The tone says which side of the body a limb is on, as the viewer
		// sees it: both the same from the front, far apart from the side.
		f := flat{part: Part{Tone: 0.5 + turn.apply(l.side).z/2}}
		for _, b := range l.bones {
			a, c := view.apply(b.a.add(shift)), view.apply(b.b.add(shift))
			f.part.Shapes = append(f.part.Shapes, Shape{a.x, a.y, b.ra, c.x, c.y, b.rb})
			f.depth += (turn.apply(b.a).z + turn.apply(b.b).z) / 2 / float64(len(l.bones))
		}
		parts = append(parts, f)
	}
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].depth < parts[j].depth })

	out := Frame{ShadowW: 84, ShadowH: 84*math.Sin(camPitch*math.Pi/180) + 5}
	for _, f := range parts {
		out.Parts = append(out.Parts, f.part)
	}
	return out
}

// Outline returns the shape as a closed polygon, grown by pad on every side,
// as x, y pairs. Every shape comes out wound the same way, so a set of them
// can be filled as one without holes where they overlap.
func (s Shape) Outline(pad float64) []float64 {
	const step = 15 // degrees per segment on the round ends
	r1, r2 := s.R1+pad, s.R2+pad
	dx, dy := s.X2-s.X1, s.Y2-s.Y1
	d := math.Hypot(dx, dy)
	var out []float64
	arc := func(cx, cy, r, from, to float64) {
		n := max(1, int(math.Ceil((to-from)/step)))
		for i := 0; i <= n; i++ {
			sin, cos := sincos(from + (to-from)*float64(i)/float64(n))
			out = append(out, cx+r*cos, cy+r*sin)
		}
	}
	// One end inside the other: only the bigger circle shows.
	if d <= math.Abs(r1-r2)+1e-6 {
		if r1 >= r2 {
			arc(s.X1, s.Y1, r1, 0, 360-step)
		} else {
			arc(s.X2, s.Y2, r2, 0, 360-step)
		}
		return out
	}
	// The straight sides touch both circles where the radius makes the same
	// angle with the axis: 90 degrees for a plain bar, less on the thick end.
	dir := math.Atan2(dy, dx) * 180 / math.Pi
	side := 90 - math.Asin((r1-r2)/d)*180/math.Pi
	arc(s.X1, s.Y1, r1, dir+side, dir+360-side)
	arc(s.X2, s.Y2, r2, dir-side, dir+side)
	return out
}
