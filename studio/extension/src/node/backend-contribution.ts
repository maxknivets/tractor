import * as fs from 'fs';
import * as ws from 'ws';
import * as os from 'os';
import * as http from 'http';
import * as https from 'https';
import { injectable, inject } from 'inversify';
import { BackendApplicationContribution } from '@theia/core/lib/node/backend-application';

import * as qmux from 'qmux';
import * as qrpc from 'qrpc';

@injectable()
export class BackendContribution implements BackendApplicationContribution {

    async onStart(server: http.Server | https.Server): Promise<void> {
        const agentSocketPath = `${os.homedir()}/.tractor/agent.sock`;
        // TODO: separate, static port might not work later on.
        //       unfortunately, no way to hook into theia websocket
        //       without using their crazy system.
        const wss = new ws.Server({ port: 3001 });
        wss.on('connection', async function connection(ws, req) {
            var path = req.url;
            if (path === "/") {
                path = agentSocketPath;
            }
            var conn = await qmux.DialUnix(path);
            conn.socket.on("data", (data: any) => ws.send(data));
            conn.socket.on("close", () => ws.close());
            ws.on('message', (msg) => conn.socket.write(msg));
            ws.on("close", () => conn.close());
        });

    }
}
