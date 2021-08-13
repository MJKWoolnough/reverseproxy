import type {Uint, Match, MatchData, ListItem} from './types.js';
import {createHTML, clearElement} from './lib/dom.js';
import {button, div, li, span, ul} from './lib/html.js';
import {stringSort, SortNode} from './lib/ordered.js';
import RPC from './rpc.js';

declare const pageLoad: Promise<void>;


const rcSort = (a: Redirect | Command, b: Redirect | Command) => a.id - b.id,
      matchData2Match = (md: MatchData[]) => md.map(([isSuffix, name]) => ({isSuffix, name})),
      add2Map = <K, T>(m: Map<K, T>, id: K, item: T) => {
	      m.set(id, item);
	      return item;
      };

class Redirect {
	server: Server;
	id: Uint;
	from: Uint;
	to: string;
	active: boolean;
	match: Match[];
	node: HTMLLIElement;
	fromSpan: HTMLSpanElement;
	toSpan: HTMLSpanElement;
	constructor(server: Server, id: Uint, from: Uint, to: string, active: boolean, match: Match[]) {
		this.server = server;
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
	}
	setFrom(f: Uint) {
		this.fromSpan.innerText = (this.from = f) + "";
	}
	setTo(t: string) {
		this.toSpan.innerText = this.to = t;
	}
}

class Command {
	server: Server;
	id: Uint;
	exe: string;
	params: string[];
	env: Record<string, string>;
	match: Match[];
	node: HTMLLIElement;
	exeSpan: HTMLSpanElement;
	constructor(server: Server, id: Uint, exe: string, params: string[], env: Record<string, string>, match: Match[]) {
		this.server = server;
		this.id = id;
		this.exe = exe;
		this.params = params;
		this.env = env;
		this.match = match;
		this.exeSpan = span(exe + " " + params.join(" "));
		this.node = li(this.exeSpan);
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
	constructor([name, rs = [], cs = []]: ListItem) {
		this.name = name;
		this.redirects = new SortNode<Redirect & {node: HTMLLIElement}>(ul(), rcSort, rs.map(([id, from, to, active, _, ...match]) => add2Map(this.redirectMap, id, new Redirect(this, id, from, to, active, matchData2Match(match)))));
		this.commands = new SortNode<Command & {node: HTMLLIElement}>(ul(), rcSort, cs.map(([id, exe, params, env, _a, _b, ...match]) => add2Map(this.commandMap, id, new Command(this, id, exe, params, env, matchData2Match(match)))));
		this.nameDiv = div(name);
		this.node = li([
			this.nameDiv,
			button({"onclick": () => {

			}}, "+"),
			this.redirects.node,
			this.commands.node
		]);
	}
}

pageLoad.then(() => RPC(`ws${window.location.protocol.slice(4)}//${window.location.host}/socket`).then(rpc => {rpc.waitList().then(list => {
	const servers = new Map<string, Server>(),
	      l = new SortNode(ul(), (a: Server, b: Server) => stringSort(a.name, b.name), list.map(i => add2Map(servers, i[0], new Server(i))));
	createHTML(clearElement(document.body), [
		button({"onclick": () => {

		}}, "New Server"),
		l.node
	]);
})}));
