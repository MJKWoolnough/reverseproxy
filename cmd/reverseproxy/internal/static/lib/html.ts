import type {DOMBind} from './dom.js';
import {createHTML} from './dom.js';

export {createHTML};

export const [br, button, div, h1, img, input, label, li, slot, span, style, ul] = "br button div h1 img input label li slot span style ul".split(" ").map(e => createHTML.bind(null, e)) as [DOMBind<HTMLElementTagNameMap["br"]>, DOMBind<HTMLElementTagNameMap["button"]>, DOMBind<HTMLElementTagNameMap["div"]>, DOMBind<HTMLElementTagNameMap["h1"]>, DOMBind<HTMLElementTagNameMap["img"]>, DOMBind<HTMLElementTagNameMap["input"]>, DOMBind<HTMLElementTagNameMap["label"]>, DOMBind<HTMLElementTagNameMap["li"]>, DOMBind<HTMLElementTagNameMap["slot"]>, DOMBind<HTMLElementTagNameMap["span"]>, DOMBind<HTMLElementTagNameMap["style"]>, DOMBind<HTMLElementTagNameMap["ul"]>];
