import type {Uint, Command, Match, Redirect, ListItem} from './types.js';
import {createHTML, clearElement} from './lib/dom.js';
import {button, ul, li} from './lib/html.js';
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
	      addToList = ([name, rs = [], cs = []]: ListItem) => {
		const server: Server = {
			name,
			node: li(),
			redirects: new SortNode<Redirect & {node: HTMLLIElement}>(ul()),
			commands: new SortNode<Command & {node: HTMLLIElement}>(ul()),
			redirectMap: new Map<Uint, Redirect>(rs.map(([id, from, to, active, ...match]) => ([id, {
				get server() {return server.name;},
				id,
				from,
				to,
				active,
				match: (match as Match[]).map(([isSuffix, name]) => ({isSuffix, name}))
			}]))),
			commandMap: new Map<Uint, Command>(cs.map(([id, exe, params, env, ...match]) => ([id, {
				get server() {return server.name;},
				id,
				exe,
				params,
				env,
				match: (match as Match[]).map(([isSuffix, name]) => ({isSuffix, name}))
			}]))),
		      };
		for (const [, redirect] of server.redirectMap) {
			server.redirects.push(Object.assign(redirect, {node: li()}));
		}
		for (const [, command] of server.commandMap) {
			server.commands.push(Object.assign(command, {node: li()}));
		}
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
