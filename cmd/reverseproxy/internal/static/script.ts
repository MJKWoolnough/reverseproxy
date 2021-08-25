import type {Uint, Match, MatchData, ListItem} from './types.js';
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

const rcSort = (a: Redirect | Command, b: Redirect | Command) => a.id - b.id,
      matchData2Match = (md: MatchData[]) => md.map(([isSuffix, name]) => ({isSuffix, name})),
      noEnum = {"enumerable": false},
      redirectProps = {
	"server": noEnum,
	"fromSpan": noEnum,
	"toSpan": noEnum
      },
      commandProps = {
	"server": noEnum,
	"exeSpan": noEnum
      },
      shell = shellElement(),
      editRedirect = (server: Server, data?: Redirect) => {
	const from = input({"type": "number", "min": 1, "max": 65535, "value": data?.from ?? 80}),
	      to = input({"value": data?.to}),
	      w = windows(),
	      matches = new MatchMaker(w, data?.match ?? []);
	shell.addWindow(createHTML(w, {"window-title": "Add Redirect"}, [
		label("From:"),
		from,
		br(),
		label("To:"),
		to,
		br(),
		matches[node],
		button({"onclick": () => {
			const f = parseInt(from.value);
			if (f <= 0 || f >= 65535) {
				w.alert("Invalid Port", `Invalid from port: ${from.value}`);
			} else if (to.value === "") {
				w.alert("Invalid address", `Invalid to address: ${to.value}`);
			} else if (matches.list.some(({name}) => name === "")) {
				w.alert("Invalid Match", "Cannot have empty match");
			} else if (data) {
				rpc.modifyRedirect({
					"server": server.name,
					"id": data.id,
					"from": f,
					"to": to.value,
					"match": matches.list
				})
				.then(() => {
					data.setFrom(f);
					data.setTo(to.value);
					data.match = matches.list;
				})
				.catch(err => shell.alert("Error", err.message));
				w.remove();
			} else {
				rpc.addRedirect({
					"server": server.name,
					"from": f,
					"to": to.value,
					"match": matches.list,
				})
				.then(id => server.redirects.set(id, new Redirect(server, id, f, to.value, false, matches.list)))
				.catch(err => shell.alert("Error", err.message));
				w.remove();
			}
		}}, "Create Redirect")
	]));
      },
      editCommand = (server: Server, data?: Command) => {
	const exe = input({"value": data?.exe}),
	      params = new NodeArray<Param>(div(), noSort, data?.params.map(p => ({[node]: input({"value": p})})) ?? []),
	      env = new EnvMaker(data?.env ?? {}),
	      w = windows(),
	      matches = new MatchMaker(w, data?.match ?? []);
	shell.addWindow(createHTML(w, {"window-title": "Add Command"}, [
		label("Executable:"),
		exe,
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
		button({"onclick": () => {
			if (exe.value === "") {
				w.alert("Invalid executable", "Executable cannot be empty");
			} else if (matches.list.some(({name}) => name === "")) {
				w.alert("Invalid Match", "Cannot have empty match");
			} else {
				const p = params.map(p => p[node].value),
				      e = env.toObject();
				(data ?
					rpc.modifyCommand({
						"server": server.name,
						"id": data.id,
						"exe": exe.value,
						"params": p,
						"env": e,
						"match": matches.list,
					})
					.then(() => {
						data.setExe(exe.value);
						data.setParams(p);
						data.env = e;
						data.match = matches.list;
					}) : rpc.addCommand({
						"server": server.name,
						"exe": exe.value,
						"params": p,
						"env": e,
						"match": matches.list,
					})
					.then(id => server.commands.set(id, new Command(server, id, exe.value, p, e, matches.list)))
				)
				.catch(err => shell.alert("Error", err.message));
				w.remove();
			}
		}}, "Create Command")
	]));
      };

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
	active: boolean;
	match: Match[];
	[node]: HTMLLIElement;
	fromSpan: HTMLSpanElement;
	toSpan: HTMLSpanElement;
	constructor(server: Server, id: Uint, from: Uint, to: string, active: boolean, match: Match[]) {
		this.id = id;
		this.from = from;
		this.to = to;
		this.active = active;
		this.match = match;
		this.fromSpan = span(from + ""),
		this.toSpan = span(to);
		this[node] = li([
			this.fromSpan,
			this.toSpan,
			button({"onclick": () => editRedirect(server, this)}, "Edit"),
			button({"onclick": () => shell.confirm("Are you sure?", "Are you sure you wish to remove this redirect?").then(c => {
				if (c) {
					server.redirects.delete(id);
					rpc.removeRedirect({"server": server.name, "id": id})
					.catch(e => shell.alert("Error removing redirect", e.message));
				}
			})}, "X")
		]);
		Object.defineProperties(this, redirectProps);
		Object.defineProperty(this, "name", {"get": () => server.name, "enumerable": true});
	}
	setFrom(f: Uint) {
		this.fromSpan.innerText = (this.from = f) + "";
	}
	setTo(t: string) {
		this.toSpan.innerText = this.to = t;
	}
}

class Command {
	id: Uint;
	exe: string;
	params: string[];
	env: Record<string, string>;
	match: Match[];
	[node]: HTMLLIElement;
	exeSpan: HTMLSpanElement;
	constructor(server: Server, id: Uint, exe: string, params: string[], env: Record<string, string>, match: Match[]) {
		this.id = id;
		this.exe = exe;
		this.params = params;
		this.env = env;
		this.match = match;
		this.exeSpan = span(exe + " " + params.join(" "));
		this[node] = li(this.exeSpan);
		Object.defineProperties(this, commandProps);
		Object.defineProperty(this, "name", {"get": () => server.name, "enumerable": true});
	}
	setExe (e: string) {
		this.exeSpan.innerText = (this.exe = e) + " " + this.params.join(" ");
	}
	setParams (p: string[]) {
		this.exeSpan.innerText = this.exe + " " + (this.params = p).join(" ");
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
		this.commands = new NodeMap<Uint, Command & {[node]: HTMLLIElement}>(ul(), rcSort, cs.map(([id, exe, params, env, _a, _b, ...match]) => [id, new Command(this, id, exe, params, env, matchData2Match(match))]));
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
		this.nameSpan.innerText = this.name = name;
	}
}

pageLoad.then(() => RPC(`ws${window.location.protocol.slice(4)}//${window.location.host}/socket`).then(rpc => {rpc.waitList().then(list => {
	const servers = new NodeMap<string, Server>(ul(), (a: Server, b: Server) => stringSort(a.name, b.name), list.map(i => [i[0], new Server(i)])),
	      s = clearElement(document.body).appendChild(createHTML(shell, desktop([
		button({"onclick": () => s.prompt("Server Name", "Please enter a name for the new server", "").then(name => {
			if (name) {
				rpc.add(name).catch(err => s.alert("Error", err)).then(() => servers.set(name, new Server([name, [], []])));
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
	rpc.waitRemoveRedirect().then(r => {
		const server = servers.get(r.server);
		if (server) {
			server.redirects.delete(r.id);
		}
	});
	rpc.waitModifyRedirect().then(r => {
		const redirect = servers.get(r.server)?.redirects.get(r.id);
		if (redirect) {
			redirect.setFrom(r.from);
			redirect.setTo(r.to);
			redirect.match = r.match;
		}
	});

})}));
