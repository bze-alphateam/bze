import { Client, registry, MissingWalletError } from 'bze-alphateam-bze-client-ts'

import { BurnCoinsProposal } from "bze-alphateam-bze-client-ts/bze.burner.v1/types"
import { BurnedCoins } from "bze-alphateam-bze-client-ts/bze.burner.v1/types"
import { CoinsBurnedEvent } from "bze-alphateam-bze-client-ts/bze.burner.v1/types"
import { FundBurnerEvent } from "bze-alphateam-bze-client-ts/bze.burner.v1/types"
import { Params } from "bze-alphateam-bze-client-ts/bze.burner.v1/types"


export { BurnCoinsProposal, BurnedCoins, CoinsBurnedEvent, FundBurnerEvent, Params };

function initClient(vuexGetters) {
	return new Client(vuexGetters['common/env/getEnv'], vuexGetters['common/wallet/signer'])
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

type Field = {
	name: string;
	type: unknown;
}
function getStructure(template) {
	let structure: {fields: Field[]} = { fields: [] }
	for (const [key, value] of Object.entries(template)) {
		let field = { name: key, type: typeof value }
		structure.fields.push(field)
	}
	return structure
}
const getDefaultState = () => {
	return {
				Params: {},
				AllBurnedCoins: {},
				
				_Structure: {
						BurnCoinsProposal: getStructure(BurnCoinsProposal.fromPartial({})),
						BurnedCoins: getStructure(BurnedCoins.fromPartial({})),
						CoinsBurnedEvent: getStructure(CoinsBurnedEvent.fromPartial({})),
						FundBurnerEvent: getStructure(FundBurnerEvent.fromPartial({})),
						Params: getStructure(Params.fromPartial({})),
						
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
				const client = initClient(rootGetters);
				let value= (await client.BzeBurnerV1.query.queryParams()).data
				
					
				commit('QUERY', { query: 'Params', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryParams', payload: { options: { all }, params: {...key},query }})
				return getters['getParams']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryParams API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryAllBurnedCoins({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeBurnerV1.query.queryAllBurnedCoins(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeBurnerV1.query.queryAllBurnedCoins({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'AllBurnedCoins', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryAllBurnedCoins', payload: { options: { all }, params: {...key},query }})
				return getters['getAllBurnedCoins']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryAllBurnedCoins API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		async sendMsgFundBurner({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeBurnerV1.tx.sendMsgFundBurner({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgFundBurner:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgFundBurner:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		
		async MsgFundBurner({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeBurnerV1.tx.msgFundBurner({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgFundBurner:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgFundBurner:Create Could not create message: ' + e.message)
				}
			}
		},
		
	}
}
