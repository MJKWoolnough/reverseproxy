import type {Match} from './types.js';
import type {WindowElement} from './lib/windows.js';
import {button, div, input, li, ul} from './lib/html.js';

export class MatchMaker {
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
