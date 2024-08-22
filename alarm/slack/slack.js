import * as https from 'https';
import * as util from 'util';

export const handler = (event, context) => {
    console.log('event: ', JSON.stringify(event));
    const content = {
        "channel": event.channel,
        "blocks": event.blocks
    };
    sendMessage(context, content);
};

function sendMessage(context, content) {
    const options = {
        method: 'POST',
        hostname: 'slack.com',
        port: 443,
        headers: {"Content-Type": "application/json", "Authorization": "Bearer " + process.env.TOKEN},
        path: "/api/chat.postMessage"
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