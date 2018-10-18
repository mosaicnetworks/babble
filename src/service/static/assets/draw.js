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
            let yPos = pos.y < -hgBack.getHeight() + window.innerHeight ? -hgBack.getHeight() + window.innerHeight : pos.y;

            yPos = yPos > 0 ? 0 : yPos;

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
            let yPos = pos.y < -blockBack.getHeight() + window.innerHeight ? -blockBack.getHeight() + window.innerHeight : pos.y;

            yPos = yPos > 0 ? 0 : yPos;

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
        x: event.x + 15,
        y: event.y - 5,
        text: event.Body.Index === -1 ? '' : '' + event.Body.Index,
        fontSize: 12,
        fontFamily: 'Calibri',
        fill: 'white',
    });

    event.text.hide();

    // If its root, draw the NodeId text
    if (event.Body.Index === -1) {
        let nodeId = new Konva.Text({
            x: event.x - xInterval / 2,
            y: 50,
            text: event.EventId.replace('Root', ''),
            fontSize: 12,
            fontFamily: 'Calibri',
            fill: 'white',
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

    hgGroup.setY(_.min([0, hgGroup.getY(), -event.y + window.innerHeight - 100]));
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
                stroke: 'white',
                strokeWidth: 2,
            });

            let txt = new Konva.Text({
                x: 5 + (_.keys(participants).length + 1) * xInterval,
                y: ev.y - yInterval / 2 - 6,
                text: '' + rId,
                fontSize: 12,
                fontFamily: 'Calibri',
                fill: 'white',
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

        blockGroup.setY(_.min([0, blockGroup.getY(), -(40 + yInterval + (yInterval * bId) + 5) + window.innerHeight]));
    })

    actualBlock = blocks.length - 1;

    blockBack.setHeight((50 + yInterval + (yInterval * actualBlock + 1)));
};

// Main draw function
let draw = (evs, rounds, blocks) => {
    _.each(evs, ([eId, event]) => {
        drawEvent(event);

        if (event.ParentEvents.length === 0) {
            return;
        }

        drawEventLinks(event);
    });

    drawRoundLines(rounds);

    drawBlocks(blocks)

    layer.draw();

    let maxY = _.maxBy(events, ([eId, event]) => event.y)[1].y;

    hgBack.setHeight(100 + maxY);
};

// Draw the legend
let drawLegend = () => {
    let legends = [
        {
            name: 'Root',
            color: getEventColor({ Body: { Index: -1 } }),
        },
        {
            name: 'Consensus',
            color: getEventColor({ Consensus: true }),
        },
        {
            name: 'Famous',
            color: getEventColor({ Famous: true }),
        },
        {
            name: 'Witness',
            color: getEventColor({ Witness: true }),
        },
        {
            name: 'Normal',
            color: getEventColor({ Body: {} }),
        },
        {
            name: 'Transaction',
            color: '#999999',
        },
    ];

    _.each(legends, (legend, i) => {
        let circle = new Konva.Circle({
            x: 15 + (i * 100),
            y: 20,
            radius: 10,
            fill: legend.color,
        });

        if (legend.name === 'Transaction') {
            circle.setStroke('black');
            circle.setStrokeWidth(3);
        }

        let text = new Konva.Text({
            x: 30 + (i * 100),
            y: 15,
            text: legend.name,
            fontSize: 12,
            fontFamily: 'Calibri',
            fill: 'black',
        });

        legendLayer.add(circle, text);
    });

    let background = new Konva.Rect({
        x: 0,
        y: 0,
        height: 40,
        width: window.innerWidth,
        fill: '#999999',
        stroke: '#000000',
        strokeWidth: 1,
    });

    legendLayer.add(background);
    background.moveToBottom();

    legendLayer.draw();
};

// Draw the legend
let drawSettings = () => {
    let settings = [
        {
            label: 'Show event ids',
            name: 'showEventIds',
            trigger: () => _.each(events, ([eId, event]) => settingValues.showEventIds ? event.text.show() : event.text.hide()),
        }
    ];

    let toggle = (setting) => {
        settingValues[setting.name] = !settingValues[setting.name];

        setting.rect.setFill(getSettingColor(settingValues[setting.name]));

        setting.trigger();

        legendLayer.draw();
    };

    let getSettingColor = (val) => val ? '#00ff00' : '#ff0000';

    _.each(settings, (setting, i) => {
        setting.rect = new Konva.Rect({
            x: 800 + 15 + (i * 100),
            y: 10,
            width: 20,
            height: 20,
            fill: getSettingColor(settingValues[setting.name]),
            stroke: 'black',
        });

        setting.text = new Konva.Text({
            x: 800 + 40 + (i * 100),
            y: 15,
            text: setting.label,
            fontSize: 12,
            fontFamily: 'Calibri',
            fill: 'black',
        });

        legendLayer.add(setting.rect, setting.text);

        setting.rect.on('mousedown', () => toggle(setting))
    });

    legendLayer.draw();
};