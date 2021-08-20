import type {Match} from './types.js';
import type {WindowElement} from './lib/windows.js';
import {button, div, input, li, ul} from './lib/html.js';
import {SortNode} from './lib/ordered.js';

type MatchNode = Match & {
	node: HTMLLIElement;
};

export class MatchMaker {
	matches: SortNode<MatchNode>;
	contents: HTMLDivElement;
	w: WindowElement;
	constructor(w: WindowElement, matches: Match[] = []) {
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
		this.w = w;
	}
	add(m: Match = {"name": "", "isSuffix": false}) {
		this.matches.push(Object.defineProperty(m as MatchNode, "node", {
			"value": li([
				input({"onchange": function(this: HTMLInputElement){m.name = this.value}, "value": m.name}),
				input({"type": "checkbox", "onchange": function(this: HTMLInputElement){m.isSuffix = this.checked}, "checked": m.isSuffix}),
				button({"onclick": () => {
					if (this.matches.length === 1) {
						this.w.alert("Cannot remove Match", "Must have at least 1 Match");
					} else {
						this.matches.filterRemove(e => Object.is(e, m));
					}
				}}, "X")
			]),
			"enumerable": false
		}));
	}
}
