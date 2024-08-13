import * as https from 'https';
import * as util from 'util';

export const handler = (event, context) => {
    console.log('event: ', JSON.stringify(event));
    const content = {
        "chat_id": event.message.chat,
        "text": event.message.text,
        "parse_mode": "HTML"
    };
    sendMessage(context, content);
};

function sendMessage(context, content) {
    const options = {
        method: 'POST',
        hostname: 'api.telegram.org',
        port: 443,
        headers: {"Content-Type": "application/json"},
        path: "/bot" + process.env.TOKEN + "/sendMessage"
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