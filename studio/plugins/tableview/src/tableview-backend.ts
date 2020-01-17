import * as path from 'path';
import * as theia from '@theia/plugin';

export function start(context: theia.PluginContext) {
    context.subscriptions.push(
        theia.commands.registerCommand({ id: 'tableview.show', label: "Tableview: Show" }, () => {
            TableViewPanel.createOrShow(context.extensionPath);
        })
    );

    if (theia.window.registerWebviewPanelSerializer) {
        // Make sure we register a serializer in activation event
        theia.window.registerWebviewPanelSerializer(TableViewPanel.viewType, {
            async deserializeWebviewPanel(webviewPanel: theia.WebviewPanel, state: any) {
                TableViewPanel.revive(webviewPanel, context.extensionPath);
            }
        });
    }
}

export function stop() {
}


/**
 * Manages inspector webview panels
 */
class TableViewPanel {
	/**
	 * Track the currently panel. Only allow a single panel to exist at a time.
	 */
    public static currentPanel: TableViewPanel | undefined;

    public static readonly viewType = 'tableview';

    private readonly _panel: theia.WebviewPanel;
    private readonly _extensionPath: string;
    private _disposables: theia.Disposable[] = [];

    public static createOrShow(extensionPath: string) {
        const column = theia.window.activeTextEditor
            ? theia.window.activeTextEditor.viewColumn
            : undefined;

        // If we already have a panel, show it.
        if (TableViewPanel.currentPanel) {
            TableViewPanel.currentPanel._panel.reveal(column);
            return;
        }

        // Otherwise, create a new panel.
        const panel = theia.window.createWebviewPanel(
            TableViewPanel.viewType,
            'Tableview',
            column || theia.ViewColumn.One,
            {
                // Enable javascript in the webview
                enableScripts: true,

                // And restrict the webview to only loading content from our extension's `media` directory.
                localResourceRoots: [theia.Uri.file(path.join(extensionPath, 'media'))]
            }
        );

        TableViewPanel.currentPanel = new TableViewPanel(panel, extensionPath);
    }

    public static revive(panel: theia.WebviewPanel, extensionPath: string) {
        TableViewPanel.currentPanel = new TableViewPanel(panel, extensionPath);
    }

    private constructor(panel: theia.WebviewPanel, extensionPath: string) {
        this._panel = panel;
        this._extensionPath = extensionPath;

        // Set the webview's initial html content
        this._panel.webview.html = this._getHtmlForWebview(this._panel.webview);


        // Listen for when the panel is disposed
        // This happens when the user closes the panel or when the panel is closed programatically
        this._panel.onDidDispose(() => this.dispose(), null, this._disposables);

        // Update the content based on view changes
        this._panel.onDidChangeViewState(
            (e: any) => {
                if (this._panel.visible) {
                    //this._update();
                }
            },
            null,
            this._disposables
        );

        // Handle messages from the webview
        this._panel.webview.onDidReceiveMessage(
            (message: any) => {
                switch (message.event) {
                    case 'edit':
                        if (message.Filepath !== undefined) {
                            theia.window.showTextDocument(theia.Uri.file(message.Filepath));
                            return;
                        }
                        if (message.params.Component === "Delegate") {
                            let folders = theia.workspace.workspaceFolders;
                            if (folders) {
                                theia.window.showTextDocument(theia.Uri.file(path.join(folders[0].uri.path, 'delegates', message.params.ID, 'delegate.go')));
                            }
                        } else {
                            theia.window.showTextDocument(theia.Uri.file(message.params.Filepath));
                        }
                        return;

                }
            },
            null,
            this._disposables
        );

    }


    public dispose() {
        TableViewPanel.currentPanel = undefined;

        // Clean up our resources
        this._panel.dispose();

        while (this._disposables.length) {
            const x = this._disposables.pop();
            if (x) {
                x.dispose();
            }
        }
    }


    private _getHtmlForWebview(webview: theia.Webview) {
        const mediaPath = path.join(this._extensionPath, "media");
        const webviewUri = (filepath: string) => webview.asWebviewUri(theia.Uri.file(path.join(mediaPath, filepath)));
        const nonce = getNonce();
        // let rootPath = "";
        // if (theia.workspace.workspaceFolders) {
        //     rootPath = theia.workspace.workspaceFolders[0].uri.path;
        // }
        return `<!DOCTYPE html>
        <html lang="en">
          <head>
            <meta charset="UTF-8">
            <!--
            Use a content security policy to only allow loading images from https or from our extension directory,
            and only allow scripts that have a specific nonce.
            -->
            <!--meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src ${webview.cspSource} https:; script-src 'nonce-${nonce}';"-->
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<link rel="stylesheet" href="https://unpkg.com/ag-grid-community/dist/styles/ag-grid.css">
    		<link rel="stylesheet" href="https://unpkg.com/ag-grid-community/dist/styles/ag-theme-balham.css">
            <script nonce="${nonce}" src="https://unpkg.com/ag-grid-community/dist/ag-grid-community.min.noStyle.js"></script>
            <script nonce="${nonce}" src="${webviewUri('qtalk/qmux.js')}"></script>
            <script nonce="${nonce}" src="${webviewUri('qtalk/qrpc.js')}"></script>
          </head>
          <body>
			<div id="myGrid" style="height: 600px;width:500px;" class="ag-theme-balham"></div>

			<script type="text/javascript" charset="utf-8">
				// specify the columns
				var columnDefs = [
				{headerName: "Make", field: "make"},
				{headerName: "Model", field: "model"},
				{headerName: "Price", field: "price"}
				];
			
				// specify the data
				var rowData = [
				{make: "Toyota", model: "Celica", price: 35000},
				{make: "Ford", model: "Mondeo", price: 32000},
				{make: "Porsche", model: "Boxter", price: 72000}
				];
			
				// let the grid know which columns and what data to use
				var gridOptions = {
				columnDefs: columnDefs,
				rowData: rowData
				};
			
			// lookup the container we want the Grid to use
			var eGridDiv = document.querySelector('#myGrid');
			
			// create the grid passing in the div to use together with the columns & data we want to use
			new agGrid.Grid(eGridDiv, gridOptions);
			
			</script>
          </body>
        </html>`
    }
}

function getNonce() {
    let text = '';
    const possible = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    for (let i = 0; i < 32; i++) {
        text += possible.charAt(Math.floor(Math.random() * possible.length));
    }
    return text;
}
