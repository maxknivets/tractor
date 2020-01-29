const { Column, Control, Checkbox, Input, Message, Delete, Box, Heading, Content, List, Dropdown, Breadcrumb, Button, Icon, Level } = rbx;

const RetryInterval = 500;

function scheduleRetry(fn) {
	setTimeout(fn, RetryInterval);
}

class InspectorContainer extends React.Component {
    instance = null;

    constructor(props) {
      super(props);
      this.state = {
        remote: {
          nodes: {},
          nodePaths: {},
          components: []
        },
        lastSelected: undefined
      };

      
      this.connectAgent();
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
        var resp = await client.call("connect", window.workspacePath);
        this.connectWorkspace(resp.reply);
    }

    async connectWorkspace(socketPath) {
		try {
			var conn = await qmux.DialWebsocket("ws://localhost:3001"+socketPath);
		} catch (e) {
			scheduleRetry(() => this.connectWorkspace(socketPath));
			return;
		}
        var session = new qmux.Session(conn);
        this.api = new qrpc.API();
        this.client = new qrpc.Client(session, this.api);
        window.rpc = this.client;
		this.api.handle("shutdown", {
			"serveRPC": async (r, c) => {
                console.log("DEBUG: reload/shutdown received...");
                scheduleRetry(() => this.connectWorkspace(socketPath));
				r.return();
			}
        });
        this.api.handle("state", {
			"serveRPC": async (r, c) => {
                var data = await c.decode();
                this.setState({"remote": data});
                if (data.selectedNode && !this.state.lastSelected) {
                    this.setState({"lastSelected": data.selectedNode});
                }
                r.return();
			}
        });
		this.client.serveAPI();
		await this.client.call("subscribe");
    }

    componentDidMount() {
      InspectorContainer.instance = this;
    }

    render() {
        let node = undefined;
        if (this.state.remote.selectedNode) {
            node = this.state.remote.nodes[this.state.remote.selectedNode];
        } else {
            node = this.state.remote.nodes[this.state.lastSelected];
        }
        console.log(node);
        return <Inspector node={node} components={this.state.remote.components} />;
    }
  }

function remoteAction(action, params) {
    switch (action) {
        case "setValue":
        case "setExpression":
        case "callMethod":
        case "removeComponent":
        case "appendComponent":
        case "reloadComponent":
        case "addDelegate":
            //console.log(action, params);
            window.rpc.call(action, params);
            return;
        case "edit":
            window.theia.postMessage({event: 'edit', path: params.path});
            return;
        default:
            throw "unknown action: "+action;
    }
    
}

function FieldControl(props) {
    const [exprMode, setExprMode] = React.useState(false);
    let onChange = (event) => remoteAction("setValue", { "Path": props.path, "Value": event.target.value });
    let readOnly = (props.expression || "").length > 0;
    function typedControl() {
        if (exprMode) {
            onChange = (event) => remoteAction("setExpression", { "Path": props.path, "Value": event.target.value });
            return <Input type="text" size="small" style={{ height: "22px", color: "white", backgroundColor: "#555", fontFamily: "monospace" }} onChange={onChange} value={props.expression||""} />
        }
        switch (props.type) {
            case "string":
                return <Input type="text" readOnly={readOnly} size="small" onChange={onChange} value={props.value} />
            case "boolean":
                onChange = (event) => remoteAction("setValue", { "Path": props.path, "Value": event.target.checked });
                return <Checkbox onChange={onChange} readOnly={readOnly} checked={props.value} />
            case "number":
                onChange = (event) => remoteAction("setValue", { "Path": props.path, "IntValue": event.target.valueAsNumber });
                return <Input type="number" readOnly={readOnly} style={{ width: "100%" }} size="small" onChange={onChange} value={props.value} />
            default:
                if (props.type.startsWith("reference:")) {
                    var refType = props.type.split(":")[1];
                    let onSet = (path) => remoteAction("setValue", { "Path": props.path, "RefValue": path + "/" + refType });
                    let onUnset = (path) => remoteAction("setValue", { "Path": props.path, "Value": null });
                    return <Reference value={props.value} type={refType} onSet={onSet} onUnset={onUnset} />;
                } else {
                    return "???";
                }
        }
    }
    return (
        <div className="field is-grouped">
            <div className="control is-expanded" style={{marginRight: "0px", height: "22px", }}>
                {typedControl()}
            </div>
            <div className="control">
                <div className="image is-16x16" style={{margin: "3px"}}>
                    <img src={window.functionIcon} onClick={() => setExprMode(!exprMode)} />
                </div>
            </div>
        </div>
    );
}

function FieldRow(props) {
    const children = props.children.slice(0);
    const label = children.shift();
    return (
        <Column.Group style={{display: "flex", marginBottom: "0"}}>
            <Column style={{ fontSize: "smaller"}}>
                {label}
            </Column>
            <Column style={{flexGrow: "2"}}>
                {children}
            </Column>
        </Column.Group>
    );
}

function LabeledField(props) {
    return (
        <FieldRow>
            <span>{props.label}</span>
            {props.children}
        </FieldRow>
    );
}

function KeyedField(props) {
    return (
        <FieldRow>
            <Input type="text" value={props.name} />
            {props.children}
        </FieldRow>
    );
  }

function EmbeddedFields(props) {
    const fields = props.fields || [];
    const [open, setOpen] = React.useState(false);
    return (
        <div>
            <header onClick={() => setOpen(!open)}>
                <Arrow opened={open} />
                <span style={{fontSize: "smaller"}}>{props.name}</span>
            </header>
            {open &&
                <Content style={{marginLeft: "20px", marginTop: "8px"}}>
                    {props.children}
                </Content>
            }
        </div>
    );
}


function ComponentField(props) {
    switch (props.type) {
        case "boolean":
        case "string":
        case "number":
        case "reference":
            return <LabeledField key={props.eventKey} label={props.name}><FieldControl {...props} /></LabeledField>
        case "struct":
        case "map":
        case "array":
            let fields = props.fields || [];
            if (props.type == "array") {
                fields = fields.map((obj, idx) => { obj.name = "Element "+idx; return obj; })
            }
            return (
                <EmbeddedFields {...props}>
                    {(props.type == "array") &&
                        <LabeledField key="count" label="Count">
                            <Input type="number" style={{ width: "100%" }} size="small" value={fields.length} />
                        </LabeledField>
                    }
                    {fields.map((field) => {
                        if (props.type == "map") {
                            return <KeyedField key={field.name} name={field.name}><FieldControl {...field} /></KeyedField>;
                        } else {
                            return <ComponentField key={field.name} {...field} />;
                        }
                    })}
                    {props.children}
                </EmbeddedFields>
            );
        default:
            return <LabeledField key={props.eventKey} label={props.name}><FieldControl {...props} /></LabeledField>
    }
}

function ComponentFields(props) {
    const addKey = (obj, key) => { obj.key = key; return obj};
    return (props.fields || []).map((el, idx) => 
        <ComponentField {...addKey(el, idx)} />
    );
}

function ComponentManageMenu(props) {
    return (
        <Dropdown align="right" style={props.style}>
            <Dropdown.Trigger>
                <Icon size="small"><i className="fas fa-cog"></i></Icon>
            </Dropdown.Trigger>
            <Dropdown.Menu>
                <Dropdown.Content>
                <Dropdown.Item onClick={() => remoteAction("reloadComponent", {ID: props.nodeId, Component: props.component.name})}>Reload</Dropdown.Item>
                    <Dropdown.Item onClick={() => remoteAction("edit", {path: props.component.filepath})}>Edit</Dropdown.Item>
                    <Dropdown.Item onClick={() => remoteAction("removeComponent", {ID: props.nodeId, Component: props.component.name})}>Delete</Dropdown.Item>
                </Dropdown.Content>
            </Dropdown.Menu>
        </Dropdown>
    );
}


function ComponentInspector(props) {
    const [open, setOpen] = React.useState(false);
    const headingStyle = {
        marginBottom: "0px", 
        overflow: "auto", 
        paddingBottom: open ? "15px" : "0"
    };
    let heading = "Main"
    if (props.component !== undefined) {
        heading = props.component.name;
    }
    return (
        <List.Item as="div">
            {props.component &&
                <ComponentManageMenu nodeId={props.nodeId} component={props.component} style={{float: "right"}} />
            }
            <Heading as="div" onClick={() => setOpen(!open)} style={headingStyle}>
                <Arrow opened={open} />
                <span>{heading}</span>
            </Heading>
            {open &&
                <Content>
                    {props.delegate &&
                        <Button size="small" onClick={() => remoteAction("addDelegate", {ID: props.nodeId})}>
                            Add Main
                        </Button>
                    }
                    {props.component &&
                        <React.Fragment>
                            <ComponentFields fields={props.component.fields} />
                            <ComponentButtons buttons={props.component.buttons} />
                        </React.Fragment>
                    }
                </Content>
            }
        </List.Item>
    );
}

function ComponentButtons(props) {
    const buttonStyle = {marginTop: "10px", width: "100%"};
    function onClicker(button) {
        if (button.onclick !== "") {
            return (event) => eval(button.onclick); 
        } else {
            return (event) => remoteAction("callMethod", button.path);
        }
    }
    return (props.buttons||[]).map((button, idx) =>
        <Button size="small" onClick={onClicker(button)} key={idx} value={button.path} style={buttonStyle}>
            {button.name}
        </Button>
    );
}

function Reference(props) {
    const [open, setOpen] = React.useState(false);
    const dropdownRef = React.useRef(null);
    const filterRef = React.useRef(null);
    const onBlur = () => {
        setTimeout(() => {
            if (dropdownRef.current.contains(document.activeElement)) {
                return;
            }
            setOpen(false);
        }, 200);
    }
    React.useEffect(() => {
        filterRef.current.focus();
    })
    function onClicker(entry) {
        return () => {
            props.onSet(entry[0]);
            setOpen(false);
        };
    }
    let nodes = [];
    for (const entry of Object.entries(InspectorContainer.instance.state.remote.nodePaths)) {
        nodes.push(entry);
    }
    return (
        <Dropdown style={{display: "block"}} managed active={open} onBlur={onBlur} ref={dropdownRef}>
            <Dropdown.Trigger>
                <Control iconRight>
                    <Input type="text" onClick={() => setOpen(!open)} readOnly size="small" value={props.value} title={props.type} />
                    <Icon align="right" onClick={() => props.onUnset()}>
                        <Delete size="small" />
                    </Icon>
                </Control>
            </Dropdown.Trigger>
            <Dropdown.Menu>
            <Dropdown.Content>
                <Dropdown.Item as="div">
                    <Input type="text" size="small" ref={filterRef} />
                </Dropdown.Item>
                <Dropdown.Divider />
                <Dropdown.Item as="div" style={{maxHeight: "100px", overflowY: "scroll", textAlign: "left"}}>
                    {nodes.map((node, idx) => 
                        <div key={idx} onClick={onClicker(node)}>{node[0]}</div>
                    )}
                </Dropdown.Item>
            </Dropdown.Content>
            </Dropdown.Menu>
        </Dropdown>
    );
}

function AddComponentButton(props) {
    const [open, setOpen] = React.useState(false);
    const [dropdownUp, setDropdownUp] = React.useState(true);
    const dropdownRef = React.useRef(null);
    const filterRef = React.useRef(null);
    const onBlur = () => {
        setTimeout(() => {
            if (dropdownRef.current.contains(document.activeElement)) {
                return;
            }
            setOpen(false);
        }, 200);
    }
    React.useEffect(() => {
        filterRef.current.focus();
        function checkLocationForDropdownDirection() {
            setDropdownUp(dropdownRef.current.getClientRects()[0].y > 200);
        }
        checkLocationForDropdownDirection();
        // TODO: cleanup this event listener?
        document.addEventListener('click', function (event) {
            checkLocationForDropdownDirection();
        });
    })
    function onClicker(component) {
        return () => {
            remoteAction("appendComponent", {Name: component.Name, ID: props.nodeId});
            setOpen(false);
        };
    }
    return (
        <section style={{textAlign: "center", marginBottom: "20px"}}>
            <Dropdown up={dropdownUp} managed active={open} onBlur={onBlur} ref={dropdownRef}>
                <Dropdown.Trigger>
                    <Button onClick={() => setOpen(!open)}>
                        <span>Add Component</span>
                        <Icon size="small"><i className={"fas fa-angle-"+(dropdownUp?"up":"down")}></i></Icon>
                    </Button>
                </Dropdown.Trigger>
                <Dropdown.Menu>
                    <Dropdown.Content>
                        <Dropdown.Item as="div">
                            <Input type="text" size="small" ref={filterRef} />
                        </Dropdown.Item>
                        <Dropdown.Item as="div" style={{maxHeight: "100px", overflowY: "scroll", textAlign: "left"}}>
                            {(props.components||[]).map((item, idx) =>
                                <div key={idx} onClick={onClicker(item)}>{item.Name}</div>
                            )}
                        </Dropdown.Item>
                    </Dropdown.Content>
                </Dropdown.Menu>
            </Dropdown>
        </section>
    );
}

function Inspector(props) {
    const node = props.node || {name: "no node", components: []};
    const delegatePlaceholder = (props.node && (node.components.length === 0 || node.components[0].name !== "Main"));
    let ancestors = [];
    if (node.path) {
        ancestors = node.path.split("/");
        ancestors.shift(); // drop empty first element
        ancestors.pop(); // drop last name element
    }
    return (
        <section>
            <Breadcrumb style={{marginBottom: "0"}}>
                {ancestors.map((name, idx) => 
                    <Breadcrumb.Item key={"path-"+idx} as="div" style={{color: "white", margin: "5px"}}>{name}</Breadcrumb.Item>
                )}
                <Breadcrumb.Item key="name" as="div" style={{color: "white", margin: "5px"}} active>{node.name}</Breadcrumb.Item>
            </Breadcrumb>
            <List>
                {delegatePlaceholder &&
                    <ComponentInspector delegate key="-1" nodeId={props.node.id} />
                }
                {node.components.map((com, idx) => 
                    <ComponentInspector component={com} nodeId={props.node.id} key={"com-"+idx} />
                )}
            </List>
            {//props.node &&
              //  <AddComponentButton components={props.components} nodeId={props.node.id} />
            }
            
        </section>
    );
}

function Arrow(props) {
    return (
        <Icon size="small"><i className={props.opened ? "fas fa-angle-down" : "fas fa-angle-right"}></i></Icon>
    );
}

function Autocomplete(props) {
    const [selected, setSelected] = React.useState(props.value);
    const [active, setActive] = React.useState(false);
    let filteredData = props.data || [];
    if (selected !== undefined && selected.length > 0) {
        filteredData = filteredData.filter((el) => el.startsWith(selected));
    }
    const items = filteredData.map((el) => 
        <Dropdown.Item key={el} onClick={() => {setSelected(el); setActive(false)}}>{el}</Dropdown.Item>
    )
    const onChange = (e) => {
        setSelected(e.target.value);
        setActive(true);
    };
    const onBlur = () => {
        setTimeout(() => setActive(false), 100);
    }
    return (
        <div>
            <Input type="text" value={selected} size="small" onChange={onChange} onBlur={onBlur} />
            <Dropdown.Content style={{display: active ? "block" : "none", "position": "absolute", "width": "90%"}}>
                {items}
            </Dropdown.Content>
        </div>
    );
}