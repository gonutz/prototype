package shapes

type ScaledShape struct {
	*Basic
	sizeScale float32
}

func newScaledShape(name string) *ScaledShape {
	return &ScaledShape{Basic: newBasic(name), sizeScale: float32(3)}
}

func (s *ScaledShape) SetScale(scale float32) {
	s.sizeScale=scale
}