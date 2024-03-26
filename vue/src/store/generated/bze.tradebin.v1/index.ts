import { Client, registry, MissingWalletError } from 'bze-alphateam-bze-client-ts'

import { OrderCreateMessageEvent } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { OrderCancelMessageEvent } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { MarketCreatedEvent } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { OrderExecutedEvent } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { OrderCanceledEvent } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { OrderSavedEvent } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { Market } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { Order } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { OrderReference } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { AggregatedOrder } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { HistoryOrder } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { Params } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"
import { QueueMessage } from "bze-alphateam-bze-client-ts/bze.tradebin.v1/types"


export { OrderCreateMessageEvent, OrderCancelMessageEvent, MarketCreatedEvent, OrderExecutedEvent, OrderCanceledEvent, OrderSavedEvent, Market, Order, OrderReference, AggregatedOrder, HistoryOrder, Params, QueueMessage };

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
				Market: {},
				MarketAll: {},
				AssetMarkets: {},
				UserMarketOrders: {},
				MarketAggregatedOrders: {},
				MarketHistory: {},
				MarketOrder: {},
				
				_Structure: {
						OrderCreateMessageEvent: getStructure(OrderCreateMessageEvent.fromPartial({})),
						OrderCancelMessageEvent: getStructure(OrderCancelMessageEvent.fromPartial({})),
						MarketCreatedEvent: getStructure(MarketCreatedEvent.fromPartial({})),
						OrderExecutedEvent: getStructure(OrderExecutedEvent.fromPartial({})),
						OrderCanceledEvent: getStructure(OrderCanceledEvent.fromPartial({})),
						OrderSavedEvent: getStructure(OrderSavedEvent.fromPartial({})),
						Market: getStructure(Market.fromPartial({})),
						Order: getStructure(Order.fromPartial({})),
						OrderReference: getStructure(OrderReference.fromPartial({})),
						AggregatedOrder: getStructure(AggregatedOrder.fromPartial({})),
						HistoryOrder: getStructure(HistoryOrder.fromPartial({})),
						Params: getStructure(Params.fromPartial({})),
						QueueMessage: getStructure(QueueMessage.fromPartial({})),
						
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
				getMarket: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.Market[JSON.stringify(params)] ?? {}
		},
				getMarketAll: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.MarketAll[JSON.stringify(params)] ?? {}
		},
				getAssetMarkets: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.AssetMarkets[JSON.stringify(params)] ?? {}
		},
				getUserMarketOrders: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.UserMarketOrders[JSON.stringify(params)] ?? {}
		},
				getMarketAggregatedOrders: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.MarketAggregatedOrders[JSON.stringify(params)] ?? {}
		},
				getMarketHistory: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.MarketHistory[JSON.stringify(params)] ?? {}
		},
				getMarketOrder: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.MarketOrder[JSON.stringify(params)] ?? {}
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
			console.log('Vuex module: bze.tradebin.v1 initialized!')
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
				let value= (await client.BzeTradebinV1.query.queryParams()).data
				
					
				commit('QUERY', { query: 'Params', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryParams', payload: { options: { all }, params: {...key},query }})
				return getters['getParams']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryParams API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryMarket({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeTradebinV1.query.queryMarket(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeTradebinV1.query.queryMarket({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'Market', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryMarket', payload: { options: { all }, params: {...key},query }})
				return getters['getMarket']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryMarket API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryMarketAll({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeTradebinV1.query.queryMarketAll(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeTradebinV1.query.queryMarketAll({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'MarketAll', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryMarketAll', payload: { options: { all }, params: {...key},query }})
				return getters['getMarketAll']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryMarketAll API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryAssetMarkets({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeTradebinV1.query.queryAssetMarkets(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeTradebinV1.query.queryAssetMarkets({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'AssetMarkets', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryAssetMarkets', payload: { options: { all }, params: {...key},query }})
				return getters['getAssetMarkets']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryAssetMarkets API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryUserMarketOrders({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeTradebinV1.query.queryUserMarketOrders( key.address, query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeTradebinV1.query.queryUserMarketOrders( key.address, {...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'UserMarketOrders', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryUserMarketOrders', payload: { options: { all }, params: {...key},query }})
				return getters['getUserMarketOrders']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryUserMarketOrders API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryMarketAggregatedOrders({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeTradebinV1.query.queryMarketAggregatedOrders(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeTradebinV1.query.queryMarketAggregatedOrders({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'MarketAggregatedOrders', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryMarketAggregatedOrders', payload: { options: { all }, params: {...key},query }})
				return getters['getMarketAggregatedOrders']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryMarketAggregatedOrders API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryMarketHistory({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeTradebinV1.query.queryMarketHistory(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeTradebinV1.query.queryMarketHistory({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'MarketHistory', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryMarketHistory', payload: { options: { all }, params: {...key},query }})
				return getters['getMarketHistory']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryMarketHistory API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryMarketOrder({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeTradebinV1.query.queryMarketOrder(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeTradebinV1.query.queryMarketOrder({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'MarketOrder', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryMarketOrder', payload: { options: { all }, params: {...key},query }})
				return getters['getMarketOrder']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryMarketOrder API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		async sendMsgCreateMarket({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeTradebinV1.tx.sendMsgCreateMarket({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgCreateMarket:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgCreateMarket:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		async sendMsgCreateOrder({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeTradebinV1.tx.sendMsgCreateOrder({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgCreateOrder:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgCreateOrder:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		async sendMsgCancelOrder({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeTradebinV1.tx.sendMsgCancelOrder({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgCancelOrder:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgCancelOrder:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		
		async MsgCreateMarket({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeTradebinV1.tx.msgCreateMarket({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgCreateMarket:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgCreateMarket:Create Could not create message: ' + e.message)
				}
			}
		},
		async MsgCreateOrder({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeTradebinV1.tx.msgCreateOrder({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgCreateOrder:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgCreateOrder:Create Could not create message: ' + e.message)
				}
			}
		},
		async MsgCancelOrder({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeTradebinV1.tx.msgCancelOrder({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgCancelOrder:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgCancelOrder:Create Could not create message: ' + e.message)
				}
			}
		},
		
	}
}
