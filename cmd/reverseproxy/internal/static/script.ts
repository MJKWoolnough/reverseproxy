import type {ListItem, Match, MatchData, Uint, UserID} from './types.js';
import type {Props} from './lib/dom.js';
import type {WindowElement} from './lib/windows.js';
import {amendNode, clearNode} from './lib/dom.js';
import {br, button, div, h1, img, input, label, li, span, table, tbody, td, th, thead, tr, ul} from './lib/html.js';
import pageLoad from './lib/load.js';
import {NodeArray, NodeMap, node, noSort, stringSort} from './lib/nodes.js';
import {circle, g, line, path, polyline, rect, svg, svgData, symbol, title, use} from './lib/svg.js';
import {desktop, shell as shellElement, windows} from './lib/windows.js';
import RPC, {rpc} from './rpc.js';

let nextID = 0;

const rcSort = (a: Redirect | Command, b: Redirect | Command) => a.id - b.id,
      matchData2Match = (md: MatchData[]) => md.map(([isSuffix, name]) => ({isSuffix, name})),
      shell = shellElement(),
      addLabel = (name: string, input: HTMLInputElement): [HTMLLabelElement, HTMLInputElement] => {
	const id = "ID_" + nextID++;
	return [label({"for": id}, name), amendNode(input, {id})];
      },
      maxID = 4294967296,
      symbols = svg(),
      addSymbol = (s: SVGSymbolElement) => {
	const id = "ID_" + nextID++;
	amendNode(symbols, amendNode(s, {id}));
	return [
		(props: Exclude<Props, NamedNodeMap> = {}) => svg(props, [
			typeof props["title"] === "string" ? title(props["title"]) : [],
			use({"href": `#${id}`})
		]),
		svgData(s)
	] as const;
      },
      [remove, removeIcon] = addSymbol(symbol({"viewBox": "0 0 32 34"}, path({"d": "M10,5 v-3 q0,-1 1,-1 h10 q1,0 1,1 v3 m8,0 h-28 q-1,0 -1,1 v2 q0,1 1,1 h28 q1,0 1,-1 v-2 q0,-1 -1,-1 m-2,4 v22 q0,2 -2,2 h-20 q-2,0 -2,-2 v-22 m2,3 v18 q0,1 1,1 h3 q1,0 1,-1 v-18 q0,-1 -1,-1 h-3 q-1,0 -1,1 m7.5,0 v18 q0,1 1,1 h3 q1,0 1,-1 v-18 q0,-1 -1,-1 h-3 q-1,0 -1,1 m7.5,0 v18 q0,1 1,1 h3 q1,0 1,-1 v-18 q0,-1 -1,-1 h-3 q-1,0 -1,1", "style": "stroke: currentColor", "fill": "none"}))),
      [rename, renameIcon] = addSymbol(symbol({"viewBox": "0 0 30 20"}, path({"d": "M1,5 v10 h28 v-10 Z M17,1 h10 m-5,0 V19 m-5,0 h10", "style": "stroke: currentColor", "stroke-linejoin": "round", "fill": "#fff"}))),
      [edit, editIcon] = addSymbol(symbol({"viewBox": "0 0 70 70", "fill": "#fff", "stroke": "#000"}, [polyline({"points": "51,7 58,0 69,11 62,18 51,7 7,52 18,63 62,18", "stroke-width": 2}), path({"d": "M7,52 L1,68 L18,63 M53,12 L14,51 M57,16 L18,55"})])),
      [addRedirect, addRedirectIcon] = addSymbol(symbol({"viewBox": "0 0 100 100"}, [
	path({"d": "M10,80 h40 a1,1 0,0,0 0,-60 h-20", "stroke-width": 15, "stroke": "#000", "fill": "none"}),
	path({"d": "M30,5 v30 l-20,-15 z", "fill": "#000"}),
	path({"d": "M60,40 v50 m-25,-25 h50", "stroke-width": 15, "stroke": "#0f0", "fill": "none"})
      ])),
      [addCommand, addCommandIcon] = addSymbol(symbol({"viewBox": "0 0 100 100"}, [
	rect({"width": 100, "height": 100, "fill": "#000", "rx": 10}),
	rect({"width": 100, "height": 30, "fill": "#aaa", "rx": 10}),
	rect({"y": 15, "width": 100, "height": 20, "fill": "#000"}),
	path({"d": "M10,25 l10,10 l-10,10 M25,45 h20", "stroke": "#fff", "stroke-width": 5}),
	path({"d": "M60,40 v50 m-25,-25 h50", "stroke-width": 15, "stroke": "#0f0", "fill": "none"})
      ])),
      [addServer, addServerIcon] = addSymbol(symbol({"viewBox": "0 0 100 100"}, [
	g({"id": "sr", "transform": "translate(0, 3)"}, [
		rect({"x": 2, "width": 96, "height": 20, "stroke": "#000", "stroke-width": 2, "fill": "#fff", "rx": 5}),
		line({"x1": 10, "x2": 30, "y1": 7, "y2": 7, "stroke": "#000"}),
		circle({"id": "sc", "cx": 70, "r": 2, "cy": 7, "fill": "#000"}),
		use({"href": "#sc", "x": 5}),
		use({"href": "#sc", "x": 10}),
		use({"href": "#sc", "x": 15})
	]),
	use({"href": "#sr", "y": 25}),
	use({"href": "#sr", "y": 50}),
	use({"href": "#sr", "y": 75}),
	path({"d": "M60,40 v50 m-25,-25 h50", "stroke-width": 15, "stroke": "#0f0", "fill": "none"})
      ])),
      [start, startIcon] = addSymbol(symbol({"viewBox": "0 0 100 100"}, [
	      path({"d": "M10,10 v80 L90,50 z", "fill": "#000"}),
	      rect({"x": 10, "y": 10, "width": 80, "height": 80, "style": {"display": "var(--h, none)"}})
      ])),
      [info, infoIcon] = addSymbol(symbol({"viewBox": "0 0 100 100"}, [
	      circle({"cx": 50, "cy": 50, "stroke-width": 6, "r": 47, "stroke": "#fff", "fill": "#15a"}),
	      rect({"x": 42, "y": 45, "width": 16, "height": 40, "rx": 8, "fill": "#fff"}),
	      circle({"cx": 50, "cy": 30, "r": 8, "fill": "#fff"})
      ])),
      editRedirect = (server: Server, data?: Redirect) => {
	const icon = data ? editIcon : addRedirectIcon,
	      from = input({"type": "number", "min": 1, "max": 65535, "value": data?.from ?? 80}),
	      to = input({"value": data?.to}),
	      w = windows(),
	      matches = new MatchMaker(w, data?.match ?? []);
	shell.addWindow(amendNode(w, {"window-title": (data ? "Edit" : "Add") + " Redirect", "window-icon": icon}, [
		addLabel("From:", from),
		br(),
		addLabel("To:", to),
		br(),
		matches[node],
		button({"onclick": function(this: HTMLButtonElement) {
			const f = parseInt(from.value);
			if (f <= 0 || f > 65535) {
				w.alert("Invalid Port", `Invalid from port: ${from.value}`, icon);
			} else if (to.value === "") {
				w.alert("Invalid address", `Invalid to address: ${to.value}`, icon);
			} else if (matches.list.some(({name}) => name === "")) {
				w.alert("Invalid Match", "Cannot have empty match", icon);
			} else {
				amendNode(this, {"disabled": true});
				(data ?
					rpc.modifyRedirect({
						"server": server.name,
						"id": data.id,
						"from": f,
						"to": to.value,
						"match": matches.list
					})
					.then(() => data.update(f, to.value, matches.list)) : rpc.addRedirect({
						"server": server.name,
						"from": f,
						"to": to.value,
						"match": matches.list
					})
					.then(id => server.redirects.set(id, new Redirect(server, id, f, to.value, false, matches.list)))
				)
				.then(() => w.remove())
				.catch(err => w.alert("Error", err.message, icon))
				.finally(() => amendNode(this, {"disabled": false}));
			}
		}}, (data ? "Edit" : "Create") + " Redirect")
	]));
      },
      editCommand = (server: Server, data?: Command) => {
	const icon = data ? editIcon : addCommandIcon,
	      exe = input({"value": data?.exe}),
	      params = new NodeArray(div(), noSort, data?.params.map(p => ({[node]: input({"value": p})})) ?? []),
	      env = new EnvMaker(data?.env ?? {}),
	      workDir = input({"value": data?.workDir}),
	      userID = input({"type": "checkbox", "checked": data?.user !== undefined, "onchange": () => {
		      amendNode(uid, {"disabled": !userID.checked});
		      amendNode(gid, {"disabled": !userID.checked});
	      }}),
	      uid = input({"type": "number", "min": 0, "max": maxID, "value": data?.user?.uid, "disabled": data?.user === undefined}),
	      gid = input({"type": "number", "min": 0, "max": maxID, "value": data?.user?.gid, "disabled": data?.user === undefined}),
	      w = windows(),
	      matches = new MatchMaker(w, data?.match ?? []);
	shell.addWindow(amendNode(w, {"window-title": (data ? "Edit" : "Add") + " Command", "window-icon": icon}, [
		addLabel("Executable:", exe),
		br(),
		label("Params:"),
		params[node],
		button({"onclick": () => {
			if (params.length > 0) {
				params.pop();
			}
		}}, "-"),
		button({"onclick": () => params.push({[node]: input()})}, "+"),
		br(),
		addLabel("Working Directory: ", workDir),
		br(),
		br(),
		env[node],
		br(),
		matches[node],
		br(),
		addLabel("Run as different user?:", userID),
		br(),
		addLabel("UID:", uid),
		br(),
		addLabel("GID:", gid),
		br(),
		button({"onclick": function(this: HTMLButtonElement) {
			const u = parseInt(uid.value),
			      g = parseInt(gid.value);
			if (exe.value === "") {
				w.alert("Invalid executable", "Executable cannot be empty", icon);
			} else if (matches.list.some(({name}) => name === "")) {
				w.alert("Invalid Match", "Cannot have empty match", icon);
			} else if (u < 0 || u > maxID) {
				w.alert("Invalid UID", `UID must be in range 0 < uid < ${maxID}`, icon);
			} else if (g < 0 || g > maxID) {
				w.alert("Invalid GID", `GID must be in range 0 < uid < ${maxID}`, icon);
			} else {
				amendNode(this, {"disabled": true});
				const p = params.map(p => p[node].value),
				      e = env.toObject(),
				      ids = userID.checked ? {
					      "uid": u,
					      "gid": g
				      } : undefined;
				(data ?
					rpc.modifyCommand({
						"server": server.name,
						"id": data.id,
						"exe": exe.value,
						"params": p,
						"workDir": workDir.value,
						"env": e,
						"match": matches.list,
						"user": ids
					})
					.then(() => data.update(exe.value, p, e, matches.list, ids)) : rpc.addCommand({
						"server": server.name,
						"exe": exe.value,
						"params": p,
						"workDir": workDir.value,
						"env": e,
						"match": matches.list,
						"user": ids
					})
					.then(id => server.commands.set(id, new Command(server, id, exe.value, p, workDir.value, e, matches.list, ids)))
				)
				.then(() => w.remove())
				.catch(err => w.alert("Error", err.message, icon))
				.finally(() => amendNode(this, {"disabled": false}));
			}
		}}, (data ? "Edit" : "Create") + " Command")
	]));
      },
      servers = new NodeMap<string, Server>(ul(), (a: Server, b: Server) => stringSort(a.name, b.name)),
      statusColours = ["#f00", "#0f0", "#f80"];

class MatchMaker {
	list: Match[];
	[node]: HTMLDivElement;
	#u = tbody();
	#w: WindowElement;
	constructor(w: WindowElement, matches: Match[]) {
		this.list = [];
		for (const m of matches) {
			this.add(m);
		}
		if (matches.length === 0) {
			this.add();
		}
		this[node] = div([
			table([
				thead(tr([
					th("Matches"),
					th("Is Suffix?"),
					th(img({"src": removeIcon, "style": {"width": "1em", "height": "1em"}}))
				])),
				this.#u
			]),
			button({"onclick": () => this.add()}, "Add Match")
		]);
		this.#w = w;
	}
	add(m: Match = {"name": "", "isSuffix": false}) {
		this.list.push(m);
		const l = tr([
			td(input({"onchange": function(this: HTMLInputElement){m.name = this.value}, "value": m.name})),
			td(input({"type": "checkbox", "onchange": function(this: HTMLInputElement){m.isSuffix = this.checked}, "checked": m.isSuffix})),
			td(remove({"title": "Remove Match", "onclick": () => {
				if (this.list.length === 1) {
					this.#w.alert("Cannot remove Match", "Must have at least 1 Match", removeIcon);
				} else {
					this.list.splice(this.list.indexOf(m), 1);
					l.remove();
				}
			}}))
		      ]);
		amendNode(this.#u, l);
	}
}

type Env = {
	key: HTMLInputElement;
	value: HTMLInputElement;
	[node]: HTMLTableRowElement;
}

class EnvMaker {
	#nextID = 0;
	#m: NodeMap<number, Env>;
	[node]: HTMLDivElement;
	constructor(environment: Record<string, string>) {
		this.#m = new NodeMap<number, Env>(tbody());
		for (const key in environment) {
			this.addEnv(key, environment[key]);
		}
		this[node] = div([
			table([
				thead(tr([
					th("Env Key"),
					th("Value"),
					th(img({"src": removeIcon, "style": {"width": "1em", "height": "1em"}}))
				])),
				this.#m[node]
			]),
			button({"onclick": () => this.addEnv()}, "+")
		]);
	}
	addEnv(key = "", value = "") {
		const id = this.#nextID++,
		      k = input({"value": key}),
		      v = input({value});
		this.#m.set(id, {
			"key": k,
			"value": v,
			[node]: tr([
				td(k),
				td(v),
				td(remove({"onclick": () => this.#m.delete(id)}))
			])
		});
	}
	toObject() {
		const env: Record<string, string> = {};
		for (const e of this.#m.values()) {
			if (e.key.value) {
				env[e.key.value] = e.value.value;
			}
		}
		return env;
	}
}

class Redirect {
	id: Uint;
	from: Uint;
	to: string;
	match: Match[];
	#active: boolean;
	[node]: HTMLLIElement;
	#fromSpan: HTMLSpanElement;
	#toSpan: HTMLSpanElement;
	#statusSpan: HTMLSpanElement;
	#startStop: SVGSVGElement;
	constructor(server: Server, id: Uint, from: Uint, to: string, active: boolean, match: Match[]) {
		this.id = id;
		this.from = from;
		this.to = to;
		this.match = match;
		this.#active = active;
		this.#fromSpan = span(from + ""),
		this.#toSpan = span(to);
		this.#statusSpan = span({"style": {"color": statusColours[+active]}});
		this.#startStop = start({"onclick": () => {
			const sid = {"server": server.name, id};
			if (this.#active) {
				rpc.stopRedirect(sid)
				.then(() => this.setActive(false))
				.catch(err => shell.alert("Error stopping redirect", err.message, startIcon));
			} else {
				rpc.startRedirect(sid)
				.then(() => this.setActive(true))
				.catch(err => shell.alert("Error starting redirect", err.message, startIcon));
			}
		}, "style": active ? "--h: auto" : undefined});
		this[node] = li	([
			this.#statusSpan,
			this.#fromSpan,
			" ➔ ",
			this.#toSpan,
			this.#startStop,
			edit({"title": "Edit Redirect", "onclick": () => editRedirect(server, this)}),
			remove({"title": "Remove Redirect", "onclick": () => shell.confirm("Are you sure?", "Are you sure you wish to remove this redirect?", removeIcon).then(c => {
				if (c) {
					rpc.removeRedirect({"server": server.name, "id": id})
					.then(() => server.redirects.delete(id))
					.catch(e => shell.alert("Error removing redirect", e.message, removeIcon));
				}
			})})
		]);
	}
	update(from: Uint, to: string, match: Match[]) {
		this.#fromSpan.innerText = (this.from = from) + "";
		this.#toSpan.innerText = this.to = to;
		this.match = match;
	}
	setActive(v: boolean) {
		amendNode(this.#statusSpan, {"style": {"color": statusColours[+v]}});
		amendNode(this.#startStop, {"style": {"--h": (this.#active = v) ? "auto" : undefined}});
	}
}

class Command {
	id: Uint;
	exe: string;
	params: string[];
	workDir: string;
	env: Record<string, string>;
	match: Match[];
	#status: Uint;
	[node]: HTMLLIElement;
	#exeSpan: HTMLSpanElement;
	#statusSpan: HTMLSpanElement;
	#error: string;
	user?: UserID;
	#startStop: SVGSVGElement;
	constructor(server: Server, id: Uint, exe: string, params: string[], workDir: string, env: Record<string, string>, match: Match[], user?: UserID, status: Uint = 0, error = "") {
		this.id = id;
		this.exe = exe;
		this.params = params;
		this.workDir = workDir;
		this.env = env;
		this.match = match;
		this.#status = status;
		this.#exeSpan = span(exe + " " + params.join(" "));
		this.#statusSpan = span({"class": "status", "style": {"color": statusColours[status]}});
		this.#error = error;
		this.user = user;
		this.#startStop = start({"onclick": () => {
			const sid = {"server": server.name, id}
			if (this.#status === 1) {
				rpc.stopCommand(sid)
				.then(() => this.setStatus(0))
				.catch(err => {
					this.setStatus(2);
					shell.alert("Error stopping command", err.message)
				});
			} else {
				rpc.startCommand(sid)
				.then(() => this.setStatus(1))
				.catch(err => {
					this.setStatus(2);
					this.setError(err.message);
					shell.alert("Error starting command", err.message);
				});
			}
		}, "style": status === 1 ? "--h: auto" : undefined});
		this[node] = li([
			this.#statusSpan,
			this.#exeSpan,
			this.#startStop,
			info({"title": "Command Information", "onclick": () => (this.#status === 1 ? rpc.getCommandPorts({"server": server.name, id}) : Promise.resolve([])).then(ports => shell.addWindow(windows({"window-title": "Command Information", "window-icon": infoIcon}, [
				div(`Error: ${this.#error}`),
				div(`Open Ports: ${ports.sort((a: Uint, b: Uint) => b - a).join(", ")}`)
			]))).catch(e => shell.alert("Error getting information", e.message, infoIcon))}),
			edit({"title": "Edit Command", "onclick": () => editCommand(server, this)}),
			remove({"title": "Remove Command", "onclick": () => shell.confirm("Are you sure?", "Are you sure you wish to remove this command?", removeIcon).then(c => {
				if (c) {
					rpc.removeCommand({"server": server.name, "id": id})
					.then(() => server.commands.delete(id))
					.catch(e => shell.alert("Error removing command", e.message, removeIcon));
				}
			})})
		]);
	}
	update(exe: string, params: string[], env: Record<string, string>, match: Match[], user?: UserID) {
		this.#exeSpan.innerText = (this.exe = exe) + " " + (this.params = params).join(" ");
		this.env = env;
		this.match = match;
		this.user = user;
	}
	setStatus (s: Uint) {
		amendNode(this.#statusSpan, {"style": {"color": statusColours[this.#status = s]}});
		amendNode(this.#startStop, {"style": {"--h": s === 1 ? "auto" : undefined}});
	}
	setError (e: string) {
		this.#error = e;
	}
}

class Server {
	name: string;
	redirects: NodeMap<Uint, Redirect>;
	commands: NodeMap<Uint, Command>;
	[node]: HTMLLIElement;
	#nameSpan: HTMLSpanElement;
	constructor([name, rs, cs]: ListItem) {
		this.name = name;
		this.redirects = new NodeMap<Uint, Redirect & {[node]: HTMLLIElement}>(ul(), rcSort, rs.map(([id, from, to, active, _, ...match]) => [id, new Redirect(this, id, from, to, active, matchData2Match(match))]));
		this.commands = new NodeMap<Uint, Command & {[node]: HTMLLIElement}>(ul(), rcSort, cs.map(([id, exe, params, workDir, env, status, error, user, ...match]) => [id, new Command(this, id, exe, params, workDir, env, matchData2Match(match), user || undefined, status, error)]));
		this.#nameSpan = span(name);
		this[node] = li([
			div([
				this.#nameSpan,
				addRedirect({"title": "Add Redirect", "onclick": () => editRedirect(this)}),
				addCommand({"title": "Add Command", "onclick": () => editCommand(this)}),
				rename({"title": "Rename Server", "onclick": () => shell.prompt("New Name", "Plese enter a new name for this server", this.name, renameIcon).then(name => {
					if (name && name !== this.name) {
						rpc.rename([this.name, name]).catch(err => shell.alert("Error", err.message, renameIcon));
						this.setName(name);
					}
				})}),
				remove({"title": "Remove Server", "onclick": () => shell.confirm("Remove", "Are you sure you wish to remove this server?", removeIcon).then(ok => {
					if (ok) {
						rpc.remove(this.name).catch(err => shell.alert("Error", err.message, removeIcon)).then(() => servers.delete(this.name));
					}
				})})
			]),
			this.redirects[node],
			this.commands[node]
		]);
	}
	setName(name: string) {
		servers.delete(this.name);
		this.#nameSpan.innerText = this.name = name;
		servers.set(name, this);
	}
}

pageLoad.then(() => RPC("/socket").then(() => {rpc.waitList().then(list => {
	for (const s of list) {
		servers.set(s[0], new Server(s));
	}
	clearNode(document.body, amendNode(shell, desktop([
		symbols,
		h1("Reverse Proxy"),
		addServer({"title": "Add Server", "onclick": () => shell.prompt("Server Name", "Please enter a name for the new server", "", addServerIcon).then(name => {
			if (name) {
				rpc.add(name).catch(err => shell.alert("Error", err, addServerIcon)).then(() => servers.set(name, new Server([name, [], []])));
			}
		})}),
		servers[node]
	])));
	rpc.waitAdd().then(name => servers.set(name, new Server([name, [], []])));
	rpc.waitRename().then(([oldName, newName]) => servers.get(oldName)?.setName(newName));
	rpc.waitRemove().then(name => servers.delete(name));
	rpc.waitAddRedirect().then(r => {
		const server = servers.get(r.server);
		server?.redirects.set(r.id, new Redirect(server, r.id, r.from, r.to, false, r.match));
	});
	rpc.waitModifyRedirect().then(r => servers.get(r.server)?.redirects.get(r.id)?.update(r.from, r.to, r.match));
	rpc.waitRemoveRedirect().then(r => servers.get(r.server)?.redirects.delete(r.id));
	rpc.waitAddCommand().then(c => {
		const server = servers.get(c.server);
		server?.commands.set(c.id, new Command(server, c.id, c.exe, c.params, c.workDir, c.env, c.match, c.user, 0, ""));
	});
	rpc.waitModifyCommand().then(c => servers.get(c.server)?.commands.get(c.id)?.update(c.exe, c.params, c.env, c.match, c.user));
	rpc.waitRemoveCommand().then(c => servers.get(c.server)?.commands.delete(c.id));
	rpc.waitStartRedirect().then(r => servers.get(r.server)?.redirects.get(r.id)?.setActive(true));
	rpc.waitStopRedirect().then(r => servers.get(r.server)?.redirects.get(r.id)?.setActive(false));
	rpc.waitStartCommand().then(c => servers.get(c.server)?.commands.get(c.id)?.setStatus(1));
	rpc.waitStopCommand().then(c => servers.get(c.server)?.commands.get(c.id)?.setStatus(0));
	rpc.waitCommandStopped().then(([server, id]) => servers.get(server)?.commands.get(id)?.setStatus(2));
	rpc.waitCommandError().then(c => servers.get(c.server)?.commands.get(c.id)?.setError(c.err));
})}));
