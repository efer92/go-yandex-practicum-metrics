package sample

// generate:reset
type ResetableStruct struct {
	i     int
	str   string
	flag  bool
	strP  *string
	s     []int
	m     map[string]string
	child *ResetableStruct
}

// generate:reset
type WithExternal struct {
	helper *PreExisting
	value  PreExisting
}

// Skipped because it has no marker.
type NotMarked struct {
	x int
}

// PreExisting already has a Reset method written by hand.
type PreExisting struct {
	v int
}

// Reset sets v to zero.
func (p *PreExisting) Reset() {
	if p == nil {
		return
	}
	p.v = 0
}
