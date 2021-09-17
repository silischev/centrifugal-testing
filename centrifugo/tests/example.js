import { check, sleep } from 'k6';
import ws from "k6/ws";

export let options = {
    vus: 2,
    duration: '1s',
};

let channel = 'chat';
const url = 'ws://host.docker.internal:8000/connection/websocket';

export default function () {
    const res = ws.connect(url, null, function (socket) {
        socket.on('open', function () {
            //console.log('connected')

            socket.send('{"params":{"name":"js"},"id":1}')
            socket.send(`{"method":1,"params":{"channel":"${channel}"},"id":2}`)

            socket.setInterval(function () {
                socket.send(`{"method":3,"params":{"channel":"${channel}","data":"test message"},"id":3}`);
            }, 500);
        });

        //socket.on('message', (data) => console.log('Message received: ', data));
        //socket.on('close', () => console.log('disconnected'));

        socket.setTimeout(function () {
            socket.close();
        }, 1500);
    });

    check(res, { 'status is 101': (r) => r && r.status === 101 });
}
