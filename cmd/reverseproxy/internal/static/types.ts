import type {Subscription} from './lib/inter.js';

export type Uint = number;

export type Match = [boolean, string];

type Redirect = [Uint, Uint, string, boolean, string, ...Match[]];

type Command = [Uint, string, string[], Record<string, string>, Uint, string, ...Match[]];

export type ListItem = [string, Redirect[], Command[]]

type List = ListItem[];

export type MatchData = {
	isSuffix: boolean;
	name:     string;
};

export type RedirectData = NameID & {
	from:   Uint;
	to:     string;
	active: boolean;
	match:  MatchData[];
};

export type CommandData = NameID & {
	exe:    string;
	params: string[];
	env:    Record<string, string>;
	match:  MatchData[];
	user?:  {
		uid: Uint;
		gid: Uint;
	};
};

export type NameID = {
	server: string;
	id:     Uint;
}

export type RPC = {
	waitList:           () => Subscription<List>;
	waitAdd:            () => Subscription<string>;
	waitRename:         () => Subscription<[string, string]>;
	waitRemove:         () => Subscription<string>;
	waitAddRedirect:    () => Subscription<RedirectData>;
	waitAddCommand:     () => Subscription<CommandData>;
	waitModifyRedirect: () => Subscription<RedirectData>;
	waitModifyCommand:  () => Subscription<CommandData>;
	waitRemoveRedirect: () => Subscription<NameID>;
	waitRemoveCommand:  () => Subscription<NameID>;
	waitStartRedirect:  () => Subscription<NameID>;
	waitStartCommand:   () => Subscription<NameID>;
	waitStopRedirect:   () => Subscription<NameID>;
	waitStopCommand:    () => Subscription<NameID>;
	waitCommandStopped: () => Subscription<[string, Uint]>;
	waitCommandError:   () => Subscription<NameID & {err: string}>;

	add:             (name: string)                     => Promise<Uint>;
	rename:          (oldName: string, newName: string) => Promise<void>;
	remove:          (name: string)                     => Promise<void>;
	addRedirect:     (data: RedirectData)               => Promise<Uint>;
	addCommand:      (data: CommandData)                => Promise<Uint>;
	modifyRedirect:  (data: RedirectData)               => Promise<void>;
	modifyCommand:   (data: CommandData)                => Promise<void>;
	removeRedirect:  (redirect: NameID)                 => Promise<void>;
	removeCommand:   (command: NameID)                  => Promise<void>;
	startRedirect:   (redirect: NameID)                 => Promise<void>;
	startCommand:    (command: NameID)                  => Promise<void>;
	stopRedirect:    (redirect: NameID)                 => Promise<void>;
	stopCommand:     (command: NameID)                  => Promise<void>;
	getCommandPorts: (command: NameID)                  => Promise<Uint[]>;
};
