package draw

import "testing"

func Test_ellipse_with_0_size_is_empty(t *testing.T) {
	if ellipsePoints(5, 5, 0, 5) != nil {
		t.Error("0 width ellipse not nil")
	}
	if ellipsePoints(5, 5, 5, 0) != nil {
		t.Error("0 height ellipse not nil")
	}
}

func Test_ellipse_with_size_1_yields_one_point_line(t *testing.T) {
	checkPoints(t,
		ellipsePoints(3, 4, 1, 1),
		p(3, 4), p(3, 4))
}

func Test_3x3_circle(t *testing.T) {
	checkPoints(t,
		ellipsePoints(1, 2, 3, 3),
		p(2, 2), p(2, 2),
		p(1, 3), p(3, 3),
		p(2, 4), p(2, 4))
}

func checkPoints(t *testing.T, actual []point, expected ...point) {
	if len(actual) != len(expected) {
		t.Errorf("wrong points\n%v expected\n%v gotten", expected, actual)
	}
}

func p(x, y int) point { return point{x, y} }
