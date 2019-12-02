'use strict';
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
const vscode = require("vscode");
const path = require("path");
const fs = require("fs");
const manifold_1 = require("./manifold");
function activate(context) {
    if (vscode.workspace.workspaceFolders === undefined) {
        return;
    }
    const myCommandId = 'tractor.toggleRun';
    context.subscriptions.push(vscode.commands.registerCommand(myCommandId, () => {
        vscode.window.showInformationMessage(`Ok`);
    }));
    // create a new status bar item that we can now manage
    let myStatusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 100);
    myStatusBarItem.command = myCommandId;
    context.subscriptions.push(myStatusBarItem);
    myStatusBarItem.text = `$(play) Running workspace`;
    myStatusBarItem.show();
    vscode.commands.registerCommand('graphview.open', () => {
        this.inspectorPanel = vscode.window.createWebviewPanel('graphview', "Graph View", vscode.ViewColumn.Two, {
            localResourceRoots: [vscode.Uri.file(path.join(context.extensionPath, 'resources'))],
            enableScripts: true
        });
        fs.readFile(path.join(context.extensionPath, 'resources', 'graphview', 'index.html'), 'utf8', (err, contents) => {
            this.inspectorPanel.webview.html = contents.replace(new RegExp("vscode-resource://", "g"), "vscode-resource://" + path.join(context.extensionPath, 'resources'));
        });
    });
    var tree = new manifold_1.TreeExplorer(context);
    vscode.commands.registerCommand('treeExplorer.addNode', () => {
        vscode.window.showInputBox({ placeHolder: 'Enter a node name' })
            .then(value => {
            if (value !== null && value !== undefined) {
                tree.addNode(value);
            }
        });
    });
    vscode.commands.registerCommand('treeExplorer.addChild', (node) => {
        vscode.window.showInputBox({ placeHolder: 'Enter a node name' })
            .then((value) => __awaiter(this, void 0, void 0, function* () {
            if (value !== null && value !== undefined) {
                tree.addNode(value, node.id);
                // const result = await vscode.window.showQuickPick(['eins', 'zwei', 'drei', 'drwefwefei', 'awerg3drei', 'dweeeefrei', 'dreiffw3433'], {
                // 	placeHolder: 'foobar'
                // });
            }
        }));
    });
    vscode.commands.registerCommand('treeExplorer.renameNode', (node) => {
        vscode.window.showInputBox({ placeHolder: 'Enter a node name' })
            .then(value => {
            if (value !== null && value !== undefined) {
                tree.updateNode(node.id, value);
            }
        });
    });
    vscode.commands.registerCommand('treeExplorer.deleteNode', (node) => {
        tree.deleteNode(node.id);
    });
    vscode.commands.registerCommand('treeExplorer.moveNodeUp', (node) => {
        if (node.index === 0) {
            return;
        }
        console.log(node.id, node.index);
        tree.moveNode(node.id, node.index - 1);
    });
    vscode.commands.registerCommand('treeExplorer.moveNodeDown', (node) => {
        tree.moveNode(node.id, node.index + 1);
    });
    vscode.tasks.executeTask(new vscode.Task({
        type: 'agent'
    }, "server", "tractor", new vscode.ShellExecution(`${path.join(context.extensionPath, '../dev/bin/tractor')} agent dev`)));
    // setTimeout(() => {
    // 	let repl = vscode.window.createTerminal("repl", path.join(context.extensionPath, '../repl.js'));
    // 	repl.show();
    // }, 3000);
}
exports.activate = activate;
//# sourceMappingURL=extension.js.map