import { injectable, inject } from 'inversify';
import { MenuModelRegistry } from '@theia/core';
import { FrontendApplicationContribution, FrontendApplication } from '@theia/core/lib/browser/frontend-application';
import { TractorTreeWidget } from './tractor-tree-widget';
import { TractorService } from './tractor-service';
import { AbstractViewContribution } from '@theia/core/lib/browser';
import { TabBarToolbarContribution, TabBarToolbarRegistry } from '@theia/core/lib/browser/shell/tab-bar-toolbar';
import { Command, CommandRegistry } from '@theia/core/lib/common/command';
import { MenuPath } from '@theia/core/lib/common';
import { QuickPickService } from '@theia/core/lib/browser/quick-open';
import { CompositeTreeNode } from '@theia/core/lib/browser/tree';
import { Widget } from '@theia/core/lib/browser/widgets';
import { WidgetManager } from '@theia/core/lib/browser';
import { MessageService, ILogger } from '@theia/core';
import { SingleTextInputDialog } from '@theia/core/lib/browser/dialogs';

export namespace TractorCommands {
    export const TOGGLE: Command = {
        id: 'tractor:command',
        label: 'Toggle'
    };
    export const DEBUG: Command = {
        id: 'tractor:debug',
        label: 'Debug Command'
    };
    export const DELETE_NODE: Command = {
        id: 'tractor:node-delete',
        label: 'Delete'
    };
    export const ADD_NODE: Command = {
        id: 'tractor:node-add',
        label: 'Empty Object'
    };
    export const RENAME_NODE: Command = {
        id: 'tractor:node-rename',
        label: 'Rename'
    };
    export const ADD_COMPONENT: Command = {
        id: 'tractor:component-add',
        label: 'Add Component...'
    };
}

export const TRACTOR_CONTEXT_MENU: MenuPath = ['tractor-context-menu'];

/**
 * Navigator context menu default groups should be aligned
 * with VS Code default groups: https://code.visualstudio.com/api/references/contribution-points#contributes.menus
 */
export namespace TractorContextMenu {
    export const NAVIGATION = [...TRACTOR_CONTEXT_MENU, 'navigation'];

    export const NEW = [...TRACTOR_CONTEXT_MENU, '0_new'];

    export const WORKSPACE = [...TRACTOR_CONTEXT_MENU, '2_workspace'];

    export const COMPARE = [...TRACTOR_CONTEXT_MENU, '3_compare'];

    export const SEARCH = [...TRACTOR_CONTEXT_MENU, '4_search'];
    export const CLIPBOARD = [...TRACTOR_CONTEXT_MENU, '5_cutcopypaste'];

    export const MODIFICATION = [...TRACTOR_CONTEXT_MENU, '7_modification'];

    export const OPEN_WITH = [...NAVIGATION, 'open_with'];
}

@injectable()
export class TractorContribution extends AbstractViewContribution<TractorTreeWidget> implements FrontendApplicationContribution, TabBarToolbarContribution {

    @inject(TractorService)
    protected readonly tractor: TractorService;
    
    @inject(MessageService)
    protected readonly messages: MessageService;

    @inject(WidgetManager)
    protected readonly widgets: WidgetManager;

    @inject(QuickPickService)
    protected readonly quickpick: QuickPickService;

    @inject(ILogger)
    protected readonly logger: ILogger;

    /**
     * `AbstractViewContribution` handles the creation and registering
     *  of the widget including commands, menus, and keybindings.
     * 
     * We can pass `defaultWidgetOptions` which define widget properties such as 
     * its location `area` (`main`, `left`, `right`, `bottom`), `mode`, and `ref`.
     * 
     */
    constructor() {
        super({
            widgetId: TractorTreeWidget.ID,
            widgetName: TractorTreeWidget.LABEL,
            defaultWidgetOptions: { area: 'left' },
            toggleCommandId: TractorCommands.TOGGLE.id
        });
    }

    onStart(app: FrontendApplication): void {
        this.tractor.connectAgent();
        this.widgets.onDidCreateWidget((e) => {
            if (e.widget.constructor.name === "WebviewWidget") {
                e.widget.title.iconClass = "fa fas fa-clipboard-list";
            }
        });
    }

    async initializeLayout(app: FrontendApplication): Promise<void> {
        await this.openView();
    }

    /**
     * Example command registration to open the widget from the menu, and quick-open.
     * For a simpler use case, it is possible to simply call:
     ```ts
        super.registerCommands(commands)
     ```
     *
     * For more flexibility, we can pass `OpenViewArguments` which define 
     * options on how to handle opening the widget:
     * 
     ```ts
        toggle?: boolean
        activate?: boolean;
        reveal?: boolean;
     ```
     *
     * @param commands
     */
    registerCommands(commands: CommandRegistry): void {
        super.registerCommands(commands);
        commands.registerCommand(TractorCommands.TOGGLE, {
            execute: () => super.openView({ activate: false, reveal: true })
        });
        commands.registerCommand(TractorCommands.DEBUG, {
            execute: () => super.openView({ activate: false, reveal: true })
        });
        commands.registerCommand(TractorCommands.DELETE_NODE, {
            execute: () => {
                let node = (this.shell.currentWidget as TractorTreeWidget).model.selectedNodes[0];
                if (node) {
                    this.tractor.deleteNode(node.id);
                }
            },
            isEnabled: () => true,
            isVisible: () => true
        });
        commands.registerCommand(TractorCommands.ADD_COMPONENT, {
            execute: () => {
                let node = (this.shell.currentWidget as TractorTreeWidget).model.selectedNodes[0];
                if (node) {
                    this.quickpick.show(this.tractor.components.map((el)=>el.name)).then((selection) => {
                        this.tractor.addComponent(selection, node.id);
                    })
                }
            }
        });
        commands.registerCommand(TractorCommands.ADD_NODE, {
            execute: () => {
                let node = (this.shell.currentWidget as TractorTreeWidget).model.selectedNodes[0];
                if (node) {
                    this.tractor.addNode("Empty Object", node.id);
                }
            }
        });
        commands.registerCommand(TractorCommands.RENAME_NODE, {
            execute: () => {
                let node = (this.shell.currentWidget as TractorTreeWidget).model.selectedNodes[0];
                if (node) {
                    const initialValue = node.name;
                    const titleStr = `Rename ${node.name}`;
                    const dialog = new SingleTextInputDialog({
                        title: titleStr,
                        initialValue,
                        initialSelectionRange: {
                            start: 0,
                            end: initialValue.length
                        },
                        validate: (name, mode) => {
                            if (initialValue === name && mode === 'preview') {
                                return false;
                            }
                            return true; //this.validateFileName(name, parent, false);
                        }
                    });
                    dialog.open().then(name => {
                        if (name) {
                            this.tractor.renameNode(node.id, name);
                        }
                    });
                }
            }
        });
    }

    registerToolbarItems(toolbar: TabBarToolbarRegistry): void {
        // toolbar.registerItem({
        //     id: OutlineViewCommands.COLLAPSE_ALL.id,
        //     command: OutlineViewCommands.COLLAPSE_ALL.id,
        //     tooltip: 'Collapse All',
        //     priority: 0
        // });
    }

    /**
     * Example menu registration to contribute a menu item used to open the widget.
     * Default location when extending the `AbstractViewContribution` is the `View` main-menu item.
     * 
     * We can however define new menu path locations in the following way:
     ```ts
        menus.registerMenuAction(CommonMenus.HELP, {
            commandId: 'id',
            label: 'label'
        });
     ```
     * 
     * @param menus
     */
    registerMenus(menus: MenuModelRegistry): void {
        super.registerMenus(menus);
        // menus.registerMenuAction(TractorContextMenu.NAVIGATION, {
        //     commandId: TractorCommands.TOGGLE.id
        // });
        
        menus.registerSubmenu(TractorContextMenu.NEW, 'New');
        menus.registerMenuAction(TractorContextMenu.NEW, {
            commandId: TractorCommands.ADD_NODE.id
        });
        
        menus.registerMenuAction(TractorContextMenu.WORKSPACE, {
            commandId: TractorCommands.ADD_COMPONENT.id
        });

        menus.registerMenuAction(TractorContextMenu.MODIFICATION, {
            commandId: TractorCommands.DELETE_NODE.id
        });
        
        menus.registerMenuAction(TractorContextMenu.MODIFICATION, {
            commandId: TractorCommands.RENAME_NODE.id
        });
        
    }

    /**
     * Collapse all nodes in the outline view tree.
     */
    protected async collapseAllItems(): Promise<void> {
        const { model } = await this.widget;
        const root = model.root;
        if (CompositeTreeNode.is(root)) {
            model.collapseAll(root);
        }
    }

    /**
     * Determine if the current widget is the `outline-view`.
     */
    protected withWidget<T>(widget: Widget | undefined = this.tryGetWidget(), cb: (widget: TractorTreeWidget) => T): T | false {
        if (widget instanceof TractorTreeWidget && widget.id === TractorTreeWidget.ID) {
            return cb(widget);
        }
        return false;
    }
}
