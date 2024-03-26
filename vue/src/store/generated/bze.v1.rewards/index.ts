import { Client, registry, MissingWalletError } from 'bze-alphateam-bze-client-ts'

import { Params } from "bze-alphateam-bze-client-ts/bze.v1.rewards/types"
import { StakingReward } from "bze-alphateam-bze-client-ts/bze.v1.rewards/types"
import { StakingRewardParticipant } from "bze-alphateam-bze-client-ts/bze.v1.rewards/types"
import { PendingUnlockParticipant } from "bze-alphateam-bze-client-ts/bze.v1.rewards/types"
import { TradingReward } from "bze-alphateam-bze-client-ts/bze.v1.rewards/types"
import { TradingRewardExpiration } from "bze-alphateam-bze-client-ts/bze.v1.rewards/types"
import { TradingRewardLeaderboard } from "bze-alphateam-bze-client-ts/bze.v1.rewards/types"
import { TradingRewardLeaderboardEntry } from "bze-alphateam-bze-client-ts/bze.v1.rewards/types"
import { TradingRewardCandidate } from "bze-alphateam-bze-client-ts/bze.v1.rewards/types"
import { MarketIdTradingRewardId } from "bze-alphateam-bze-client-ts/bze.v1.rewards/types"


export { Params, StakingReward, StakingRewardParticipant, PendingUnlockParticipant, TradingReward, TradingRewardExpiration, TradingRewardLeaderboard, TradingRewardLeaderboardEntry, TradingRewardCandidate, MarketIdTradingRewardId };

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
				StakingReward: {},
				StakingRewardAll: {},
				TradingReward: {},
				TradingRewardAll: {},
				StakingRewardParticipant: {},
				StakingRewardParticipantAll: {},
				
				_Structure: {
						Params: getStructure(Params.fromPartial({})),
						StakingReward: getStructure(StakingReward.fromPartial({})),
						StakingRewardParticipant: getStructure(StakingRewardParticipant.fromPartial({})),
						PendingUnlockParticipant: getStructure(PendingUnlockParticipant.fromPartial({})),
						TradingReward: getStructure(TradingReward.fromPartial({})),
						TradingRewardExpiration: getStructure(TradingRewardExpiration.fromPartial({})),
						TradingRewardLeaderboard: getStructure(TradingRewardLeaderboard.fromPartial({})),
						TradingRewardLeaderboardEntry: getStructure(TradingRewardLeaderboardEntry.fromPartial({})),
						TradingRewardCandidate: getStructure(TradingRewardCandidate.fromPartial({})),
						MarketIdTradingRewardId: getStructure(MarketIdTradingRewardId.fromPartial({})),
						
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
				getStakingReward: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.StakingReward[JSON.stringify(params)] ?? {}
		},
				getStakingRewardAll: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.StakingRewardAll[JSON.stringify(params)] ?? {}
		},
				getTradingReward: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.TradingReward[JSON.stringify(params)] ?? {}
		},
				getTradingRewardAll: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.TradingRewardAll[JSON.stringify(params)] ?? {}
		},
				getStakingRewardParticipant: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.StakingRewardParticipant[JSON.stringify(params)] ?? {}
		},
				getStakingRewardParticipantAll: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.StakingRewardParticipantAll[JSON.stringify(params)] ?? {}
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
			console.log('Vuex module: bze.v1.rewards initialized!')
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
				let value= (await client.BzeV1Rewards.query.queryParams()).data
				
					
				commit('QUERY', { query: 'Params', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryParams', payload: { options: { all }, params: {...key},query }})
				return getters['getParams']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryParams API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryStakingReward({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeV1Rewards.query.queryStakingReward( key.reward_id)).data
				
					
				commit('QUERY', { query: 'StakingReward', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryStakingReward', payload: { options: { all }, params: {...key},query }})
				return getters['getStakingReward']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryStakingReward API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryStakingRewardAll({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeV1Rewards.query.queryStakingRewardAll(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeV1Rewards.query.queryStakingRewardAll({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'StakingRewardAll', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryStakingRewardAll', payload: { options: { all }, params: {...key},query }})
				return getters['getStakingRewardAll']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryStakingRewardAll API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryTradingReward({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeV1Rewards.query.queryTradingReward( key.reward_id)).data
				
					
				commit('QUERY', { query: 'TradingReward', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryTradingReward', payload: { options: { all }, params: {...key},query }})
				return getters['getTradingReward']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryTradingReward API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryTradingRewardAll({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeV1Rewards.query.queryTradingRewardAll(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeV1Rewards.query.queryTradingRewardAll({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'TradingRewardAll', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryTradingRewardAll', payload: { options: { all }, params: {...key},query }})
				return getters['getTradingRewardAll']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryTradingRewardAll API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryStakingRewardParticipant({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeV1Rewards.query.queryStakingRewardParticipant( key.address, query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeV1Rewards.query.queryStakingRewardParticipant( key.address, {...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'StakingRewardParticipant', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryStakingRewardParticipant', payload: { options: { all }, params: {...key},query }})
				return getters['getStakingRewardParticipant']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryStakingRewardParticipant API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryStakingRewardParticipantAll({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const client = initClient(rootGetters);
				let value= (await client.BzeV1Rewards.query.queryStakingRewardParticipantAll(query ?? undefined)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await client.BzeV1Rewards.query.queryStakingRewardParticipantAll({...query ?? {}, 'pagination.key':(<any> value).pagination.next_key} as any)).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'StakingRewardParticipantAll', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryStakingRewardParticipantAll', payload: { options: { all }, params: {...key},query }})
				return getters['getStakingRewardParticipantAll']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new Error('QueryClient:QueryStakingRewardParticipantAll API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		async sendMsgUpdateStakingReward({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeV1Rewards.tx.sendMsgUpdateStakingReward({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgUpdateStakingReward:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgUpdateStakingReward:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		async sendMsgCreateTradingReward({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeV1Rewards.tx.sendMsgCreateTradingReward({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgCreateTradingReward:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgCreateTradingReward:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		async sendMsgJoinStaking({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeV1Rewards.tx.sendMsgJoinStaking({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgJoinStaking:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgJoinStaking:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		async sendMsgExitStaking({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeV1Rewards.tx.sendMsgExitStaking({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgExitStaking:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgExitStaking:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		async sendMsgCreateStakingReward({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeV1Rewards.tx.sendMsgCreateStakingReward({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgCreateStakingReward:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgCreateStakingReward:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		async sendMsgClaimStakingRewards({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const client=await initClient(rootGetters)
				const result = await client.BzeV1Rewards.tx.sendMsgClaimStakingRewards({ value, fee: {amount: fee, gas: "200000"}, memo })
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgClaimStakingRewards:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgClaimStakingRewards:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		
		async MsgUpdateStakingReward({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeV1Rewards.tx.msgUpdateStakingReward({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgUpdateStakingReward:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgUpdateStakingReward:Create Could not create message: ' + e.message)
				}
			}
		},
		async MsgCreateTradingReward({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeV1Rewards.tx.msgCreateTradingReward({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgCreateTradingReward:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgCreateTradingReward:Create Could not create message: ' + e.message)
				}
			}
		},
		async MsgJoinStaking({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeV1Rewards.tx.msgJoinStaking({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgJoinStaking:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgJoinStaking:Create Could not create message: ' + e.message)
				}
			}
		},
		async MsgExitStaking({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeV1Rewards.tx.msgExitStaking({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgExitStaking:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgExitStaking:Create Could not create message: ' + e.message)
				}
			}
		},
		async MsgCreateStakingReward({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeV1Rewards.tx.msgCreateStakingReward({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgCreateStakingReward:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgCreateStakingReward:Create Could not create message: ' + e.message)
				}
			}
		},
		async MsgClaimStakingRewards({ rootGetters }, { value }) {
			try {
				const client=initClient(rootGetters)
				const msg = await client.BzeV1Rewards.tx.msgClaimStakingRewards({value})
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgClaimStakingRewards:Init Could not initialize signing client. Wallet is required.')
				} else{
					throw new Error('TxClient:MsgClaimStakingRewards:Create Could not create message: ' + e.message)
				}
			}
		},
		
	}
}
