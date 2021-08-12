import type {Uint, Command, Match, Redirect, ListItem} from './types.js';
import {createHTML, clearElement} from './lib/dom.js';
import {button, div, li, span, ul} from './lib/html.js';
import {stringSort, SortNode} from './lib/ordered.js';
import RPC from './rpc.js';

declare const pageLoad: Promise<void>;

type Server = {
	name: string;
	node: HTMLLIElement;
	redirects: SortNode<Redirect & {node: HTMLLIElement}>;
	commands: SortNode<Command & {node: HTMLLIElement}>;
	redirectMap: Map<Uint, Redirect>;
	commandMap: Map<Uint, Command>;
};

pageLoad.then(() => RPC(`ws${window.location.protocol.slice(4)}//${window.location.host}/socket`).then(rpc => {rpc.waitList().then(list => {
	const l = new SortNode(ul(), (a: Server, b: Server) => stringSort(a.name, b.name)),
	      rcSort = (a: Redirect | Command, b: Redirect | Command) => a.id - b.id,
	      addToList = ([name, rs = [], cs = []]: ListItem) => {
		const nameDiv = div(name),
		      redirects = ul(),
		      commands = ul(),
		      redirectMap = new Map<Uint, Redirect>(),
		      commandMap = new Map<Uint, Command>(),
		      server: Server = {
			get name(){return name},
			set name(n: string){nameDiv.innerText = name = n},
			node: li([
				nameDiv,
				button({"onclick": () => {

				}}, "+"),
				redirects,
				commands
			]),
			redirects: new SortNode<Redirect & {node: HTMLLIElement}>(redirects, rcSort, rs.map(([id, from, to, active, _, ...match]) => {
				const fromSpan = span(from + ""),
				      toSpan = span(to),
				      r = {
					get server() {return name},
					id,
					get from() {return from},
					set from(f: Uint) {fromSpan.innerText = (from = f) + ""},
					get to() {return to},
					set to(t: string) {toSpan.innerText = to = t},
					get active() {return active},
					set active(a: boolean) {active = a},
					match: (match as Match[]).map(([isSuffix, name]) => ({isSuffix, name})),
					node: li([
						fromSpan,
						toSpan
					])
				};
				redirectMap.set(id, r);
				return r;
			})),
			commands: new SortNode<Command & {node: HTMLLIElement}>(commands, rcSort, cs.map(([id, exe, params, env, ...match]) => {
				const c = {
					get server() {return server.name;},
					id,
					exe,
					params,
					env,
					match: (match as Match[]).map(([isSuffix, name]) => ({isSuffix, name})),
					node: li()
				};
				commandMap.set(id, c);
				return c;
			})),
			redirectMap,
			commandMap
		      };
		l.push(server);
		servers.set(name, server);
	      },
	      servers = new Map<string, Server>();
	list.forEach(addToList);
	createHTML(clearElement(document.body), [
		button({"onclick": () => {

		}}, "New Server"),
		l.node
	]);
})}));
