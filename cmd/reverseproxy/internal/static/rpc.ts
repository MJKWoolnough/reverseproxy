import type {RPC as RPCType, RedirectData, CommandData, NameID} from './types.js';
import RPC from './lib/rpc_ws.js';

const broadcastList = -1, broadcastAdd = -2, broadcastRename = -3, broadcastRemove = -4, broadcastAddRedirect = -5, broadcastAddCommand = -6, broadcastModifyRedirect = -7, broadcastModifyCommand = -8, broadcastRemoveRedirect = -9, broadcastRemoveCommand = -10, broadcastStartRedirect = -11, broadcastStartCommand = -12, broadcastStopRedirect = -13, broadcastStopCommand = -14, broadcastCommandStopped = -15, broadcastCommandError = -16;

export default (url: string): Promise<RPCType> => {
	return RPC(url, 1.1).then(arpc => ({
		waitList:           () => arpc.await(broadcastList, true),
		waitAdd:            () => arpc.await(broadcastAdd, true),
		waitRename:         () => arpc.await(broadcastRename, true),
		waitRemove:         () => arpc.await(broadcastRemove, true),
		waitAddRedirect:    () => arpc.await(broadcastAddRedirect, true),
		waitAddCommand:     () => arpc.await(broadcastAddCommand, true),
		waitModifyRedirect: () => arpc.await(broadcastModifyRedirect, true),
		waitModifyCommand:  () => arpc.await(broadcastModifyCommand, true),
		waitRemoveRedirect: () => arpc.await(broadcastRemoveRedirect, true),
		waitRemoveCommand:  () => arpc.await(broadcastRemoveCommand, true),
		waitStartRedirect:  () => arpc.await(broadcastStartRedirect, true),
		waitStartCommand:   () => arpc.await(broadcastStartCommand, true),
		waitStopRedirect:   () => arpc.await(broadcastStopRedirect, true),
		waitStopCommand:    () => arpc.await(broadcastStopCommand, true),
		waitCommandStopped: () => arpc.await(broadcastCommandStopped, true),
		waitCommandError:   () => arpc.await(broadcastCommandError, true),

		add:            (name: string)                     => arpc.request("add", name),
		rename:         (oldName: string, newName: string) => arpc.request("rename", [oldName, newName]),
		remove:         (name: string)                     => arpc.request("remove", name),
		addRedirect:    (data: RedirectData)               => arpc.request("addRedirect", data),
		addCommand:     (data: CommandData)                => arpc.request("addCommand", data),
		modifyRedirect: (data: RedirectData | NameID)      => arpc.request("modifyRedirect", data),
		modifyCommand:  (data: CommandData | NameID)       => arpc.request("modifyCommand", data),
		removeRedirect: (data: NameID)                     => arpc.request("removeRedirect", data),
		removeCommand:  (data: NameID)                     => arpc.request("removeCommand", data),
		startRedirect:  (data: NameID)                     => arpc.request("startRedirect", data),
		startCommand:   (data: NameID)                     => arpc.request("startCommand", data),
		stopRedirect:   (data: NameID)                     => arpc.request("stopRedirect", data),
		stopCommand:    (data: NameID)                     => arpc.request("stopCommand", data),
	}));
};
