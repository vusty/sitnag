package figure

import "math"

// Count is how many moves there are. The texts that go with them live with
// the rest of the interface strings, in the same order as Move.
const Count = 10

// Move numbers a move. The order is the order of a warm-up: from the head
// down, the loosening first and the legs last.
type Move int

const (
	Neck Move = iota
	Shoulders
	ArmCircles
	Reach
	SideBend
	Twist
	ForwardBend
	Squat
	CalfRaise
	March
)

// move is one exercise: how long one repetition takes, and the pose at each
// point of it, phase running from 0 to 1.
type move struct {
	period float64 // seconds
	pose   func(phase float64) pose
}

var moves = [Count]move{
	Neck:        {5, neck},
	Shoulders:   {2.6, shoulders},
	ArmCircles:  {3, armCircles},
	Reach:       {6, reach},
	SideBend:    {6, sideBend},
	Twist:       {3.4, twist},
	ForwardBend: {7, forwardBend},
	Squat:       {3.8, squat},
	CalfRaise:   {2.4, calfRaise},
	March:       {1.5, march},
}

// swing goes 0, 1, 0, -1 and back to 0 over one repetition.
func swing(phase float64) float64 { return math.Sin(2 * math.Pi * phase) }

// wave goes from 0 up to 1 and back, with no pause at either end.
func wave(phase float64) float64 { return (1 - math.Cos(2*math.Pi*phase)) / 2 }

// ease is a smooth step from 0 to 1.
func ease(x float64) float64 {
	x = min(1, max(0, x))
	return x * x * (3 - 2*x)
}

// pulse rises from 0 to 1 over the first rise of the repetition, stays there
// for hold, comes back down over fall, and rests for what is left.
func pulse(phase, rise, hold, fall float64) float64 {
	switch {
	case phase < rise:
		return ease(phase / rise)
	case phase < rise+hold:
		return 1
	default:
		return 1 - ease((phase-rise-hold)/fall)
	}
}

// standing is the pose every move starts from: upright, arms loose.
func standing(yaw float64) pose {
	loose := arm{side: 7, elbow: 10}
	return pose{yaw: yaw, armL: loose, armR: loose}
}

// onHips puts both hands on the hips.
func onHips(p *pose) {
	hip := arm{side: 30, front: -12, twist: -80, elbow: 78}
	p.armL, p.armR = hip, hip
}

func neck(ph float64) pose {
	p := standing(-8)
	p.tilt = 40 * swing(ph)
	// The shoulder on the far side drops a little, the way it does.
	p.shrugL = -1.5 * max(0, -swing(ph))
	p.shrugR = -1.5 * max(0, swing(ph))
	return p
}

func shoulders(ph float64) pose {
	p := standing(-30)
	// Forward, up, back, down: a circle backwards, drawn bigger than life so
	// it can be seen at all.
	sin, cos := math.Sincos(2 * math.Pi * ph)
	p.shrugL, p.shrugR = 3+9*sin, 3+9*sin
	p.rollL, p.rollR = 8*cos, 8*cos
	a := arm{side: 7, front: -8 * cos, elbow: 14 + 10*sin}
	p.armL, p.armR = a, a
	return p
}

func armCircles(ph float64) pose {
	p := standing(-52)
	a := arm{side: 12, front: 360 * ph, elbow: 6}
	p.armL, p.armR = a, a
	return p
}

func reach(ph float64) pose {
	p := standing(-14)
	up := pulse(ph, 0.3, 0.3, 0.3)
	a := arm{side: 7 + 166*up, elbow: 10 - 6*up}
	p.armL, p.armR = a, a
	// Onto the toes at the top of the stretch.
	toes := 40 * ease((up-0.55)/0.45)
	p.legL.toe, p.legR.toe = toes, toes
	p.nod = -10 * up
	return p
}

func sideBend(ph float64) pose {
	p := standing(0)
	s := swing(ph)
	p.lean = 27 * s
	p.tilt = 8 * s
	// The arm on the outside of the bend goes over the head.
	over := func(k float64) arm {
		k = ease(k)
		return arm{side: 7 + 170*k, twist: 85 * k, elbow: 10 + 28*k}
	}
	p.armL, p.armR = over(-s), over(s)
	return p
}

func twist(ph float64) pose {
	p := standing(-16)
	p.twist = 58 * swing(ph)
	// Arms out and a little bent: they sweep round and show the turn.
	a := arm{side: 80, front: 8, twist: 0, elbow: 32}
	p.armL, p.armR = a, a
	return p
}

func forwardBend(ph float64) pose {
	p := standing(-62)
	down := pulse(ph, 0.32, 0.3, 0.28)
	p.bend = 94 * down
	p.nod = 14 * down
	// The hips go back to keep the balance, the arms hang to the floor.
	p.legL.front, p.legR.front = -9*down, -9*down
	a := arm{side: 7, front: 92 * down, elbow: 10 - 6*down}
	p.armL, p.armR = a, a
	return p
}

func squat(ph float64) pose {
	p := standing(-50)
	down := wave(ph)
	l := leg{side: 6, front: 80 * down, knee: 104 * down}
	p.legL, p.legR = l, l
	p.bend = 40 * down
	p.nod = -18 * down
	a := arm{side: 7, front: 128 * down, elbow: 10 - 4*down}
	p.armL, p.armR = a, a
	return p
}

func calfRaise(ph float64) pose {
	p := standing(-58)
	onHips(&p)
	up := pulse(ph, 0.3, 0.2, 0.3)
	p.legL.toe, p.legR.toe = 44*up, 44*up
	return p
}

func march(ph float64) pose {
	p := standing(-42)
	s := swing(ph)
	lift := func(k float64) leg {
		k = ease(max(0, k))
		return leg{front: 78 * k, knee: 86 * k, toe: 18 * k}
	}
	p.legL, p.legR = lift(s), lift(-s)
	// Opposite arm to the knee that is up.
	p.armL = arm{side: 6, front: -34 * s, elbow: 62}
	p.armR = arm{side: 6, front: 34 * s, elbow: 62}
	return p
}
