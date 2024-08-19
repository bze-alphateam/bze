import { txClient, queryClient, MissingWalletError , registry} from './module'

import { BurnCoinsProposal } from "./module/types/burner/burn_coins_proposal"
import { BurnedCoins } from "./module/types/burner/burned_coins"
import { CoinsBurnedEvent } from "./module/types/burner/events"
import { FundBurnerEvent } from "./module/types/burner/events"
import { RaffleWinnerEvent } from "./module/types/burner/events"
import { RaffleLostEvent } from "./module/types/burner/events"
import { RaffleFinishedEvent } from "./module/types/burner/events"
import { Params } from "./module/types/burner/params"
import { Raffle } from "./module/types/burner/raffle"
import { RaffleDeleteHook } from "./module/types/burner/raffle"
import { RaffleWinner } from "./module/types/burner/raffle"
import { RaffleParticipant } from "./module/types/burner/raffle"


export { BurnCoinsProposal, BurnedCoins, CoinsBurnedEvent, FundBurnerEvent, RaffleWinnerEvent, RaffleLostEvent, RaffleFinishedEvent, Params, Raffle, RaffleDeleteHook, RaffleWinner, RaffleParticipant };

async function initTxClient(vuexGetters) {
	return await txClient(vuexGetters['common/wallet/signer'], {
		addr: vuexGetters['common/env/apiTendermint']
	})
}

async function initQueryClient(vuexGetters) {
	return await queryClient({
		addr: vuexGetters['common/env/apiCosmos']
	})
}

function mergeResults(value, next_values) {
	for (let prop of Object.keys(next_values)) {
		if (Array.isArray(next_values[prop])) {
			value[prop]=[...value[prop], ...next_values[prop]]
		}else{
			value[prop]=next_values[prop]
		}
	}
	return value
}

function getStructure(template) {
	let structure = { fields: [] }
	for (const [key, value] of Object.entries(template)) {
		let field: any = {}
		field.name = key
		field.type = typeof value
		structure.fields.push(field)
	}
	return structure
}

const getDefaultState = () => {
	return {
				Params: {},
				Raffles: {},
				RaffleWinners: {},
				AllBurnedCoins: {},
				
				_Structure: {
						BurnCoinsProposal: getStructure(BurnCoinsProposal.fromPartial({})),
						BurnedCoins: getStructure(BurnedCoins.fromPartial({})),
						CoinsBurnedEvent: getStructure(CoinsBurnedEvent.fromPartial({})),
						FundBurnerEvent: getStructure(FundBurnerEvent.fromPartial({})),
						RaffleWinnerEvent: getStructure(RaffleWinnerEvent.fromPartial({})),
						RaffleLostEvent: getStructure(RaffleLostEvent.fromPartial({})),
						RaffleFinishedEvent: getStructure(RaffleFinishedEvent.fromPartial({})),
						Params: getStructure(Params.fromPartial({})),
						Raffle: getStructure(Raffle.fromPartial({})),
						RaffleDeleteHook: getStructure(RaffleDeleteHook.fromPartial({})),
						RaffleWinner: getStructure(RaffleWinner.fromPartial({})),
						RaffleParticipant: getStructure(RaffleParticipant.fromPartial({})),
						
		},
		_Registry: registry,
		_Subscriptions: new Set(),
	}
}

// initial state
const state = getDefaultState()

export default {
	namespaced: true,
	state,
	mutations: {
		RESET_STATE(state) {
			Object.assign(state, getDefaultState())
		},
		QUERY(state, { query, key, value }) {
			state[query][JSON.stringify(key)] = value
		},
		SUBSCRIBE(state, subscription) {
			state._Subscriptions.add(JSON.stringify(subscription))
		},
		UNSUBSCRIBE(state, subscription) {
			state._Subscriptions.delete(JSON.stringify(subscription))
		}
	},
	getters: {
				getParams: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.Params[JSON.stringify(params)] ?? {}
		},
				getRaffles: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.Raffles[JSON.stringify(params)] ?? {}
		},
				getRaffleWinners: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.RaffleWinners[JSON.stringify(params)] ?? {}
		},
				getAllBurnedCoins: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.AllBurnedCoins[JSON.stringify(params)] ?? {}
		},
				
		getTypeStructure: (state) => (type) => {
			return state._Structure[type].fields
		},
		getRegistry: (state) => {
			return state._Registry
		}
	},
	actions: {
		init({ dispatch, rootGetters }) {
			console.log('Vuex module: bze.burner.v1 initialized!')
			if (rootGetters['common/env/client']) {
				rootGetters['common/env/client'].on('newblock', () => {
					dispatch('StoreUpdate')
				})
			}
		},
		resetState({ commit }) {
			commit('RESET_STATE')
		},
		unsubscribe({ commit }, subscription) {
			commit('UNSUBSCRIBE', subscription)
		},
		async StoreUpdate({ state, dispatch }) {
			state._Subscriptions.forEach(async (subscription) => {
				try {
					const sub=JSON.parse(subscription)
					await dispatch(sub.action, sub.payload)
				}catch(e) {
					throw new Error('Subscriptions: ' + e.message)
				}
			})
		},
		
		
		
		 		
		
		
		async QueryParams({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.queryParams()).data
				
					
				commit('QUERY', { query: 'Params', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryParams', payload: { options: { all }, params: {...key},query }})
				return getters['getParams']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryParams API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryRaffles({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.queryRaffles(query)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await queryClient.queryRaffles({...query, 'pagination.key':(<any> value).pagination.next_key})).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'Raffles', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryRaffles', payload: { options: { all }, params: {...key},query }})
				return getters['getRaffles']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryRaffles API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryRaffleWinners({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.queryRaffleWinners(query)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await queryClient.queryRaffleWinners({...query, 'pagination.key':(<any> value).pagination.next_key})).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'RaffleWinners', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryRaffleWinners', payload: { options: { all }, params: {...key},query }})
				return getters['getRaffleWinners']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryRaffleWinners API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryAllBurnedCoins({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.queryAllBurnedCoins(query)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await queryClient.queryAllBurnedCoins({...query, 'pagination.key':(<any> value).pagination.next_key})).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'AllBurnedCoins', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryAllBurnedCoins', payload: { options: { all }, params: {...key},query }})
				return getters['getAllBurnedCoins']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryAllBurnedCoins API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		async sendMsgStartRaffle({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const txClient=await initTxClient(rootGetters)
				const msg = await txClient.msgStartRaffle(value)
				const result = await txClient.signAndBroadcast([msg], {fee: { amount: fee, 
	gas: "200000" }, memo})
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgStartRaffle:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgStartRaffle:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		async sendMsgFundBurner({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const txClient=await initTxClient(rootGetters)
				const msg = await txClient.msgFundBurner(value)
				const result = await txClient.signAndBroadcast([msg], {fee: { amount: fee, 
	gas: "200000" }, memo})
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgFundBurner:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgFundBurner:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		async sendMsgJoinRaffle({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const txClient=await initTxClient(rootGetters)
				const msg = await txClient.msgJoinRaffle(value)
				const result = await txClient.signAndBroadcast([msg], {fee: { amount: fee, 
	gas: "200000" }, memo})
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgJoinRaffle:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgJoinRaffle:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		
		async MsgStartRaffle({ rootGetters }, { value }) {
			try {
				const txClient=await initTxClient(rootGetters)
				const msg = await txClient.msgStartRaffle(value)
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgStartRaffle:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgStartRaffle:Create Could not create message: ' + e.message)
				}
			}
		},
		async MsgFundBurner({ rootGetters }, { value }) {
			try {
				const txClient=await initTxClient(rootGetters)
				const msg = await txClient.msgFundBurner(value)
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgFundBurner:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgFundBurner:Create Could not create message: ' + e.message)
				}
			}
		},
		async MsgJoinRaffle({ rootGetters }, { value }) {
			try {
				const txClient=await initTxClient(rootGetters)
				const msg = await txClient.msgJoinRaffle(value)
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgJoinRaffle:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgJoinRaffle:Create Could not create message: ' + e.message)
				}
			}
		},
		
	}
}
