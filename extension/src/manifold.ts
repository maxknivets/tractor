import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import * as os from 'os';
import * as qmux from 'qmux';
import * as qrpc from 'qrpc';

const RetryInterval = 500;

export class TreeExplorer {

	explorer: vscode.TreeView<Node>;

	inspectorPanel: vscode.WebviewPanel;
	client: any;
	api: any;
    remoteState: any;

    selectedNodeId: any;

	constructor(context: vscode.ExtensionContext, workspacePath: string) {
		
		const treeDataProvider = new NodeProvider(this);
		this.explorer = vscode.window.createTreeView('treeExplorer', { treeDataProvider });
		vscode.commands.registerCommand('treeExplorer.inspectNode', (nodeId) => this.inspectNode(nodeId, context));
		
		this.api = new qrpc.API();
		this.api.handle("state", treeDataProvider);

		this.connectAgent(workspacePath);
	}

	async connectAgent(workspacePath: string) {
		try {
			var conn = await qmux.DialUnix(`${os.homedir()}/.tractor/agent.sock`);
		} catch (e) {
			setTimeout(() => {
				this.connectAgent(workspacePath);
			}, RetryInterval);
			return;
		}
		conn.socket.onclose = () => {
			conn.close();
			setTimeout(() => {
				this.connectAgent(workspacePath);
			}, RetryInterval);
		};
		let agent = new qrpc.Client(new qmux.Session(conn));
		let resp = await agent.call("connect", workspacePath);
		this.connect(resp.reply);
	}

	async connect(socketPath: string) {
		console.log("connecting...");
		try {
			var conn = await qmux.DialUnix(socketPath);
		} catch (e) {
			setTimeout(() => {
				this.connect(socketPath);
			}, RetryInterval);
			return;
		}
		conn.socket.onclose = () => {
			conn.close();
			setTimeout(() => {
				this.connect(socketPath);
			}, RetryInterval);
		};
		var session = new qmux.Session(conn);
		this.client = new qrpc.Client(session, this.api);
		this.api.handle("shutdown", {
			"serveRPC": async (r, c) => {
				console.log("reload/shutdown received...");
				//this.client.close();
				setTimeout(() => {
					this.connect(socketPath);
				}, 4000); // TODO: something better
			}
		});
		this.client.serveAPI();
		//window.rpc = client;
		await this.client.call("subscribe");
	}

	addNode(name: string, parentId?: string) {
		this.client.call("appendNode", {"ID": parentId||"", "Name": name});
	}

	updateNode(id: string, name: string, active?: boolean) {
		let params = {
			"ID": id,
			"Name": name
		};
		if (active !== undefined) {
			params["Active"] = active;
		}
		this.client.call("updateNode", params);
	}

	deleteNode(id: string) {
		this.client.call("deleteNode", id);
	}

	moveNode(id: string, index: number) {
		this.client.call("moveNode", {"ID": id, "Index": index});
	}

	incr() {
		this.client.call("incr");
		vscode.window.showInformationMessage(`incremented`);
	}

	inspectNode(nodeId: any, context: vscode.ExtensionContext): void {
        this.selectedNodeId = nodeId;
        const sendState = () => {
            this.inspectorPanel.webview.postMessage({"event": "state", "state": this.remoteState});
            this.inspectorPanel.webview.postMessage({"event": "select", "nodeId": this.selectedNodeId});
        };
		if (this.inspectorPanel === undefined) {
			// TODO: make another if this one is closed!
			this.inspectorPanel = vscode.window.createWebviewPanel(
				'inspector',
				"Inspector",
				vscode.ViewColumn.One,
				{
					localResourceRoots: [vscode.Uri.file(path.join(context.extensionPath, 'resources'))],
					enableScripts: true
				}
			);
			fs.readFile(path.join(context.extensionPath, 'resources', 'inspector', 'inspector.html'), 'utf8', (err, contents) => {
				this.inspectorPanel.webview.html = contents.replace(new RegExp("vscode-resource://", "g"), "vscode-resource://"+path.join(context.extensionPath, 'resources'));
            });
            this.inspectorPanel.webview.onDidReceiveMessage(
                message => {
                  switch (message.event) {
                    case 'ready':
                      	sendState();
					  	return;
					case 'rpc':
						this.client.call(message.method, message.params);
						return;
					case 'edit':
						if (message.Filepath !== undefined) {
							vscode.window.showTextDocument(vscode.Uri.file(message.Filepath), {
								viewColumn: vscode.ViewColumn.Two
							});
							return;
						}
						if (message.params.Component === "Delegate") {
							vscode.window.showTextDocument(vscode.Uri.file(path.join(vscode.workspace.workspaceFolders[0].uri.path, 'delegates', message.params.ID, 'delegate.go')), {
								viewColumn: vscode.ViewColumn.Two
							});
						} else {
							vscode.window.showTextDocument(vscode.Uri.file(this.remoteState.componentPaths[message.params.Component]), {
								viewColumn: vscode.ViewColumn.Two
							});
						}
						return;
                  }
                },
                undefined,
                context.subscriptions
              );
		}
        sendState();
	}
}


export class NodeProvider implements vscode.TreeDataProvider<Node> {

	private _onDidChangeTreeData: vscode.EventEmitter<Node | undefined> = new vscode.EventEmitter<Node | undefined>();
	readonly onDidChangeTreeData: vscode.Event<Node | undefined> = this._onDidChangeTreeData.event;

    private explorer: TreeExplorer;

	constructor(explorer: TreeExplorer) {
        this.explorer = explorer;
    }
    
    async serveRPC(r, c) {
        var msg = await c.decode();
        if (this.explorer.inspectorPanel !== undefined) {
            this.explorer.inspectorPanel.webview.postMessage({"event": "state", "state": msg});
        }
        this.explorer.remoteState = msg;
        this.refresh();
        // output.appendLine(JSON.stringify(msg));
        r.return();
    }

	refresh(): void {
		this._onDidChangeTreeData.fire();
	}

	getTreeItem(element: Node): vscode.TreeItem {
		return element;
	}

	getChildren(element?: Node): Thenable<Node[]> {
        if (this.explorer.remoteState === undefined) {
            return Promise.resolve([]);
        }
		if (element) {
            let n = this.explorer.remoteState.nodes[element.id];
            let childrenPaths = this.explorer.remoteState.hierarchy.filter((p) => {
                let basePath = element.abspath+"/";
                if (p.startsWith(basePath)) {
                    return (p.replace(basePath, "").lastIndexOf("/") === -1);
                    
                } else {
                    return false;
                }
            });
			return Promise.resolve(childrenPaths.map((p) => {
                return {id: this.explorer.remoteState.nodePaths[p], path: p};
            }).map((obj) => {
                let n = this.explorer.remoteState.nodes[obj.id];
                let collapse = vscode.TreeItemCollapsibleState.None;
                if (this.explorer.remoteState.hierarchy.filter((p) => p.startsWith(obj.path+"/")).length > 0) {
                    collapse = vscode.TreeItemCollapsibleState.Collapsed;
                }
                return new Node(n.name, obj.path, obj.id, n.index, collapse, { command: 'treeExplorer.inspectNode', title: "Inspect", arguments: [obj.id], });
            }));
		} else {
			let rootPaths = this.explorer.remoteState.hierarchy.filter((p) => {
                return (p.lastIndexOf("/") === 0);
            });
            return Promise.resolve(rootPaths.map((p) => {
                return {id: this.explorer.remoteState.nodePaths[p], path: p};
            }).map((obj) => {
                let n = this.explorer.remoteState.nodes[obj.id];
                let collapse = vscode.TreeItemCollapsibleState.None;
                if (this.explorer.remoteState.hierarchy.filter((p) => p.startsWith(obj.path+"/")).length > 0) {
                    collapse = vscode.TreeItemCollapsibleState.Collapsed;
                }
                return new Node(n.name, obj.path, obj.id, n.index, collapse, { command: 'treeExplorer.inspectNode', title: "Inspect", arguments: [obj.id], });
            }));
		}

	}

}

export class Node extends vscode.TreeItem {

	constructor(
        public readonly label: string,
        public readonly abspath: string,
		public readonly id: string,
		public readonly index: number,
		public readonly collapsibleState: vscode.TreeItemCollapsibleState,
		public readonly command?: vscode.Command
	) {
		super(label, collapsibleState);
	}

	get tooltip(): string {
		return `${this.label} (${this.id})`;
	}

	// get description(): string {
	// 	return "$(alert)";
	// }

	iconPath = {
		light: path.join(__filename, '..', '..', 'resources', 'icons', 'light', 'document.svg'),
		dark: path.join(__filename, '..', '..', 'resources', 'icons', 'dark', 'document.svg')
	};

	contextValue = 'node';

}


