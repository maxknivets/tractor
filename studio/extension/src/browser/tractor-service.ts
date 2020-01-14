/********************************************************************************
 * Copyright (C) 2017 TypeFox and others.
 *
 * This program and the accompanying materials are made available under the
 * terms of the Eclipse Public License v. 2.0 which is available at
 * http://www.eclipse.org/legal/epl-2.0.
 *
 * This Source Code may also be made available under the following Secondary
 * Licenses when the conditions for such availability set forth in the Eclipse
 * Public License v. 2.0 are satisfied: GNU General Public License, version 2
 * with the GNU Classpath Exception which is available at
 * https://www.gnu.org/software/classpath/license.html.
 *
 * SPDX-License-Identifier: EPL-2.0 OR GPL-2.0 WITH Classpath-exception-2.0
 ********************************************************************************/

import { injectable, inject } from 'inversify';
import URI from '@theia/core/lib/common/uri';
import { Event, Emitter, DisposableCollection } from '@theia/core';
import { WidgetFactory } from '@theia/core/lib/browser';
import { TractorTreeWidget, ObjectNode, TractorTreeWidgetFactory } from './tractor-tree-widget';
import { Widget } from '@phosphor/widgets';
import { WorkspaceService } from '@theia/workspace/lib/browser';
import { ILogger, MessageService } from '@theia/core';

import * as qmux from './qmux';
import * as qrpc from 'qrpc';

const RetryInterval = 500;

function scheduleRetry(fn: any) {
	setTimeout(fn, RetryInterval);
}


@injectable()
export class TractorService implements WidgetFactory {

    id = TractorTreeWidget.ID;

    @inject(WorkspaceService)
    protected readonly workspace: WorkspaceService;

    @inject(MessageService)
    protected readonly messages: MessageService;

    @inject(ILogger)
    protected readonly logger: ILogger;

    protected client: qrpc.Client;
    protected api: qrpc.API;

    protected widget?: TractorTreeWidget;
    protected readonly onDidChangeEmitter = new Emitter<ObjectNode[]>();
    protected readonly onDidChangeOpenStateEmitter = new Emitter<boolean>();
    protected readonly onDidSelectEmitter = new Emitter<ObjectNode>();
    protected readonly onDidOpenEmitter = new Emitter<ObjectNode>();

    constructor(@inject(TractorTreeWidgetFactory) protected factory: TractorTreeWidgetFactory) { }

    get onDidSelect(): Event<ObjectNode> {
        return this.onDidSelectEmitter.event;
    }

    get onDidOpen(): Event<ObjectNode> {
        return this.onDidOpenEmitter.event;
    }

    get onDidChange(): Event<ObjectNode[]> {
        return this.onDidChangeEmitter.event;
    }

    get onDidChangeOpenState(): Event<boolean> {
        return this.onDidChangeOpenStateEmitter.event;
    }

    get open(): boolean {
        return this.widget !== undefined && this.widget.isVisible;
    }

    async connectAgent() {
        try {
			var conn = await qmux.DialWebsocket("ws://localhost:3001/");
		} catch (e) {
			scheduleRetry(() => this.connectAgent());
			return;
		}
        var session = new qmux.Session(conn);
        var client = new qrpc.Client(session);
        var path = new URI(this.workspace.workspace.uri).path.toString()
        var resp = await client.call("connect", path);
        this.connectWorkspace(resp.reply);
    }

    async connectWorkspace(socketPath: string) {
		try {
			var conn = await qmux.DialWebsocket("ws://localhost:3001"+socketPath);
		} catch (e) {
			scheduleRetry(() => this.connectWorkspace(socketPath));
			return;
		}
        var session = new qmux.Session(conn);
        this.api = new qrpc.API();
		this.client = new qrpc.Client(session, this.api);
		this.api.handle("shutdown", {
			"serveRPC": async (r, c) => {
                this.messages.info("DEBUG: reload/shutdown received...");
				setTimeout(() => this.connectWorkspace(socketPath), 4000); // TODO: something better
			}
        });
        this.api.handle("state", {
			"serveRPC": async (r, c) => {
                var data = await c.decode();
                //this.messages.info("DEBUG: got data");
                if (this.widget) {
                    this.widget.setData(data);
                    this.onDidChangeEmitter.fire(this.widget.rootObjects());
                }
                r.return();
			}
        });
        this.client.serveAPI();
        if (this.widget) {
            this.widget.model.onSelectionChanged(event => {
                const node = this.widget.model.selectedNodes[0];
                this.client.call("selectNode", node.id);
            });
        }
		await this.client.call("subscribe");
    }
    
    renameNode(id: string, name: string) {
		this.client.call("updateNode", {
			"ID": id,
			"Name": name
		});
	}

    addNode(name: string, parentId?: string) {
		this.client.call("appendNode", {"ID": parentId||"", "Name": name});
	}

	deleteNode(id: string) {
		this.client.call("deleteNode", id);
    }
    

    createWidget(): Promise<Widget> {
        this.widget = this.factory();
        const disposables = new DisposableCollection();
        disposables.push(this.widget.onDidChangeOpenStateEmitter.event(open => this.onDidChangeOpenStateEmitter.fire(open)));
        disposables.push(this.widget.model.onOpenNode(node => this.onDidOpenEmitter.fire(node as ObjectNode)));
        disposables.push(this.widget.model.onSelectionChanged(selection => this.onDidSelectEmitter.fire(selection[0] as ObjectNode)));
        this.widget.disposed.connect(() => {
            this.widget = undefined;
            disposables.dispose();
        });
        return Promise.resolve(this.widget);
    }
}