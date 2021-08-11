import type {Subscription} from './lib/inter.js';

export type Uint = number;

export type Redirect = [Uint, Uint, string, boolean, string];

export type Command = [Uint, string, string[], Record<string, string>, Uint, string];

export type ListItem = [string, Redirect[], Command[]]

type List = ListItem[];

type Match = {
	isSuffix: boolean;
	name:     string;
};

export type RedirectData = {
	from:  Uint;
	to:    string;
	match: Match[];
};

export type CommandData = {
	exe:    string;
	params: string[];
	env:    Record<string, string>;
	match:  Match[];
	user?:  {
		uid: Uint;
		gid: Uint;
	};
};

export type NameID = {
	Server: string;
	ID:     Uint;
}

export type RPC = {
	waitList:           () => Subscription<List>;
	waitAdd:            () => Subscription<string>;
	waitRename:         () => Subscription<[string, string]>;
	waitRemove:         () => Subscription<string>;
	waitAddRedirect:    () => Subscription<RedirectData | NameID>;
	waitAddCommand:     () => Subscription<CommandData | NameID>;
	waitModifyRedirect: () => Subscription<RedirectData | NameID>;
	waitModifyCommand:  () => Subscription<CommandData | NameID>;
	waitRemoveRedirect: () => Subscription<NameID>;
	waitRemoveCommand:  () => Subscription<NameID>;
	waitStartRedirect:  () => Subscription<NameID>;
	waitStartCommand:   () => Subscription<NameID>;
	waitStopRedirect:   () => Subscription<NameID>;
	waitStopCommand:    () => Subscription<NameID>;
	waitCommandStopped: () => Subscription<[string, Uint]>;
	waitCommandError:   () => Subscription<NameID | {err: string}>;

	add:            (name: string)                     => Promise<Uint>;
	rename:         (oldName: string, newName: string) => Promise<void>;
	remove:         (name: string)                     => Promise<void>;
	addRedirect:    (data: RedirectData)               => Promise<Uint>;
	addCommand:     (data: CommandData)                => Promise<Uint>;
	modifyRedirect: (data: RedirectData | NameID)      => Promise<void>;
	modifyCommand:  (data: CommandData | NameID)       => Promise<void>;
	removeRedirect: (redirect: NameID)                 => Promise<void>;
	removeCommand:  (command: NameID)                  => Promise<void>;
	startRedirect:  (redirect: NameID)                 => Promise<void>;
	startCommand:   (command: NameID)                  => Promise<void>;
	stopRedirect:   (redirect: NameID)                 => Promise<void>;
	stopCommand:    (command: NameID)                  => Promise<void>;
};
