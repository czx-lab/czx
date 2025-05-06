package bezier

import (
	"math/rand/v2"
	"time"
)

const (
	_ Edge = iota
	Left
	Right
	Top
	Bottom
	TopLeft
	TopRight
	BottomLeft
	BottomRight

	// forward direction
	Forward Direction = 1.0
	// backward direction
	Reverse Direction = -1.0
)

type (
	// edge of the screen
	Edge uint8
	// direction of the bezier curve
	Direction float64
	// x, y coordinates
	// of a point in the bezier curve
	// and the screen
	Point struct {
		X float64
		Y float64
	}
	BezierConf struct {
		ScreenW float64
		ScreenH float64
	}
	// CtrlArgs is the arguments for the Ctrls function
	// SepCtrls is the start and end points of the bezier curve
	CtrlArgs struct {
		SepCtrls [2]Point
		N        uint
		Factor   float64
	}
	// curveCtrl is the control point of the bezier curve
	// Point is the point on the bezier curve
	curveCtrl struct {
		Point Point
		Ctrl  Point
	}
	Bezier struct {
		conf         BezierConf
		curveCtrls   []curveCtrl
		speedHandler func(curvature float64) float64
	}
)

func New(conf BezierConf) *Bezier {
	return &Bezier{conf: conf}
}

// WithSpeedHandler sets the speed handler for the bezier curve
// The handler should return the speed based on the curvature of the bezier curve
func (b *Bezier) WithSpeedHandler(handler func(curvature float64) float64) {
	b.speedHandler = handler
}

// Ctrls generates the control points of the bezier curve
// based on the given arguments
func (b *Bezier) Ctrls(args CtrlArgs) (points []Point) {
	points = make([]Point, 0, args.N)

	// add the first control point
	if len(args.SepCtrls) > 0 {
		points = append(points, args.SepCtrls[0])
	}

	for i := range args.N {
		t := float64(i) / float64(args.N-1) // [0, 1]
		var prev Point
		if i > 0 {
			prev = points[i-1]
		}

		// generate a random point on the bezier curve
		// based on the previous point
		points = append(points, Point{
			X: prev.X*(1-t) + b.conf.ScreenW*t*rand.Float64(),
			Y: prev.Y*(1-t) + b.conf.ScreenH*t*rand.Float64(),
		})
	}

	// add the last control point
	if len(args.SepCtrls) == 2 {
		points = append(points, args.SepCtrls[1])
	}

	// smooth the points
	if args.Factor > 0 {
		points = b.smooth(points, args.Factor)
	}

	b.CurveCtrl(points)
	return
}

// CurvePoints generates the points of the bezier curve
// based on the given points
func (b *Bezier) CurvePoints(points []Point) (pts []Point) {
	if len(points) < 2 {
		return nil
	}

	b.curveCtrls = make([]curveCtrl, len(points))

	b.CurveCtrl(points)

	pts = make([]Point, len(points))

	return b.curve(pts)
}

func (b *Bezier) curve(p []Point) []Point {
	for i, nf := 0, float64(len(p)-1); i < len(p); i++ {
		p[i] = b.Point(float64(i) / nf)
	}
	return p
}

// Points generates the points of the bezier curve
// based on the given points and direction
func (b *Bezier) Points(points []Point, direction Direction) (pts []Point) {
	var t float64
	endCtrl := points[len(points)-1]
	for {
		point := b.Point(t)
		pts = append(pts, point)

		// calculate the speed based on the curvature
		t += b.calculateSpeed(points, point, direction)

		if t >= 1.0 || t <= 0.0 {
			pts = append(pts, endCtrl)
			break
		}

		time.Sleep(30 * time.Millisecond)
	}

	return
}

// calculateSpeed calculates the speed of the bezier curve
// based on the curvature of the bezier curve
func (b *Bezier) calculateSpeed(points []Point, current Point, direction Direction) float64 {
	var curvature float64

	if len(points) == 2 {
		curvature = b.calcCurvature(points[0], points[1], current)
	} else {
		curvature = b.calcCurvature(points[0], points[1], points[2])
	}

	var speed float64
	if b.speedHandler != nil {
		speed = b.speedHandler(curvature)
	} else {
		speed = defaultSpeedHandler(curvature)
	}

	return float64(direction) * speed
}

func defaultSpeedHandler(curvature float64) float64 {
	return max(0.01/(1+curvature*10), 0.001)
}

// calcCurvature calculates the curvature of the bezier curve
// based on the three points of the bezier curve
func (b *Bezier) calcCurvature(p1, p2, p3 Point) float64 {
	dx1 := p2.X - p1.X
	dy1 := p2.Y - p1.Y
	dx2 := p3.X - p2.X
	dy2 := p3.Y - p2.Y

	cross := dx1*dy2 - dy1*dx2
	dot := dx1*dx2 + dy1*dy2

	curvature := max(min(cross/(dot+1e-6), 10), -10)
	return curvature
}

// Point generates the point of the bezier curve
// based on the given t value
// de Casteljau algorithm
// https://en.wikipedia.org/wiki/De_Casteljau%27s_algorithm
func (b *Bezier) Point(t float64) Point {
	b.curveCtrls[0].Point = b.curveCtrls[0].Ctrl
	u := t
	for i, p := range b.curveCtrls[1:] {
		b.curveCtrls[i+1].Point = Point{
			X: p.Ctrl.X * u,
			Y: p.Ctrl.Y * u,
		}
		u *= t
	}

	var (
		t1 = 1 - t
		tt = t1
	)
	p := b.curveCtrls[len(b.curveCtrls)-1].Point
	for i := len(b.curveCtrls) - 2; i >= 0; i-- {
		p.X += b.curveCtrls[i].Point.X * tt
		p.Y += b.curveCtrls[i].Point.Y * tt
		tt *= t1
	}

	return p
}

// generate a random point on the edge of the screen
// based on the given edge and offset
// offset is the distance from the edge
func (b *Bezier) SepCtrl(edge Edge, offset float64) (point Point) {
	switch edge {
	case Left:
		point = Point{X: 0, Y: rand.Float64() * b.conf.ScreenH}
	case Right:
		point = Point{X: b.conf.ScreenW, Y: rand.Float64() * b.conf.ScreenH}
	case Top:
		point = Point{X: rand.Float64() * b.conf.ScreenW, Y: 0}
	case Bottom:
		point = Point{X: rand.Float64() * b.conf.ScreenW, Y: b.conf.ScreenH}
	case TopLeft:
		point = Point{X: 0, Y: 0}
		if offset > 0 {
			point.X = b.conf.ScreenW * offset * rand.Float64()
		}
	case TopRight:
		point = Point{X: b.conf.ScreenW, Y: 0}
		if offset > 0 {
			point.X = b.conf.ScreenW*(1-offset) + rand.Float64()*b.conf.ScreenW*offset
		}
	case BottomLeft:
		point = Point{X: 0, Y: b.conf.ScreenH}
		if offset > 0 {
			point.X = b.conf.ScreenW * offset * rand.Float64()
		}
	case BottomRight:
		point = Point{X: b.conf.ScreenW, Y: b.conf.ScreenH}
		if offset > 0 {
			point.X = b.conf.ScreenW*(1-offset) + rand.Float64()*b.conf.ScreenW*offset
		}
	default:
		point = Point{X: 0, Y: 0}
	}

	return
}

func (b *Bezier) smooth(points []Point, factor float64) (smoothed []Point) {
	if len(points) < 3 {
		return points
	}

	smoothed = make([]Point, len(points))
	for i := range points {
		if i == 0 || i == len(points)-1 {
			smoothed[i] = points[i]
			continue
		}

		smoothed[i] = Point{
			X: points[i].X*(1-factor) + (points[i-1].X+points[i+1].X)*factor/2,
			Y: points[i].Y*(1-factor) + (points[i-1].Y+points[i+1].Y)*factor/2,
		}
	}
	return
}

func (b *Bezier) CurveCtrl(points []Point) {
	curve := make([]curveCtrl, len(points))

	for i, p := range points {
		curve[i].Point = p
	}

	var w float64
	for i, p := range points {
		switch i {
		case 0:
			w = 1
		case 1:
			w = float64(len(points)) - 1
		default:
			w *= float64(len(points)-i) / float64(i)
		}
		curve[i].Ctrl.X = p.X * w
		curve[i].Ctrl.Y = p.Y * w
	}

	b.curveCtrls = curve
}
