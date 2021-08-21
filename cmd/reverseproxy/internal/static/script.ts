import type {Uint, Match, MatchData, ListItem} from './types.js';
import type {WindowElement} from './lib/windows.js';
import {clearElement, createHTML} from './lib/dom.js';
import {br, button, div, input, label, li, span, ul} from './lib/html.js';
import {stringSort, node, MapNode} from './lib/ordered.js';
import {desktop, shell as shellElement, windows} from './lib/windows.js';
import RPC, {rpc} from './rpc.js';

declare const pageLoad: Promise<void>;

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
      shell = shellElement();

class MatchMaker {
	list: Match[];
	contents: HTMLDivElement;
	u = ul();
	w: WindowElement;
	constructor(w: WindowElement, matches: Match[] = []) {
		this.list = matches;
		for (const m of matches) {
			this.add(m);
		}
		if (matches.length === 0) {
			this.add();
		}
		this.contents = div([
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
			this.toSpan
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
	redirects: MapNode<Uint, Redirect>;
	commands: MapNode<Uint, Command>;
	[node]: HTMLLIElement;
	nameDiv: HTMLDivElement;
	constructor([name, rs, cs]: ListItem) {
		this.name = name;
		this.redirects = new MapNode<Uint, Redirect & {[node]: HTMLLIElement}>(ul(), rcSort, rs.map(([id, from, to, active, _, ...match]) => [id, new Redirect(this, id, from, to, active, matchData2Match(match))]));
		this.commands = new MapNode<Uint, Command & {[node]: HTMLLIElement}>(ul(), rcSort, cs.map(([id, exe, params, env, _a, _b, ...match]) => [id, new Command(this, id, exe, params, env, matchData2Match(match))]));
		this.nameDiv = div(name);
		this[node] = li([
			this.nameDiv,
			button({"onclick": () => {
				const from = input({"type": "number", "min": 1, "max": 65535, "value": 80}),
				      to = input(),
				      w = windows(),
				      matches = new MatchMaker(w);
				shell.addWindow(createHTML(w, {"window-title": "Add Redirect"}, [
					label("From:"),
					from,
					br(),
					label("To:"),
					to,
					br(),
					matches.contents,
					button({"onclick": () => {
						const f = parseInt(from.value);
						if (f <= 0 || f >= 65535) {
							shell.alert("Invalid Port", `Invalid from port: ${from.value}`);
						} else if (to.value === "") {
							shell.alert("Invalid address", `Invalid to address: ${to.value}`);
						} else if (matches.list.some(({name}) => name === "")) {
							shell.alert("Invalid Match", "Cannot have empty match");
						} else {
							rpc.addRedirect({
								"server": this.name,
								"from": f,
								"to": to.value,
								"match": matches.list,
							})
							.then(id => this.redirects.set(id, new Redirect(this, id, f, to.value, false, matches.list)))
							.catch(err => shell.alert("Error", err));
							w.remove();
						}
					}}, "Create Redirect")
				]));
			}}, "Add Redirect"),
			button({"onclick": () => {
			}}, "Add Command"),
			this.redirects[node],
			this.commands[node]
		]);
	}
}

pageLoad.then(() => RPC(`ws${window.location.protocol.slice(4)}//${window.location.host}/socket`).then(rpc => {rpc.waitList().then(list => {
	const servers = new MapNode<string, Server>(ul(), (a: Server, b: Server) => stringSort(a.name, b.name), list.map(i => [i[0], new Server(i)])),
	      s = clearElement(document.body).appendChild(createHTML(shell, desktop([
		button({"onclick": () => s.prompt("Server Name", "Please enter a name for the new server", "").then(name => {
			if (name) {
				rpc.add(name).catch(err => s.alert("Error", err)).then(() => servers.set(name, new Server([name, [], []])));
			}
		})}, "New Server"),
		servers[node]
	      ])));
	rpc.waitAdd().then(name => servers.set(name, new Server([name, [], []])));
	rpc.waitAddRedirect().then(r => {
		const server = servers.get(r.server)!
		if (server) {
			server.redirects.set(r.id, new Redirect(server, r.id, r.from, r.to, false, r.match));
		}
	});

})}));
