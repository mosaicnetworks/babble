package hashgraph

type Frame struct {
	Round  int
	Roots  map[string]Root
	Events []Event
}
