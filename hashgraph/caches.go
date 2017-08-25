package hashgraph

import "github.com/babbleio/babble/common"

type Key struct {
	x string
	y string
}

//++++++++++++++++++++++++++++++++++++++++++++++++
//PARTICIPANT EVENTS CACHE
type ParticipantEventsCache struct {
	size              int
	participants      map[string]int //[public key] => id
	participantEvents map[string]*common.RollingList
}

func NewParticipantEventsCache(size int, participants map[string]int) *ParticipantEventsCache {
	items := make(map[string]*common.RollingList)
	for pk, _ := range participants {
		items[pk] = common.NewRollingList(size)
	}
	return &ParticipantEventsCache{
		size:              size,
		participants:      participants,
		participantEvents: items,
	}
}

func (pec *ParticipantEventsCache) Get(participant string, skip int) ([]string, error) {
	pe, ok := pec.participantEvents[participant]
	if !ok {
		return []string{}, ErrKeyNotFound
	}

	cached, tot := pe.Get()

	if skip >= tot {
		return []string{}, nil
	}

	oldestCached := tot - len(cached)
	if skip < oldestCached {
		//XXX TODO
		//LOAD REST FROM FILE
		return []string{}, ErrTooLate
	}

	//index of 'skipped' in RollingList
	start := skip - oldestCached

	if start >= len(cached) {
		return []string{}, nil
	}

	res := []string{}
	for k := start; k < len(cached); k++ {
		res = append(res, cached[k].(string))
	}
	return res, nil
}

func (pec *ParticipantEventsCache) GetItem(participant string, index int) (string, error) {
	res, err := pec.participantEvents[participant].GetItem(index)
	if err != nil {
		return "", err
	}
	return res.(string), nil
}

func (pec *ParticipantEventsCache) GetLast(participant string) (string, error) {
	pe, ok := pec.participantEvents[participant]
	if !ok {
		return "", ErrKeyNotFound
	}
	cached, _ := pe.Get()
	if len(cached) == 0 {
		return "", nil
	}
	last := cached[len(cached)-1]
	return last.(string), nil
}

func (pec *ParticipantEventsCache) Add(participant string, hash string) {
	pe, ok := pec.participantEvents[participant]
	if !ok {
		pe = common.NewRollingList(pec.size)
		pec.participantEvents[participant] = pe
	}
	pe.Add(hash)
}

func (pec *ParticipantEventsCache) Known() map[int]int {
	known := make(map[int]int)
	for p, evs := range pec.participantEvents {
		_, tot := evs.Get()
		known[pec.participants[p]] = tot
	}
	return known
}

func (pec *ParticipantEventsCache) Reset() error {
	items := make(map[string]*common.RollingList)
	for pk, _ := range pec.participants {
		items[pk] = common.NewRollingList(pec.size)
	}
	pec.participantEvents = items
	return nil
}
