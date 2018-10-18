// Axis 
// ---- x
// |
// | y

// Setup the scene
let setupStage = () => {
    stage = new Konva.Stage({
        container: 'container',
        width: window.innerWidth,
        height: window.innerHeight,
        draggable: true,
    });

    layer = new Konva.Layer();

    stage.add(layer);
};

// Return the color of the event
let getEventColor = event => {
    let color = '#555555';

    if (event.Famous) {
        color = '#00ffff';
    } else if (event.Witness) {
        color = '#5555ff';
    } else if (event.Consensus) {
        color = '#00ff00';
    } else if (event.Body.Index === -1) {
        color = '#ff0000';
    }

    return color;
};

// Draw the event, the associated index text
// and set the borderLine (stroke) if it contains a Tx
let drawEvent = event => {
    // The event circle
    event.circle = new Konva.Circle({
        x: event.x,
        y: event.y,
        radius: 10,
        fill: getEventColor(event),
    });

    // The event index
    event.text = new Konva.Text({
        x: event.x - 5,
        y: event.y - 5,
        text: event.Body.Index === -1 ? '' : '' + event.Body.Index,
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    // If its root, draw the NodeId text
    if (event.Body.Index === -1) {
        let nodeId = new Konva.Text({
            x: event.x - xInterval / 2,
            y: 50,
            text: event.EventId.replace('Root', ''),
            fontSize: 12,
            fontFamily: 'Calibri',
            fill: 'black',
        });

        layer.add(nodeId);
    }

    // Set the border line if it contains a transaction
    if (event.Body.Transactions.length) {
        event.circle.setStroke('#000000');
        event.circle.setStrokeWidth(3);
    }

    layer.add(event.circle);
    layer.add(event.text);
};

// Draw the links between an event and its parents
let drawEventLinks = event => {
    _.each(event.ParentEvents, parentEvent => {
        let arrow = new Konva.Arrow({
            points: [
                parentEvent.x,
                parentEvent.y,
                event.x,
                event.y,
            ],
            pointerLength: 5,
            pointerWidth: 5,
            fill: 'black',
            stroke: 'black',
            strokeWidth: 1,
        });

        layer.add(arrow);

        arrow.moveToBottom();
    });
};

// Draw the round separators
let drawRoundLines = rounds => {
    // Dirty tmp fix for events that are in the first and second round
    if (rounds.length >= 2) {
        rounds[1].Events = _.fromPairs(_.differenceBy(_.toPairs(rounds[1].Events), _.toPairs(rounds[0].Events), ([rId, round]) => rId));
    }

    _(rounds)
        .each((round, rId) => {
            // Dont draw existing round
            if (rId <= actualRound) {
                return;
            }

            let roundEvents = [];

            _.forIn(round.Events, (event, reId) => {
                roundEvents.push(_.find(events, ([eId, e]) => eId === reId))
            });

            let [eId, ev] = _.minBy(roundEvents, ([eId, ev]) => ev.y);

            let line = new Konva.Line({
                points: [
                    0,
                    ev.y - yInterval / 2,
                    (_.keys(participants).length + 1) * xInterval,
                    ev.y - yInterval / 2,
                ],
                fill: 'black',
                stroke: 'black',
                strokeWidth: 1,
            });

            let txt = new Konva.Text({
                x: (_.keys(participants).length + 1) * xInterval,
                y: ev.y - yInterval / 2 - 6,
                text: '' + rId,
                fontSize: 12,
                fontFamily: 'Calibri',
                fill: 'black',
            });

            layer.add(line);
            layer.add(txt);
        });

    actualRound = rounds.length - 1;
};

// Main draw function
let draw = evs => {
    _.each(evs, ([eId, event]) => {
        drawEvent(event);

        if (event.ParentEvents.length === 0) {
            return;
        }

        drawEventLinks(event);
    });

    layer.draw();
};

// Draw the legend
let drawLegend = () => {
    let root = new Konva.Circle({
        x: 10,
        y: 10,
        radius: 10,
        fill: getEventColor({ Body: { Index: -1 } }),
    });

    let rootText = new Konva.Text({
        x: 20,
        y: 5,
        text: 'Root',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    let consensus = new Konva.Circle({
        x: 70,
        y: 10,
        radius: 10,
        fill: getEventColor({ Consensus: true }),
    });

    let consensusText = new Konva.Text({
        x: 80,
        y: 5,
        text: 'Consensus',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    let famous = new Konva.Circle({
        x: 160,
        y: 10,
        radius: 10,
        fill: getEventColor({ Famous: true }),
    });

    let famousText = new Konva.Text({
        x: 170,
        y: 5,
        text: 'Famous',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    let witness = new Konva.Circle({
        x: 240,
        y: 10,
        radius: 10,
        fill: getEventColor({ Witness: true }),
    });

    let witnessText = new Konva.Text({
        x: 250,
        y: 5,
        text: 'Witness',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    let normal = new Konva.Circle({
        x: 320,
        y: 10,
        radius: 10,
        fill: getEventColor({ Body: {} }),
    });

    let normalText = new Konva.Text({
        x: 330,
        y: 5,
        text: 'Normal',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    let tx = new Konva.Circle({
        x: 400,
        y: 10,
        radius: 10,
        fill: 'white',
        stroke: 'black',
        strokeWidth: 3,
    });

    let txText = new Konva.Text({
        x: 410,
        y: 5,
        text: 'Transaction',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    layer.add(root);
    layer.add(rootText);

    layer.add(consensus);
    layer.add(consensusText);

    layer.add(famous);
    layer.add(famousText);

    layer.add(witness);
    layer.add(witnessText);

    layer.add(normal);
    layer.add(normalText);

    layer.add(tx);
    layer.add(txText);
};
