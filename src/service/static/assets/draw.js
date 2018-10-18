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
    });

    layer = new Konva.Layer();

    legendLayer = new Konva.Layer();

    hgGroup = new Konva.Group({
        draggable: true,
        dragBoundFunc: pos => {
            let yPos = pos.y > 0 ? 0 : pos.y;

            return {
                x: 0,
                y: yPos,
            };
        },
    });

    hgBack = new Konva.Rect({
        x: 0,
        y: 0,
        width: window.innerWidth - 100,
        height: window.innerHeight,
    });

    blockGroup = new Konva.Group({
        draggable: true,
        dragBoundFunc: pos => {
            let yPos = pos.y > 0 ? 0 : pos.y;

            return {
                x: 0,
                y: yPos,
            };
        },
    });

    blockBack = new Konva.Rect({
        x: window.innerWidth - 100,
        y: 0,
        width: 100,
        height: window.innerHeight,
    });

    hgGroup.add(hgBack);

    blockGroup.add(blockBack);

    layer.add(hgGroup);
    layer.add(blockGroup);

    stage.add(layer);
    stage.add(legendLayer);
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

        hgGroup.add(nodeId);
    }

    // Set the border line if it contains a transaction
    if (event.Body.Transactions.length) {
        event.circle.setStroke('#000000');
        event.circle.setStrokeWidth(3);
    }

    hgGroup.add(event.circle);
    hgGroup.add(event.text);
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

        hgGroup.add(arrow);

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

            hgGroup.add(line);
            hgGroup.add(txt);
        });

    actualRound = rounds.length - 1;
};

let drawBlocks = blocks => {
    _.each(blocks, (block, bId) => {
        if (bId <= actualBlock) {
            return
        }

        let b = new Konva.Rect({
            x: window.innerWidth - 80,
            y: 20 + yInterval + (yInterval * bId),
            height: 20,
            width: 60,
            fill: '#999999',
            stroke: '#000000',
            strokeWidth: 1,
        });

        let txt = new Konva.Text({
            x: window.innerWidth - 60,
            y: 20 + yInterval + (yInterval * bId) + 5,
            text: '' + bId + ' (' + block.Body.Transactions.length + ')',
            fontSize: 12,
            fontFamily: 'Calibri',
            fill: 'black',
        });

        blockGroup.add(b, txt);
    })

    actualBlock = blocks.length - 1;
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

    hgBack.setHeight(100 + _.maxBy(events, ([eId, event]) => event.y)[1].y);

    layer.draw();
};

// Draw the legend
let drawLegend = () => {
    let background = new Konva.Rect({
        x: 0,
        y: 0,
        height: 40,
        width: window.innerWidth,
        fill: '#999999',
        stroke: '#000000',
        strokeWidth: 1,
    });

    let root = new Konva.Circle({
        x: 15,
        y: 20,
        radius: 10,
        fill: getEventColor({ Body: { Index: -1 } }),
    });

    let rootText = new Konva.Text({
        x: 25,
        y: 15,
        text: 'Root',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    let consensus = new Konva.Circle({
        x: 70,
        y: 20,
        radius: 10,
        fill: getEventColor({ Consensus: true }),
    });

    let consensusText = new Konva.Text({
        x: 80,
        y: 15,
        text: 'Consensus',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    let famous = new Konva.Circle({
        x: 160,
        y: 20,
        radius: 10,
        fill: getEventColor({ Famous: true }),
    });

    let famousText = new Konva.Text({
        x: 170,
        y: 15,
        text: 'Famous',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    let witness = new Konva.Circle({
        x: 240,
        y: 20,
        radius: 10,
        fill: getEventColor({ Witness: true }),
    });

    let witnessText = new Konva.Text({
        x: 250,
        y: 15,
        text: 'Witness',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    let normal = new Konva.Circle({
        x: 320,
        y: 20,
        radius: 10,
        fill: getEventColor({ Body: {} }),
    });

    let normalText = new Konva.Text({
        x: 330,
        y: 15,
        text: 'Normal',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    let tx = new Konva.Circle({
        x: 400,
        y: 20,
        radius: 10,
        fill: 'white',
        stroke: 'black',
        strokeWidth: 3,
    });

    let txText = new Konva.Text({
        x: 410,
        y: 15,
        text: 'Transaction',
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'black',
    });

    legendLayer.add(background);

    legendLayer.add(root);
    legendLayer.add(rootText);

    legendLayer.add(consensus);
    legendLayer.add(consensusText);

    legendLayer.add(famous);
    legendLayer.add(famousText);

    legendLayer.add(witness);
    legendLayer.add(witnessText);

    legendLayer.add(normal);
    legendLayer.add(normalText);

    legendLayer.add(tx);
    legendLayer.add(txText);

    legendLayer.draw();

    background.moveToBottom();
};
