package hashgraph

type Frame struct {
	Roots  map[string]Root
	Events []Event
}
