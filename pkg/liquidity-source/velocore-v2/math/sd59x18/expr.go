package sd59x18

type Expr struct {
	result *SD59x18
	err    error
}

func NewExpr(r *SD59x18) *Expr {
	return &Expr{
		result: r,
		err:    nil,
	}
}

func (e *Expr) Result() (*SD59x18, error) {
	return e.result, e.err
}

func (e *Expr) Add(x *SD59x18) *Expr {
	if e.err != nil {
		return e
	}

	e.result = new(SD59x18).Add(e.result, x)

	return e
}

func (e *Expr) Mul(x *SD59x18) *Expr {
	if e.err != nil {
		return e
	}

	e.result, e.err = new(SD59x18).Mul(e.result, x)

	return e
}

func (e *Expr) Log2() *Expr {
	if e.err != nil {
		return e
	}

	e.result, e.err = new(SD59x18).Log2(e.result)

	return e
}

func (e *Expr) Sub(x *SD59x18) *Expr {
	if e.err != nil {
		return e
	}

	e.result = new(SD59x18).Sub(e.result, x)

	return e
}

func (e *Expr) Exp2() *Expr {
	if e.err != nil {
		return e
	}

	e.result, e.err = new(SD59x18).Exp2(e.result)

	return e
}

func (e *Expr) Neg() *Expr {
	if e.err != nil {
		return e
	}

	e.result = new(SD59x18).Sub(Zero, e.result)

	return e
}

func (e *Expr) Div(x *SD59x18) *Expr {
	if e.err != nil {
		return e
	}

	e.result, e.err = new(SD59x18).Div(e.result, x)

	return e
}

func (e *Expr) SubExpr(other *Expr) *Expr {
	if e.err != nil {
		return e
	}

	if other.err != nil {
		return other
	}

	e.result = new(SD59x18).Sub(e.result, other.result)

	return e
}
