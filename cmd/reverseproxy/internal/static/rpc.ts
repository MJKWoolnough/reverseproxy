import type {RPC as RPCType} from './types.js';
import RPC from './lib/rpc_ws.js';

const broadcastList = -1, broadcastAdd = -2, broadcastRename = -3, broadcastRemove = -4, broadcastAddRedirect = -5, broadcastAddCommand = -6, broadcastModifyRedirect = -7, broadcastModifyCommand = -8, broadcastRemoveRedirect = -9, broadcastRemoveCommand = -10, broadcastStartRedirect = -11, broadcastStartCommand = -12, broadcastStopRedirect = -13, broadcastStopCommand = -14, broadcastCommandStopped = -15, broadcastCommandError = -16;

export let rpc: Readonly<RPCType>;

export default (url: string): Promise<RPCType> => {
	return RPC(url, 1.1).then(arpc => (rpc = Object.freeze(Object.fromEntries([
		([
			["waitList",           broadcastList],
			["waitAdd",            broadcastAdd],
			["waitRename",         broadcastRename],
			["waitRemove",         broadcastRemove],
			["waitAddRedirect",    broadcastAddRedirect],
			["waitAddCommand",     broadcastAddCommand],
			["waitModifyRedirect", broadcastModifyRedirect],
			["waitModifyCommand",  broadcastModifyCommand],
			["waitRemoveRedirect", broadcastRemoveRedirect],
			["waitRemoveCommand",  broadcastRemoveCommand],
			["waitStartRedirect",  broadcastStartRedirect],
			["waitStartCommand",   broadcastStartCommand],
			["waitStopRedirect",   broadcastStopRedirect],
			["waitStopCommand",    broadcastStopCommand],
			["waitCommandStopped", broadcastCommandStopped],
			["waitCommandError",   broadcastCommandError]
		] as [string, number][]).map(([wait, id]) => [wait, () => arpc.await(id, true)]),
		[
			"add",
			"rename",
			"remove",
			"addRedirect",
			"addCommand",
			"modifyRedirect",
			"modifyCommand",
			"removeRedirect",
			"removeCommand",
			"startRedirect",
			"startCommand",
			"stopRedirect",
			"stopCommand",
			"getCommandPorts"
		].map(ep => [ep, arpc.request.bind(ep)])
	].flat()) as RPCType)));
};
