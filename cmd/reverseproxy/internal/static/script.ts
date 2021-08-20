import type {Uint, Match, MatchData, ListItem} from './types.js';
import {clearElement, createHTML} from './lib/dom.js';
import {br, button, div, input, label, li, span, ul} from './lib/html.js';
import {stringSort, SortNode} from './lib/ordered.js';
import {desktop, shell as shellElement, windows} from './lib/windows.js';
import {MatchMaker} from './match.js';
import RPC, {rpc} from './rpc.js';

declare const pageLoad: Promise<void>;

const rcSort = (a: Redirect | Command, b: Redirect | Command) => a.id - b.id,
      matchData2Match = (md: MatchData[]) => md.map(([isSuffix, name]) => ({isSuffix, name})),
      add2All = <K, T>(id: K, item: T, m: Map<K, T>, list?: T[]) => {
	      m.set(id, item);
	      if (list) {
			list.push(item);
	      }
	      return item;
      },
      noEnum = {"enumerable": false},
      redirectProps = {
	"server": noEnum,
	"node": noEnum,
	"fromSpan": noEnum,
	"toSpan": noEnum
      },
      commandProps = {
	"server": noEnum,
	"node": noEnum,
	"exeSpan": noEnum
      },
      shell = shellElement();

class Redirect {
	id: Uint;
	from: Uint;
	to: string;
	active: boolean;
	match: Match[];
	node: HTMLLIElement;
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
		this.node = li([
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
	node: HTMLLIElement;
	exeSpan: HTMLSpanElement;
	constructor(server: Server, id: Uint, exe: string, params: string[], env: Record<string, string>, match: Match[]) {
		this.id = id;
		this.exe = exe;
		this.params = params;
		this.env = env;
		this.match = match;
		this.exeSpan = span(exe + " " + params.join(" "));
		this.node = li(this.exeSpan);
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
	redirects: SortNode<Redirect>;
	commands: SortNode<Command>;
	redirectMap = new Map<Uint, Redirect>();
	commandMap = new Map<Uint, Command>();
	node: HTMLLIElement;
	nameDiv: HTMLDivElement;
	constructor([name, rs, cs]: ListItem) {
		this.name = name;
		this.redirects = new SortNode<Redirect & {node: HTMLLIElement}>(ul(), rcSort, rs.map(([id, from, to, active, _, ...match]) => add2All(id, new Redirect(this, id, from, to, active, matchData2Match(match)), this.redirectMap)));
		this.commands = new SortNode<Command & {node: HTMLLIElement}>(ul(), rcSort, cs.map(([id, exe, params, env, _a, _b, ...match]) => add2All(id, new Command(this, id, exe, params, env, matchData2Match(match)), this.commandMap)));
		this.nameDiv = div(name);
		this.node = li([
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
							.then(id => this.redirects.push(new Redirect(this, id, f, to.value, false, matches.list)))
							.catch(err => shell.alert("Error", err));
							w.remove();
						}
					}}, "Create Redirect")
				]));
			}}, "Add Redirect"),
			button({"onclick": () => {
			}}, "Add Command"),
			this.redirects.node,
			this.commands.node
		]);
	}
}

pageLoad.then(() => RPC(`ws${window.location.protocol.slice(4)}//${window.location.host}/socket`).then(rpc => {rpc.waitList().then(list => {
	const servers = new Map<string, Server>(),
	      l = new SortNode(ul(), (a: Server, b: Server) => stringSort(a.name, b.name), list.map(i => add2All(i[0], new Server(i), servers))),
	      s = clearElement(document.body).appendChild(createHTML(shell, desktop([
		button({"onclick": () => s.prompt("Server Name", "Please enter a name for the new server", "").then(name => {
			if (name) {
				rpc.add(name).catch(err => s.alert("Error", err)).then(() => add2All(name, new Server([name, [], []]), servers, l));
			}
		})}, "New Server"),
		l.node
	      ])));
	rpc.waitAdd().then(name => add2All(name, new Server([name, [], []]), servers, l));
})}));
