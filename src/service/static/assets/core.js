let setInitial = pId => ([eId, event]) => {
    event.EventId = eId;
    event.ParticipantHash = pId;

    event.x = xInterval + (participants[pId] * xInterval);
    event.y = -1;

    if (event.Body.Index === -1) {
        event.y = 80;
    }

    events.push([eId, event]);

    return [eId, event];
};

// Set the initial event pos
// Returns only unknown events
let processParticipant = ([pId, participant]) => {
    // Assign line ID
    if (participants[pId] == null) {
        participants[pId] = _.values(participants).length;
    }

    // Get new Events only
    // Sort by Index
    // Assign base position
    return _(participant)
        .toPairs()
        .differenceBy(events, ([eId, event]) => eId)
        .sortBy(([eId, ev]) => ev.Body.Index)
        .map(setInitial(pId))
        .value();
}

// Loop through the participants
let filterPopulate = participantsEvents => {
    return _(participantsEvents)
        .toPairs()
        .map(processParticipant)
        .flatten()
        .value();
};

// Find and assign parents
let assignParents = ([eId, event]) => {
    event.ParentEvents = event.ParentEvents || [];

    _.each(event.Body.Parents, parentId => {
        let parentEvent = _.find(events, ([eId, event]) => eId === parentId);

        if (parentEvent == null) {
            return;
        }

        event.ParentEvents.push(parentEvent[1]);
    });
};

let setYPos = ([eId, event]) => {
    if (_.every(event.ParentEvents, ev => ev.y !== -1)) {
        let higherParent = _.maxBy(event.ParentEvents, ev => ev.y);

        event.y = higherParent.y + yInterval;

        // fix the position if in another round than a Y neighbour
        let neighbour = _.find(events, ([eId, ev]) => ev.y === event.y && eId !== event.EventId && ev.Round < event.Round)

        if (neighbour != null) {
            event.y = neighbour[1].y + yInterval;
        }
    }

    return [eId, event];
};

// Loop through the unknown events and gradualy set y position
// only when parents are both already set
let processParents = evs => {
    let processed = [];

    evs = _.filter(evs, ([eId, event]) => event.ParentEvents.length && event.y == -1);

    let lastLength = evs.length;

    while (evs.length) {
        evs = _(evs)
            .map(setYPos)
            .filter(([eId, event]) => event.y === -1)
            .value();

        if (evs.length === lastLength) {
            console.log("ERROR: nothing more to do", lastLength, evs);

            return;
        }

        lastLength = evs.length;
    }
};

// Assign the events round info
let assignRound = (rounds) => {
    _(rounds).each((round, rId) => {
        _.forIn(round.Events, (roundEvent, reId) => {
            let eventInfos = _.find(events, ([eId, _]) => eId === reId)

            if (eventInfos == null) {
                console.log('ERROR: Unknown event', reId);

                return
            }

            let event = eventInfos[1];

            event.Round = rId;
            event.Consensus = roundEvent.Consensus;
            event.Witness = roundEvent.Witness;
            event.Famous = !!roundEvent.Famous;

            if (event.circle != null) {
                event.circle.setFill(getEventColor(event));
            }
        })
    });
};
