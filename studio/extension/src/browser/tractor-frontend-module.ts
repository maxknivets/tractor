import { ContainerModule, interfaces } from 'inversify';
import { TractorTreeWidget, TractorTreeWidgetFactory } from './tractor-tree-widget';
import { TractorContribution } from './tractor-contribution';
import { TractorService } from './tractor-service';
import { TractorDecoratorService, TractorTreeDecorator } from './tractor-decorator-service';
import { TractorTreeModel } from './tractor-tree-model';
import { TRACTOR_CONTEXT_MENU } from './tractor-contribution';
import { bindContributionProvider } from '@theia/core/lib/common/contribution-provider';
import {
    FrontendApplicationContribution,
    createTreeContainer,
    TreeWidget,
    bindViewContribution,
    TreeProps,
    TreeDecoratorService,
    defaultTreeProps,
    TreeModel,
    TreeModelImpl
} from '@theia/core/lib/browser';
import { WidgetFactory } from '@theia/core/lib/browser/widget-manager';

import '../../src/browser/style/index.css';

export const TRACTOR_TREE_PROPS = <TreeProps>{
    ...defaultTreeProps,
    contextMenuPath: TRACTOR_CONTEXT_MENU,
    search: true
};

export default new ContainerModule(bind => {
    bind(TractorTreeWidgetFactory).toFactory(ctx =>
        () => createTractorTreeWidget(ctx.container)
    );

    bind(TractorService).toSelf().inSingletonScope();
    bind(WidgetFactory).toService(TractorService);

    bindViewContribution(bind, TractorContribution);
    bind(FrontendApplicationContribution).toService(TractorContribution);
    // bind(TractorWidget).toSelf();
    // bind(WidgetFactory).toDynamicValue(ctx => ({
    //     id: TractorWidget.ID,
    //     createWidget: () => ctx.container.get<TractorWidget>(TractorWidget)
    // })).inSingletonScope();
});

/**
 * Create an `TractorTreeWidget`.
 * - The creation of the `TractorTreeWidget` includes:
 *  - The creation of the tree widget itself with it's own customized props.
 *  - The binding of necessary components into the container.
 * @param parent the Inversify container.
 *
 * @returns the `TractorTreeWidget`.
 */
function createTractorTreeWidget(parent: interfaces.Container): TractorTreeWidget {
    const child = createTreeContainer(parent);

    child.rebind(TreeProps).toConstantValue(TRACTOR_TREE_PROPS);

    child.unbind(TreeWidget);
    child.bind(TractorTreeWidget).toSelf();

    child.unbind(TreeModelImpl);
    child.bind(TractorTreeModel).toSelf();
    child.rebind(TreeModel).toService(TractorTreeModel);

    child.bind(TractorDecoratorService).toSelf().inSingletonScope();
    child.rebind(TreeDecoratorService).toDynamicValue(ctx => ctx.container.get(TractorDecoratorService)).inSingletonScope();
    bindContributionProvider(child, TractorTreeDecorator);

    return child.get(TractorTreeWidget);
}