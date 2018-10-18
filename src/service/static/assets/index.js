// Globals
let stage;
let layer;

let events = [];       // [[id, event]]
let participants = {}; // {hash: id}

let xInterval = 80;
let yInterval = 30;

let intervalHandler;

let actualRound = -1;

// Main loop
let loop = () => {
    fetch("/graph")
        .then(res => res.json())
        .then(data => {
            let newEvents = filterPopulate(data.ParticipantEvents);

            assignRound(data.Rounds);

            _.each(newEvents, assignParents);

            processParents(newEvents);

            draw(newEvents);

            drawRoundLines(data.Rounds);
        })
        .catch(err => {
            console.log("ERROR: fetch", err);

            clearInterval(intervalHandler);
        });
};

// Main function
let main = () => {
    setupStage();

    drawLegend();

    intervalHandler = setInterval(loop, 1000);
};

main();
