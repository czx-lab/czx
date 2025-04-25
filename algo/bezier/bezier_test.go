package bezier

import "testing"

func TestCurve(t *testing.T) {
	t.Run("TestCurve", func(t *testing.T) {
		b := New(BezierConf{
			ScreenW: 1000,
			ScreenH: 1000,
		})

		ctrls := b.Ctrls(CtrlArgs{
			N:      10,
			Factor: 0.5,
		})

		pts := b.Points(ctrls, Forward)

		t.Logf("Points: %v", pts)
	})
}
