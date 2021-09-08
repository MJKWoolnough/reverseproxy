import type {DOMBind} from './dom.js';
import {createSVG} from './dom.js';

export {createSVG};

export const [circle, g, line, path, polyline, rect, svg, symbol, title, use] = "circle g line path polyline rect svg symbol title use".split(" ").map(e => createSVG.bind(null, e)) as [DOMBind<SVGElementTagNameMap["circle"]>,DOMBind<SVGElementTagNameMap["g"]>,DOMBind<SVGElementTagNameMap["line"]>,DOMBind<SVGElementTagNameMap["path"]>,DOMBind<SVGElementTagNameMap["polyline"]>,DOMBind<SVGElementTagNameMap["rect"]>,DOMBind<SVGElementTagNameMap["svg"]>,DOMBind<SVGElementTagNameMap["symbol"]>,DOMBind<SVGElementTagNameMap["title"]>,DOMBind<SVGElementTagNameMap["use"]>];
