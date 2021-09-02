import type {Uint, Match, MatchData, ListItem, UserID} from './types.js';
import type {WindowElement} from './lib/windows.js';
import {clearElement, createHTML} from './lib/dom.js';
import {br, button, div, input, label, li, span, ul} from './lib/html.js';
import {stringSort, node, NodeMap, NodeArray, noSort} from './lib/nodes.js';
import {desktop, shell as shellElement, windows} from './lib/windows.js';
import RPC, {rpc} from './rpc.js';

declare const pageLoad: Promise<void>;

type Param = {
	[node]: HTMLInputElement;
}

let nextID = 0;

const rcSort = (a: Redirect | Command, b: Redirect | Command) => a.id - b.id,
      matchData2Match = (md: MatchData[]) => md.map(([isSuffix, name]) => ({isSuffix, name})),
      shell = shellElement(),
      addLabel = (name: string, input: HTMLInputElement): [HTMLLabelElement, HTMLInputElement] => {
	const id = "ID_" + nextID++;
	return [label({"for": id}, name), createHTML(input, {id})];
      },
      maxID = 4294967296,
      editRedirect = (server: Server, data?: Redirect) => {
	const from = input({"type": "number", "min": 1, "max": 65535, "value": data?.from ?? 80}),
	      to = input({"value": data?.to}),
	      w = windows(),
	      matches = new MatchMaker(w, data?.match ?? []);
	shell.addWindow(createHTML(w, {"window-title": (data ? "Edit" : "Add") + " Redirect"}, [
		addLabel("From:", from),
		br(),
		addLabel("To:", to),
		br(),
		matches[node],
		button({"onclick": () => {
			const f = parseInt(from.value);
			if (f <= 0 || f > 65535) {
				w.alert("Invalid Port", `Invalid from port: ${from.value}`);
			} else if (to.value === "") {
				w.alert("Invalid address", `Invalid to address: ${to.value}`);
			} else if (matches.list.some(({name}) => name === "")) {
				w.alert("Invalid Match", "Cannot have empty match");
			} else {
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
						"match": matches.list,
					})
					.then(id => server.redirects.set(id, new Redirect(server, id, f, to.value, false, matches.list)))
				).catch(err => shell.alert("Error", err.message));
				w.remove();
			}
		}}, (data ? "Edit" : "Create") + " Redirect")
	]));
      },
      editCommand = (server: Server, data?: Command) => {
	const exe = input({"value": data?.exe}),
	      params = new NodeArray<Param>(div(), noSort, data?.params.map(p => ({[node]: input({"value": p})})) ?? []),
	      env = new EnvMaker(data?.env ?? {}),
	      userID = input({"type": "checkbox", "checked": data?.user !== undefined, "onchange": () => {
		      uid.toggleAttribute("disabled", !userID.checked);
		      gid.toggleAttribute("disabled", !userID.checked);
	      }}),
	      uid = input({"type": "number", "min": 0, "max": maxID, "value": data?.user?.uid, "disabled": data?.user === undefined}),
	      gid = input({"type": "number", "min": 0, "max": maxID, "value": data?.user?.gid, "disabled": data?.user === undefined}),
	      w = windows(),
	      matches = new MatchMaker(w, data?.match ?? []);
	shell.addWindow(createHTML(w, {"window-title": (data ? "Edit" : "Add") + " Command"}, [
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
		label("Environment:"),
		env[node],
		br(),
		matches[node],
		addLabel("Run as different user?:", userID),
		br(),
		addLabel("UID:", uid),
		br(),
		addLabel("GID:", gid),
		br(),
		button({"onclick": () => {
			if (exe.value === "") {
				w.alert("Invalid executable", "Executable cannot be empty");
			} else if (matches.list.some(({name}) => name === "")) {
				w.alert("Invalid Match", "Cannot have empty match");
			} else {
				const p = params.map(p => p[node].value),
				      e = env.toObject(),
				      u = userID.checked ? {
					      "uid": parseInt(uid.value),
					      "gid": parseInt(gid.value)
				      } : undefined;
				(data ?
					rpc.modifyCommand({
						"server": server.name,
						"id": data.id,
						"exe": exe.value,
						"params": p,
						"env": e,
						"match": matches.list,
						"user": u
					})
					.then(() => data.update(exe.value, p, e, matches.list, u)) : rpc.addCommand({
						"server": server.name,
						"exe": exe.value,
						"params": p,
						"env": e,
						"match": matches.list,
						"user": u
					})
					.then(id => server.commands.set(id, new Command(server, id, exe.value, p, e, matches.list)))
				)
				.catch(err => shell.alert("Error", err.message));
				w.remove();
			}
		}}, (data ? "Edit" : "Create") + " Command")
	]));
      },
      servers = new NodeMap<string, Server>(ul(), (a: Server, b: Server) => stringSort(a.name, b.name)),
      statusColours = ["#f00", "#0f0", "#f80"];

class MatchMaker {
	list: Match[];
	[node]: HTMLDivElement;
	u = ul();
	w: WindowElement;
	constructor(w: WindowElement, matches: Match[]) {
		this.list = [];
		for (const m of matches) {
			this.add(m);
		}
		if (matches.length === 0) {
			this.add();
		}
		this[node] = div([
			"Matches",
			this.u,
			button({"onclick": () => this.add()}, "Add Match")
		]);
		this.w = w;
	}
	add(m: Match = {"name": "", "isSuffix": false}) {
		this.list.push(m);
		const l = this.u.appendChild(li([
				input({"onchange": function(this: HTMLInputElement){m.name = this.value}, "value": m.name}),
				input({"type": "checkbox", "onchange": function(this: HTMLInputElement){m.isSuffix = this.checked}, "checked": m.isSuffix}),
				button({"onclick": () => {
					if (this.list.length === 1) {
						this.w.alert("Cannot remove Match", "Must have at least 1 Match");
					} else {
						this.list.splice(this.list.indexOf(m), 1);
						l.remove();
					}
				}}, "X")
		      ]));
	}
}

type Env = {
	key: HTMLInputElement;
	value: HTMLInputElement;
	[node]: HTMLLIElement;
}

class EnvMaker {
	nextID = 0;
	m: NodeMap<number, Env>;
	[node]: HTMLDivElement;
	constructor(environment: Record<string, string>) {
		this.m = new NodeMap<number, Env>(ul());
		for (const key in environment) {
			this.addEnv(key, environment[key]);
		}
		this[node] = div([
			this.m[node],
			button({"onclick": () => this.addEnv()}, "+")
		]);
	}
	addEnv(key = "", value = "") {
		const id = this.nextID++,
		      k = input({"value": key}),
		      v = input({value});
		this.m.set(id, {
			"key": k,
			"value": v,
			[node]: li([
				k,
				v,
				button({"onclick": () => this.m.delete(id)}, "-")
			])
		});
	}
	toObject() {
		const env: Record<string, string> = {};
		for (const e of this.m.values()) {
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
	active: boolean;
	[node]: HTMLLIElement;
	fromSpan: HTMLSpanElement;
	toSpan: HTMLSpanElement;
	statusSpan: HTMLSpanElement;
	startStop: HTMLButtonElement;
	constructor(server: Server, id: Uint, from: Uint, to: string, active: boolean, match: Match[]) {
		this.id = id;
		this.from = from;
		this.to = to;
		this.match = match;
		this.active = active;
		this.fromSpan = span(from + ""),
		this.toSpan = span(to);
		this.statusSpan = span({"class": "status", "style": {"color": statusColours[active ? 1 : 0]}})
		this.startStop = button({"onclick": () => {
			const sid = {"server": server.name, id};
			if (this.active) {
				rpc.stopRedirect(sid)
				.then(() => this.setActive(false))
				.catch(err => shell.alert("Error stopping redirect", err.message));
			} else {
				rpc.startRedirect(sid)
				.then(() => this.setActive(true))
				.catch(err => shell.alert("Error starting redirect", err.message));
			}
		}}, active ? "Stop" : "Start");
		this[node] = li	([
			this.statusSpan,
			this.fromSpan,
			this.toSpan,
			this.startStop,
			button({"onclick": () => editRedirect(server, this)}, "Edit"),
			button({"onclick": () => shell.confirm("Are you sure?", "Are you sure you wish to remove this redirect?").then(c => {
				if (c) {
					rpc.removeRedirect({"server": server.name, "id": id})
					.then(() => server.redirects.delete(id))
					.catch(e => shell.alert("Error removing redirect", e.message));
				}
			})}, "X")
		]);
	}
	update(from: Uint, to: string, match: Match[]) {
		this.fromSpan.innerText = (this.from = from) + "";
		this.toSpan.innerText = this.to = to;
		this.match = match;
	}
	setActive(v: boolean) {
		this.statusSpan.style.setProperty("color", statusColours[v ? 1 : 0]);
		this.startStop.innerText = v ? "Stop" : "Start";
	}
}

class Command {
	id: Uint;
	exe: string;
	params: string[];
	env: Record<string, string>;
	match: Match[];
	status: Uint;
	[node]: HTMLLIElement;
	exeSpan: HTMLSpanElement;
	statusSpan: HTMLSpanElement;
	error: string;
	user?: UserID;
	startStop: HTMLButtonElement;
	constructor(server: Server, id: Uint, exe: string, params: string[], env: Record<string, string>, match: Match[], status: Uint = 0, error = "", user?: UserID) {
		this.id = id;
		this.exe = exe;
		this.params = params;
		this.env = env;
		this.match = match;
		this.status = status;
		this.exeSpan = span(exe + " " + params.join(" "));
		this.statusSpan = span({"class": "status", "style": {"color": statusColours[status]}});
		this.error = error;
		this.user = user;
		this.startStop = button({"onclick": () => {
			const sid = {"server": server.name, id}
			if (this.status === 1) {
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
		}}, status === 1 ? "Stop" : "Start");
		this[node] = li([
			this.statusSpan,
			this.exeSpan,
			this.startStop,
			button({"onclick": () => editCommand(server, this)}, "Edit"),
			button({"onclick": () => shell.confirm("Are you sure?", "Are you sure you wish to remove this command?").then(c => {
				if (c) {
					rpc.removeCommand({"server": server.name, "id": id})
					.then(() => server.commands.delete(id))
					.catch(e => shell.alert("Error removing command", e.message));
				}
			})}, "X")
		]);
	}
	update(exe: string, params: string[], env: Record<string, string>, match: Match[], user?: UserID) {
		this.exeSpan.innerText = (this.exe = exe) + " " + (this.params = params).join(" ");
		this.env = env;
		this.match = match;
		this.user = user;
	}
	setStatus (s: Uint) {
		this.statusSpan.style.setProperty("color", statusColours[this.status = s]);
		if (s === 1) {
			this.startStop.innerText = "Stop";
		} else {
			this.startStop.innerText = "Start";
		}
	}
	setError (e: string) {
		this.error = e;
	}
}

class Server {
	name: string;
	redirects: NodeMap<Uint, Redirect>;
	commands: NodeMap<Uint, Command>;
	[node]: HTMLLIElement;
	nameSpan: HTMLSpanElement;
	constructor([name, rs, cs]: ListItem) {
		this.name = name;
		this.redirects = new NodeMap<Uint, Redirect & {[node]: HTMLLIElement}>(ul(), rcSort, rs.map(([id, from, to, active, _, ...match]) => [id, new Redirect(this, id, from, to, active, matchData2Match(match))]));
		this.commands = new NodeMap<Uint, Command & {[node]: HTMLLIElement}>(ul(), rcSort, cs.map(([id, exe, params, env, status, error, ...match]) => [id, new Command(this, id, exe, params, env, matchData2Match(match), status, error)]));
		this.nameSpan = div(name);
		this[node] = li([
			div([
				this.nameSpan,
				button({"onclick": () => shell.prompt("New Name", "Plese enter a new name for this server", this.name).then(name => {
					if (name && name !== this.name) {
						rpc.rename([this.name, name]).catch(err => shell.alert("Error", err.message));
						this.setName(name);
					}
				})}, "Rename"),
				button({"onclick": () => shell.confirm("Remove", "Are you sure you wish to remove this server?").then(ok => {
					if (ok) {
						rpc.remove(this.name).catch(err => shell.alert("Error", err.message));
					}
				})}, "Remove")
			]),
			button({"onclick": () => editRedirect(this)}, "Add Redirect"),
			button({"onclick": () => editCommand(this)}, "Add Command"),
			this.redirects[node],
			this.commands[node]
		]);
	}
	setName(name: string) {
		servers.delete(this.name);
		this.nameSpan.innerText = this.name = name;
		servers.set(name, this);
	}
}

pageLoad.then(() => RPC(`ws${window.location.protocol.slice(4)}//${window.location.host}/socket`).then(() => {rpc.waitList().then(list => {
	for (const s of list) {
		servers.set(s[0], new Server(s));
	}
	clearElement(document.body).appendChild(createHTML(shell, desktop([
		button({"onclick": () => shell.prompt("Server Name", "Please enter a name for the new server", "").then(name => {
			if (name) {
				rpc.add(name).catch(err => shell.alert("Error", err)).then(() => servers.set(name, new Server([name, [], []])));
			}
		})}, "New Server"),
		servers[node]
	])));
	rpc.waitAdd().then(name => servers.set(name, new Server([name, [], []])));
	rpc.waitRename().then(([oldName, newName]) => servers.get(oldName)?.setName(newName));
	rpc.waitRemove().then(name => servers.delete(name));
	rpc.waitAddRedirect().then(r => {
		const server = servers.get(r.server);
		if (server) {
			server.redirects.set(r.id, new Redirect(server, r.id, r.from, r.to, false, r.match));
		}
	});
	rpc.waitModifyRedirect().then(r => servers.get(r.server)?.redirects.get(r.id)?.update(r.from, r.to, r.match));
	rpc.waitRemoveRedirect().then(r => servers.get(r.server)?.redirects.delete(r.id));
	rpc.waitAddCommand().then(c => {
		const server = servers.get(c.server);
		if (server) {
			server.commands.set(c.id, new Command(server, c.id, c.exe, c.params, c.env, c.match, 0, "", c.user));
		}
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
