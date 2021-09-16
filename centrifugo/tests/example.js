import http from 'k6/http';
import { sleep } from 'k6';
import { check } from 'k6';
import ws from "k6/ws";
//import Centrifuge from 'https://cdn.jsdelivr.net/gh/centrifugal/centrifuge-js@2.8.2/dist/centrifuge.js';
//import Centrifuge from "/node_modules/centrifuge/src/centrifuge.js";
//import promise from 'https://cdn.jsdelivr.net/npm/es6-promise@4/dist/es6-promise.auto.min.js';

export default function () {
    const url = 'ws://host.docker.internal:8000/connection/websocket';

    const res = ws.connect(url, null, function (socket) {
        socket.on('open', function () {
            console.log('connected')
            socket.send('{"params":{"name":"js"},"id":1}')
            socket.send('{"method":1,"params":{"channel":"chat"},"id":2}')

            socket.setInterval(function () {
                socket.send('{"method":3,"params":{"channel":"chat","data":"test message"},"id":3}');
            }, 500);

            console.log('after send')
        });
        socket.on('message', (data) => console.log('Message received: ', data));
        socket.on('close', () => console.log('disconnected'));

        socket.setTimeout(function () {
            socket.close();
        }, 10000);
    });

    check(res, { 'status is 101': (r) => r && r.status === 101 });
}
