import { check, sleep } from 'k6';
import ws from "k6/ws";
import crypto from "k6/crypto";
import encoding from "k6/encoding";

export let options = {
    scenarios: {
        contacts: {
            executor: 'constant-vus',
            vus: 1000,
            duration: '10s',
            gracefulStop: '10s',
        },
    }
    /*stages: [
        { duration: "5s", target: 100},
    ],*/
};

let channel = 'chat';
const url = 'ws://host.docker.internal:8000/connection/websocket';

export default function () {
    const res = ws.connect(url, null, function (socket) {
        socket.on('open', function () {
            let payload = {"sub": `${__VU}`, "exp": 9590186316}
            let jwtToken = encode(payload, "test")

            socket.send(`{"params":{"token":"${jwtToken}","name":"js"},"id":1}`)
            socket.send(`{"method":1,"params":{"channel":"${channel}"},"id":2}`)

            socket.setInterval(function () {
                socket.send(`{"method":3,"params":{"channel":"${channel}","data":"test message"},"id":3}`);
            }, 500);
        });

        //socket.on('message', (data) => console.log('Message received: ', data));
        //socket.on('close', () => console.log('disconnected'));

        socket.setTimeout(function () {
            socket.close();
        }, 15000);
    });

    //check(res, { 'status is 101': (r) => r && r.status === 101 });
}

function encode(payload, secret) {
    let header = encoding.b64encode(JSON.stringify({ typ: "JWT", alg: "HS256" }), "rawurl");
    payload = encoding.b64encode(JSON.stringify(payload), "rawurl");
    let sig = sign(header + "." + payload, secret);
    return [header, payload, sig].join(".");
}

function sign(data, secret) {
    let hasher = crypto.createHMAC("sha256", secret);
    hasher.update(data);
    return hasher.digest("base64").replace(/\//g, "_").replace(/\+/g, "-").replace(/=/g, "");
}