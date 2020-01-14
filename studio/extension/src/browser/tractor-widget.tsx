import * as React from 'react';
import { injectable, postConstruct, inject } from 'inversify';
import { AlertMessage } from '@theia/core/lib/browser/widgets/alert-message';
import { ReactWidget } from '@theia/core/lib/browser/widgets/react-widget';
import { MenuPath, MenuAction } from '@theia/core/lib/common';
import { MenuModelRegistry, CompositeMenuNode } from '@theia/core/lib/common/menu';
import { MessageService, CommandRegistry } from '@theia/core';
import { ContextMenuRenderer } from '@theia/core/lib/browser/context-menu-renderer';

export const CONTEXT_MENU: MenuPath = ['TRACTOR_CONTEXT'];

@injectable()
export class TractorWidget extends ReactWidget {

    static readonly ID = 'tractor:widget';
    static readonly LABEL = 'Tractor';

    @inject(MessageService)
    protected readonly messageService!: MessageService;

    @inject(ContextMenuRenderer)
    protected readonly contextMenuRenderer: ContextMenuRenderer;

    @inject(MenuModelRegistry)
    protected readonly menuRegistry: MenuModelRegistry;

    @inject(CommandRegistry)
    protected readonly commandRegistry: CommandRegistry;

    @postConstruct()
    protected async init(): Promise < void> {
        this.id = TractorWidget.ID;
        this.title.label = TractorWidget.LABEL;
        this.title.caption = TractorWidget.LABEL;
        this.title.closable = true;
        this.title.iconClass = 'fa fa-home'; // example widget icon.
        this.update(); 

        this.displayMessage = this.displayMessage.bind(this);
        let cmd = {
            id: 'tractor.foo',
            label: 'Tractor: Foo...'
        };
        this.commandRegistry.registerCommand(cmd, {
            execute: () => this.foo(),
            isEnabled: () => true
        });

        this.menuRegistry.registerMenuAction([...CONTEXT_MENU, '0_main'], {
            commandId: cmd.id
        });
        this.menuRegistry.registerSubmenu([...CONTEXT_MENU, '1_section'], "Submenu");
    }

    protected foo(): void {
        this.messageService.info('Foo');
    }

    protected render(): React.ReactNode {
        const header = `This is a sample widget which simply calls the messageService
        in order to display an info message to end users.`;
        return <div id='widget-container'>
            <AlertMessage type='INFO' header={header} />
            <button className='theia-button secondary' title='Display Message' onClick={this.displayMessage}>Display Message</button>
        </div>
    }

    protected displayMessage(event: React.MouseEvent<HTMLButtonElement, MouseEvent>): void {
        const { x, y } = event.nativeEvent;
        this.menuRegistry.unregisterMenuAction('1_section', [...CONTEXT_MENU]);
        this.menuRegistry.registerSubmenu([...CONTEXT_MENU, '1_section'], `Submenu-${Date.now()}`);
        this.menuRegistry.registerMenuAction([...CONTEXT_MENU, '1_section'], {
            commandId: 'tractor.foo'
        });
        this.contextMenuRenderer.render({
            menuPath: CONTEXT_MENU,
            anchor: { x, y }
        })
        this.messageService.info('Congratulations!! Tractor Widget Successfully Created!');
    }

}
