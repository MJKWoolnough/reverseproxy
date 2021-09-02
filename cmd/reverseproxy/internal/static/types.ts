import type {Subscription} from './lib/inter.js';

export type Uint = number;

export type MatchData = [boolean, string];

export type ListItem = [string, [Uint, Uint, string, boolean, string, ...MatchData[]][], [Uint, string, string[], Record<string, string>, Uint, string, ...MatchData[]][]]

type List = ListItem[];

export type Match = {
	isSuffix: boolean;
	name:     string;
};

type Redirect = NameID & {
	from:   Uint;
	to:     string;
	match:  Match[];
};

export type UserID = {
	uid: Uint;
	gid: Uint;
}

type Command = NameID & {
	exe:    string;
	params: string[];
	env:    Record<string, string>;
	match:  Match[];
	user?:  UserID;
};

type NameID = {
	server: string;
	id:     Uint;
}

export type RPC = {
	waitList:           () => Subscription<List>;
	waitAdd:            () => Subscription<string>;
	waitRename:         () => Subscription<[string, string]>;
	waitRemove:         () => Subscription<string>;
	waitAddRedirect:    () => Subscription<Redirect>;
	waitAddCommand:     () => Subscription<Command>;
	waitModifyRedirect: () => Subscription<Redirect>;
	waitModifyCommand:  () => Subscription<Command>;
	waitRemoveRedirect: () => Subscription<NameID>;
	waitRemoveCommand:  () => Subscription<NameID>;
	waitStartRedirect:  () => Subscription<NameID>;
	waitStartCommand:   () => Subscription<NameID>;
	waitStopRedirect:   () => Subscription<NameID>;
	waitStopCommand:    () => Subscription<NameID>;
	waitCommandStopped: () => Subscription<[string, Uint]>;
	waitCommandError:   () => Subscription<NameID & {err: string}>;

	add:             (name: string)                          => Promise<void>;
	rename:          (data: [string, string])                => Promise<void>;
	remove:          (name: string)                          => Promise<void>;
	addRedirect:     (data: Omit<Redirect, "id">)            => Promise<Uint>;
	addCommand:      (data: Omit<Command, "id">)             => Promise<Uint>;
	modifyRedirect:  (data: Redirect)                        => Promise<void>;
	modifyCommand:   (data: Command)                         => Promise<void>;
	removeRedirect:  (redirect: NameID)                      => Promise<void>;
	removeCommand:   (command: NameID)                       => Promise<void>;
	startRedirect:   (redirect: NameID)                      => Promise<void>;
	startCommand:    (command: NameID)                       => Promise<void>;
	stopRedirect:    (redirect: NameID)                      => Promise<void>;
	stopCommand:     (command: NameID)                       => Promise<void>;
	getCommandPorts: (command: NameID)                       => Promise<Uint[]>;
};
