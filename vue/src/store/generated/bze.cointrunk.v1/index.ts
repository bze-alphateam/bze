import { Client, registry, MissingWalletError } from 'bze-alphateam-bze-client-ts'

import { AcceptedDomain } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { AcceptedDomainProposal } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { AnonArticlesCounter } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { Article } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { ArticleAddedEvent } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { PublisherAddedEvent } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { PublisherUpdatedEvent } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { AcceptedDomainAddedEvent } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { AcceptedDomainUpdatedEvent } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { PublisherRespectPaidEvent } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { PublisherRespectParams } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { Params } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { Publisher } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"
import { PublisherProposal } from "bze-alphateam-bze-client-ts/bze.cointrunk.v1/types"


export { AcceptedDomain, AcceptedDomainProposal, AnonArticlesCounter, Article, ArticleAddedEvent, PublisherAddedEvent, PublisherUpdatedEvent, AcceptedDomainAddedEvent, AcceptedDomainUpdatedEvent, PublisherRespectPaidEvent, PublisherRespectParams, Params, Publisher, PublisherProposal };

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
				AcceptedDomain: {},
				Publisher: {},
				PublisherByIndex: {},
				AllArticles: {},
				AllAnonArticlesCounters: {},
				
				_Structure: {
						AcceptedDomain: getStructure(AcceptedDomain.fromPartial({})),
						AcceptedDomainProposal: getStructure(AcceptedDomainProposal.fromPartial({})),
						AnonArticlesCounter: getStructure(AnonArticlesCounter.fromPartial({})),
						Article: getStructure(Article.fromPartial({})),
						ArticleAddedEvent: getStructure(ArticleAddedEvent.fromPartial({})),
						PublisherAddedEvent: getStructure(PublisherAddedEvent.fromPartial({})),
						PublisherUpdatedEvent: getStructure(PublisherUpdatedEvent.fromPartial({})),
						AcceptedDomainAddedEvent: getStructure(AcceptedDomainAddedEvent.fromPartial({})),
						AcceptedDomainUpdatedEvent: getStructure(AcceptedDomainUpdatedEvent.fromPartial({})),
						PublisherRespectPaidEvent: getStructure(PublisherRespectPaidEvent.fromPartial({})),
						PublisherRespectParams: getStructure(PublisherRespectParams.fromPartial({})),
						Params: getStructure(Params.fromPartial({})),
						Publisher: getStructure(Publisher.fromPartial({})),
						PublisherProposal: getStructure(PublisherProposal.fromPartial({})),
						
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
				getAcceptedDomain: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.AcceptedDomain[JSON.stringify(params)] ?? {}
		},
				getPublisher: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.Publisher[JSON.stringify(params)] ?? {}
		},
				getPublisherByIndex: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.PublisherByIndex[JSON.stringify(params)] ?? {}
		},
				getAllArticles: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.AllArticles[JSON.stringify(params)] ?? {}
		},
				getAllAnonArticlesCounters: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.AllAnonArticlesCounters[JSON.stringify(params)] ?? {}
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
			console.log('Vuex module: bze.cointrunk.v1 initialized!')
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
				let value= (await client.BzeCointrunkV1.query.queryParams()).data
				
					
				commit('QUERY', { query: 'Params', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryParams', payload: { options: { all }, params: {...key},query }})
				return getters['getParams']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryParams API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryAcceptedDomain({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeCointrunkV1.query.queryAcceptedDomain(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeCointrunkV1.query.queryAcceptedDomain({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'AcceptedDomain', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryAcceptedDomain', payload: { options: { all }, params: {...key},query }})
				return getters['getAcceptedDomain']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryAcceptedDomain API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryPublisher({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeCointrunkV1.query.queryPublisher(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeCointrunkV1.query.queryPublisher({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'Publisher', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryPublisher', payload: { options: { all }, params: {...key},query }})
				return getters['getPublisher']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryPublisher API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryPublisherByIndex({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeCointrunkV1.query.queryPublisherByIndex( key.index)).data
				
					
				commit('QUERY', { query: 'PublisherByIndex', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryPublisherByIndex', payload: { options: { all }, params: {...key},query }})
				return getters['getPublisherByIndex']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryPublisherByIndex API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryAllArticles({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeCointrunkV1.query.queryAllArticles(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeCointrunkV1.query.queryAllArticles({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'AllArticles', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryAllArticles', payload: { options: { all }, params: {...key},query }})
				return getters['getAllArticles']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryAllArticles API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryAllAnonArticlesCounters({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeCointrunkV1.query.queryAllAnonArticlesCounters(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeCointrunkV1.query.queryAllAnonArticlesCounters({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'AllAnonArticlesCounters', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryAllAnonArticlesCounters', payload: { options: { all }, params: {...key},query }})
				return getters['getAllAnonArticlesCounters']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryAllAnonArticlesCounters API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		async sendMsgPayPublisherRespect({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeCointrunkV1.tx.sendMsgPayPublisherRespect({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgPayPublisherRespect:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgPayPublisherRespect:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		async sendMsgAddArticle({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeCointrunkV1.tx.sendMsgAddArticle({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgAddArticle:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgAddArticle:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		
		async MsgPayPublisherRespect({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeCointrunkV1.tx.msgPayPublisherRespect({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgPayPublisherRespect:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgPayPublisherRespect:Create Could not create message: ' + e.message)
				}
			}
		},
		async MsgAddArticle({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeCointrunkV1.tx.msgAddArticle({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgAddArticle:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgAddArticle:Create Could not create message: ' + e.message)
				}
			}
		},
		
	}
}
