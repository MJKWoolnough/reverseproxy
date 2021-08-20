import type {Match} from './types.js';
import {button, div, input, li, ul} from './lib/html.js';
import {SortNode} from './lib/ordered.js';

type MatchNode = Match & {
	node: HTMLLIElement;
};

export class MatchMaker {
	matches: SortNode<MatchNode>;
	contents: HTMLDivElement;
	constructor(matches: Match[] = []) {
		this.matches = new SortNode<MatchNode>(ul());
		for (const m of matches) {
			this.add(m);
		}
		if (matches.length === 0) {
			this.add();
		}
		this.contents = div([
			"Matches",
			this.matches.node,
			button({"onclick": () => this.add()}, "Add Match")
		]);
	}
	add(m: Match = {"name": "", "isSuffix": false}) {
		this.matches.push(Object.defineProperty(m as MatchNode, "node", {
			"value": li([
				input({"onchange": function(this: HTMLInputElement){m.name = this.value}, "value": m.name}),
				input({"type": "checkbox", "onchange": function(this: HTMLInputElement){m.isSuffix = this.checked}, "checked": m.isSuffix}),
				button({"onclick": () => this.matches.filterRemove(e => Object.is(e, m))}, "X")
			]),
			"enumerable": false
		}));
	}
}
