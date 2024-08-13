import * as https from 'https';
import * as util from 'util';

export const handler = (event, context) => {
    console.log('event: ', JSON.stringify(event));
    const content = {
        "routing_key": process.env.ROUTING_KEY,
        "event_action": event.event_action,
        "payload": {
            "summary": event.payload.summary,
            "severity": event.payload.severity,
            "source": event.payload.source
        }
    };
    sendMessage(context, content);
};

function sendMessage(context, content) {
    const options = {
        method: 'POST',
        hostname: 'events.pagerduty.com',
        port: 443,
        headers: {"Content-Type": "application/json"},
        path: "/v2/enqueue"
    };

    const req = https.request(options, (res) => {
        res.setEncoding('utf8');
        res.on('data', (chunk) => {
            context.done(null);
        });
    });

    req.on('error', function (e) {
        console.log('problem with request: ' + e.message);
    });

    req.write(util.format("%j", content));
    req.end();
}