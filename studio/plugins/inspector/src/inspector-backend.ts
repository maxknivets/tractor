import * as path from 'path';
import * as theia from '@theia/plugin';

export function start(context: theia.PluginContext) {
    context.subscriptions.push(
        theia.commands.registerCommand({ id: 'inspector.show', label: "Inspector: Show" }, () => {
            InspectorPanel.createOrShow(context.extensionPath);
        })
    );

    if (theia.window.registerWebviewPanelSerializer) {
        // Make sure we register a serializer in activation event
        theia.window.registerWebviewPanelSerializer(InspectorPanel.viewType, {
            async deserializeWebviewPanel(webviewPanel: theia.WebviewPanel, state: any) {
                InspectorPanel.revive(webviewPanel, context.extensionPath);
            }
        });
    }
}

export function stop() {
}


/**
 * Manages inspector webview panels
 */
class InspectorPanel {
	/**
	 * Track the currently panel. Only allow a single panel to exist at a time.
	 */
    public static currentPanel: InspectorPanel | undefined;

    public static readonly viewType = 'inspector';

    private readonly _panel: theia.WebviewPanel;
    private readonly _extensionPath: string;
    private _disposables: theia.Disposable[] = [];

    public static createOrShow(extensionPath: string) {
        const column = theia.window.activeTextEditor
            ? theia.window.activeTextEditor.viewColumn
            : undefined;

        // If we already have a panel, show it.
        if (InspectorPanel.currentPanel) {
            InspectorPanel.currentPanel._panel.reveal(column);
            return;
        }

        // Otherwise, create a new panel.
        const panel = theia.window.createWebviewPanel(
            InspectorPanel.viewType,
            'Inspector',
            column || theia.ViewColumn.One,
            {
                // Enable javascript in the webview
                enableScripts: true,

                // And restrict the webview to only loading content from our extension's `media` directory.
                localResourceRoots: [theia.Uri.file(path.join(extensionPath, 'media'))]
            }
        );

        InspectorPanel.currentPanel = new InspectorPanel(panel, extensionPath);
    }

    public static revive(panel: theia.WebviewPanel, extensionPath: string) {
        InspectorPanel.currentPanel = new InspectorPanel(panel, extensionPath);
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
                        if (message.path !== undefined) {
                            theia.window.showTextDocument(theia.Uri.file(message.path));
                            return;
                        }
                        return;

                }
            },
            null,
            this._disposables
        );

    }


    public dispose() {
        InspectorPanel.currentPanel = undefined;

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
        let rootPath = "";
        if (theia.workspace.workspaceFolders) {
            rootPath = theia.workspace.workspaceFolders[0].uri.path;
        }
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
            <link rel="stylesheet" SameSite=None href="https://unpkg.com/rbx@2.2.0/index.css" />
            <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/bulma/0.7.5/css/bulma.min.css" />
            <link rel="stylesheet" href="${webviewUri('fontawesome/css/all.min.css')}" />  
            <link rel="stylesheet" href="${webviewUri('inspector/inspector.css')}" />  
            <script nonce="${nonce}" src="https://unpkg.com/babel-standalone@6.15.0/babel.min.js"></script>
            <script nonce="${nonce}" src="https://unpkg.com/react@16/umd/react.development.js"></script>
            <script nonce="${nonce}" src="https://unpkg.com/react-dom@16/umd/react-dom.development.js"></script>
            <script nonce="${nonce}" src="https://unpkg.com/prop-types@15.6/prop-types.min.js"></script>
            <script nonce="${nonce}" src="https://unpkg.com/classnames@2.2.6/index.js"></script>
            <script nonce="${nonce}" src="https://unpkg.com/rbx@2.2.0/rbx.umd.js"></script>
            <script nonce="${nonce}" src="${webviewUri('inspector/inspector.js')}" type="text/babel"></script>
            <script nonce="${nonce}" src="${webviewUri('qtalk/qmux.js')}"></script>
            <script nonce="${nonce}" src="${webviewUri('qtalk/qrpc.js')}"></script>
          </head>
          <body>
            <div id="app"></div>
            <script type="text/babel">
              window.workspacePath = "${rootPath}";
              window.functionIcon = "${webviewUri('inspector/function-icon.png')}";
              window.rpc = undefined;
              window.theia = acquireTheiaApi();
        
              ReactDOM.render(<InspectorContainer />, document.getElementById('app'));
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
