/********************************************************************************
 * Copyright (C) 2017-2018 TypeFox and others.
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
import {
    TreeWidget,
    TreeNode,
    NodeProps,
    SelectableTreeNode,
    TreeProps,
    ContextMenuRenderer,
    TreeModel,
    ExpandableTreeNode
} from '@theia/core/lib/browser';
import { TractorTreeModel } from './tractor-tree-model';
import { Message } from '@phosphor/messaging';
import { Emitter } from '@theia/core';
import { CompositeTreeNode } from '@theia/core/lib/browser';
import * as React from 'react';

/**
 * Representation of an object node.
 */
export interface ObjectNode extends CompositeTreeNode, SelectableTreeNode, ExpandableTreeNode {
    
    iconClass: string;

    absPath: string;

    relatedComponents: string[];
}

/**
 * Collection of outline symbol information node functions.
 */
export namespace ObjectNode {
    /**
     * Determine if the given tree node is an `OutlineSymbolInformationNode`.
     * - The tree node is an `OutlineSymbolInformationNode` if:
     *  - The node exists.
     *  - The node is selectable.
     *  - The node contains a defined `iconClass` property.
     * @param node the tree node.
     *
     * @returns `true` if the given node is an `OutlineSymbolInformationNode`.
     */
    export function is(node: TreeNode): node is ObjectNode {
        return !!node && SelectableTreeNode.is(node) && 'absPath' in node;
    }
}

export type TractorTreeWidgetFactory = () => TractorTreeWidget;
export const TractorTreeWidgetFactory = Symbol('TractorTreeWidgetFactory');

@injectable()
export class TractorTreeWidget extends TreeWidget {
    static readonly ID = 'tractor:tree';
    static readonly LABEL = 'Objects';

    readonly onDidChangeOpenStateEmitter = new Emitter<boolean>();

    constructor(
        @inject(TreeProps) protected readonly treeProps: TreeProps,
        @inject(TractorTreeModel) model: TractorTreeModel,
        @inject(ContextMenuRenderer) protected readonly contextMenuRenderer: ContextMenuRenderer
    ) {
        super(treeProps, model, contextMenuRenderer);

        this.id = 'tractor-view';
        this.title.label = 'Objects';
        this.title.caption = 'Objects';
        this.title.closable = true;
        this.title.iconClass = 'fa fas fa-project-diagram';
        this.addClass('theia-outline-view');

    }

    public setData(data: any) {
        this.model.root = {
            id: 'tractor-root',
            name: 'Tractor Root',
            visible: false,
            children: this.nodesFromData(data, undefined),
            parent: undefined
        } as CompositeTreeNode;
    }

    public rootObjects(): ObjectNode[] {
        return (this.model.root as CompositeTreeNode).children as ObjectNode[];
    }

    nodesFromData(data: any, parent: ObjectNode|undefined): TreeNode[] {
        let paths = [];
        if (parent) {
            paths = data.hierarchy.filter((p) => {
                let basePath = parent.absPath+"/";
                if (p.startsWith(basePath)) {
                    return (p.replace(basePath, "").lastIndexOf("/") === -1);  
                } else {
                    return false;
                }
            });
        } else {
            paths = data.hierarchy.filter((p) => {
                return (p.lastIndexOf("/") === 0);
            }); 
        }
        return paths.map((p) => {
            return {id: data.nodePaths[p], path: p};
        }).map((obj) => {
            let n = data.nodes[obj.id];
            let related = []
            n.components.forEach((com) => {
                if (com.related) {
                    related.push(...com.related);
                }
            });
            let objNode = {
                id: obj.id,
                name: n.name,
                iconClass: "",
                visible: true,
                expanded: true,
                absPath: obj.path,
                selected: false,
                relatedComponents: [...new Set(related)]
            } as ObjectNode;
            objNode.children = this.nodesFromData(data, objNode);
            let treeNode = this.model.getNode(obj.id);
            if (treeNode && ObjectNode.is(treeNode)) {
                objNode.expanded = treeNode.expanded;
                objNode.selected = treeNode.selected;
            }
            return objNode;
        });
    }


    protected onAfterHide(msg: Message): void {
        super.onAfterHide(msg);
        this.onDidChangeOpenStateEmitter.fire(false);
    }

    protected onAfterShow(msg: Message): void {
        super.onAfterShow(msg);
        this.onDidChangeOpenStateEmitter.fire(true);
    }

    renderIcon(node: TreeNode, props: NodeProps): React.ReactNode {
        if (ObjectNode.is(node)) {
            return <div className={'symbol-icon symbol-icon-center ' + node.iconClass}></div>;
        }
        return undefined;
    }

    protected createNodeAttributes(node: TreeNode, props: NodeProps): React.Attributes & React.HTMLAttributes<HTMLElement> {
        const elementAttrs = super.createNodeAttributes(node, props);
        return {
            ...elementAttrs,
            title: this.getNodeTooltip(node)
        };
    }

    /**
     * Get the tooltip for the given tree node.
     * - The tooltip is discovered when hovering over a tree node.
     * - If available, the tooltip is the concatenation of the node name, and it's type.
     * @param node the tree node.
     *
     * @returns the tooltip for the tree node if available, else `undefined`.
     */
    protected getNodeTooltip(node: TreeNode): string | undefined {
        if (ObjectNode.is(node)) {
            return node.name + ` (${node.iconClass})`;
        }
        return undefined;
    }

    protected isExpandable(node: TreeNode): node is ExpandableTreeNode {
        return ObjectNode.is(node) && node.children.length > 0;
    }

    protected renderTree(model: TreeModel): React.ReactNode {
        if (CompositeTreeNode.is(this.model.root) && !this.model.root.children.length) {
            return <div className='theia-widget-noInfo no-outline'>No outline information available.</div>;
        }
        return super.renderTree(model);
    }

}