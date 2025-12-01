package orderbook

type (
	Gas struct{ Base, Level int64 }

	Level [2]float64 // [size, price]

	Extra struct {
		LevelsFrom [2][]Level `json:"l"`
	}
)

func (l *Level) Size() float64 {
	return l[0]
}

func (l *Level) SetSize(s float64) {
	l[0] = s
}

func (l *Level) Price() float64 {
	return l[1]
}
