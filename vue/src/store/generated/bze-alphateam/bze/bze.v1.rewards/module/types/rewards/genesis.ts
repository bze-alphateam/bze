/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Params } from "../rewards/params";
import { StakingReward } from "../rewards/staking_reward";
import {
  TradingReward,
  TradingRewardLeaderboard,
  TradingRewardCandidate,
  MarketIdTradingRewardId,
  TradingRewardExpiration,
} from "../rewards/trading_reward";
import {
  StakingRewardParticipant,
  PendingUnlockParticipant,
} from "../rewards/staking_reward_participant";

export const protobufPackage = "bze.v1.rewards";

/** GenesisState defines the rewards module's genesis state. */
export interface GenesisState {
  params: Params | undefined;
  staking_reward_list: StakingReward[];
  staking_rewards_counter: number;
  trading_rewards_counter: number;
  active_trading_reward_list: TradingReward[];
  pending_trading_reward_list: TradingReward[];
  staking_reward_participant_list: StakingRewardParticipant[];
  pending_unlock_participant_list: PendingUnlockParticipant[];
  trading_reward_leaderboard_list: TradingRewardLeaderboard[];
  trading_reward_candidate_list: TradingRewardCandidate[];
  market_id_trading_reward_id_list: MarketIdTradingRewardId[];
  pending_trading_reward_expiration_list: TradingRewardExpiration[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  active_trading_reward_expiration_list: TradingRewardExpiration[];
}

const baseGenesisState: object = {
  staking_rewards_counter: 0,
  trading_rewards_counter: 0,
};

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.staking_reward_list) {
      StakingReward.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.staking_rewards_counter !== 0) {
      writer.uint32(24).uint64(message.staking_rewards_counter);
    }
    if (message.trading_rewards_counter !== 0) {
      writer.uint32(32).uint64(message.trading_rewards_counter);
    }
    for (const v of message.active_trading_reward_list) {
      TradingReward.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.pending_trading_reward_list) {
      TradingReward.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.staking_reward_participant_list) {
      StakingRewardParticipant.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    for (const v of message.pending_unlock_participant_list) {
      PendingUnlockParticipant.encode(v!, writer.uint32(66).fork()).ldelim();
    }
    for (const v of message.trading_reward_leaderboard_list) {
      TradingRewardLeaderboard.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    for (const v of message.trading_reward_candidate_list) {
      TradingRewardCandidate.encode(v!, writer.uint32(82).fork()).ldelim();
    }
    for (const v of message.market_id_trading_reward_id_list) {
      MarketIdTradingRewardId.encode(v!, writer.uint32(90).fork()).ldelim();
    }
    for (const v of message.pending_trading_reward_expiration_list) {
      TradingRewardExpiration.encode(v!, writer.uint32(98).fork()).ldelim();
    }
    for (const v of message.active_trading_reward_expiration_list) {
      TradingRewardExpiration.encode(v!, writer.uint32(106).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.staking_reward_list = [];
    message.active_trading_reward_list = [];
    message.pending_trading_reward_list = [];
    message.staking_reward_participant_list = [];
    message.pending_unlock_participant_list = [];
    message.trading_reward_leaderboard_list = [];
    message.trading_reward_candidate_list = [];
    message.market_id_trading_reward_id_list = [];
    message.pending_trading_reward_expiration_list = [];
    message.active_trading_reward_expiration_list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.staking_reward_list.push(
            StakingReward.decode(reader, reader.uint32())
          );
          break;
        case 3:
          message.staking_rewards_counter = longToNumber(
            reader.uint64() as Long
          );
          break;
        case 4:
          message.trading_rewards_counter = longToNumber(
            reader.uint64() as Long
          );
          break;
        case 5:
          message.active_trading_reward_list.push(
            TradingReward.decode(reader, reader.uint32())
          );
          break;
        case 6:
          message.pending_trading_reward_list.push(
            TradingReward.decode(reader, reader.uint32())
          );
          break;
        case 7:
          message.staking_reward_participant_list.push(
            StakingRewardParticipant.decode(reader, reader.uint32())
          );
          break;
        case 8:
          message.pending_unlock_participant_list.push(
            PendingUnlockParticipant.decode(reader, reader.uint32())
          );
          break;
        case 9:
          message.trading_reward_leaderboard_list.push(
            TradingRewardLeaderboard.decode(reader, reader.uint32())
          );
          break;
        case 10:
          message.trading_reward_candidate_list.push(
            TradingRewardCandidate.decode(reader, reader.uint32())
          );
          break;
        case 11:
          message.market_id_trading_reward_id_list.push(
            MarketIdTradingRewardId.decode(reader, reader.uint32())
          );
          break;
        case 12:
          message.pending_trading_reward_expiration_list.push(
            TradingRewardExpiration.decode(reader, reader.uint32())
          );
          break;
        case 13:
          message.active_trading_reward_expiration_list.push(
            TradingRewardExpiration.decode(reader, reader.uint32())
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.staking_reward_list = [];
    message.active_trading_reward_list = [];
    message.pending_trading_reward_list = [];
    message.staking_reward_participant_list = [];
    message.pending_unlock_participant_list = [];
    message.trading_reward_leaderboard_list = [];
    message.trading_reward_candidate_list = [];
    message.market_id_trading_reward_id_list = [];
    message.pending_trading_reward_expiration_list = [];
    message.active_trading_reward_expiration_list = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    if (
      object.staking_reward_list !== undefined &&
      object.staking_reward_list !== null
    ) {
      for (const e of object.staking_reward_list) {
        message.staking_reward_list.push(StakingReward.fromJSON(e));
      }
    }
    if (
      object.staking_rewards_counter !== undefined &&
      object.staking_rewards_counter !== null
    ) {
      message.staking_rewards_counter = Number(object.staking_rewards_counter);
    } else {
      message.staking_rewards_counter = 0;
    }
    if (
      object.trading_rewards_counter !== undefined &&
      object.trading_rewards_counter !== null
    ) {
      message.trading_rewards_counter = Number(object.trading_rewards_counter);
    } else {
      message.trading_rewards_counter = 0;
    }
    if (
      object.active_trading_reward_list !== undefined &&
      object.active_trading_reward_list !== null
    ) {
      for (const e of object.active_trading_reward_list) {
        message.active_trading_reward_list.push(TradingReward.fromJSON(e));
      }
    }
    if (
      object.pending_trading_reward_list !== undefined &&
      object.pending_trading_reward_list !== null
    ) {
      for (const e of object.pending_trading_reward_list) {
        message.pending_trading_reward_list.push(TradingReward.fromJSON(e));
      }
    }
    if (
      object.staking_reward_participant_list !== undefined &&
      object.staking_reward_participant_list !== null
    ) {
      for (const e of object.staking_reward_participant_list) {
        message.staking_reward_participant_list.push(
          StakingRewardParticipant.fromJSON(e)
        );
      }
    }
    if (
      object.pending_unlock_participant_list !== undefined &&
      object.pending_unlock_participant_list !== null
    ) {
      for (const e of object.pending_unlock_participant_list) {
        message.pending_unlock_participant_list.push(
          PendingUnlockParticipant.fromJSON(e)
        );
      }
    }
    if (
      object.trading_reward_leaderboard_list !== undefined &&
      object.trading_reward_leaderboard_list !== null
    ) {
      for (const e of object.trading_reward_leaderboard_list) {
        message.trading_reward_leaderboard_list.push(
          TradingRewardLeaderboard.fromJSON(e)
        );
      }
    }
    if (
      object.trading_reward_candidate_list !== undefined &&
      object.trading_reward_candidate_list !== null
    ) {
      for (const e of object.trading_reward_candidate_list) {
        message.trading_reward_candidate_list.push(
          TradingRewardCandidate.fromJSON(e)
        );
      }
    }
    if (
      object.market_id_trading_reward_id_list !== undefined &&
      object.market_id_trading_reward_id_list !== null
    ) {
      for (const e of object.market_id_trading_reward_id_list) {
        message.market_id_trading_reward_id_list.push(
          MarketIdTradingRewardId.fromJSON(e)
        );
      }
    }
    if (
      object.pending_trading_reward_expiration_list !== undefined &&
      object.pending_trading_reward_expiration_list !== null
    ) {
      for (const e of object.pending_trading_reward_expiration_list) {
        message.pending_trading_reward_expiration_list.push(
          TradingRewardExpiration.fromJSON(e)
        );
      }
    }
    if (
      object.active_trading_reward_expiration_list !== undefined &&
      object.active_trading_reward_expiration_list !== null
    ) {
      for (const e of object.active_trading_reward_expiration_list) {
        message.active_trading_reward_expiration_list.push(
          TradingRewardExpiration.fromJSON(e)
        );
      }
    }
    return message;
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined &&
      (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    if (message.staking_reward_list) {
      obj.staking_reward_list = message.staking_reward_list.map((e) =>
        e ? StakingReward.toJSON(e) : undefined
      );
    } else {
      obj.staking_reward_list = [];
    }
    message.staking_rewards_counter !== undefined &&
      (obj.staking_rewards_counter = message.staking_rewards_counter);
    message.trading_rewards_counter !== undefined &&
      (obj.trading_rewards_counter = message.trading_rewards_counter);
    if (message.active_trading_reward_list) {
      obj.active_trading_reward_list = message.active_trading_reward_list.map(
        (e) => (e ? TradingReward.toJSON(e) : undefined)
      );
    } else {
      obj.active_trading_reward_list = [];
    }
    if (message.pending_trading_reward_list) {
      obj.pending_trading_reward_list = message.pending_trading_reward_list.map(
        (e) => (e ? TradingReward.toJSON(e) : undefined)
      );
    } else {
      obj.pending_trading_reward_list = [];
    }
    if (message.staking_reward_participant_list) {
      obj.staking_reward_participant_list = message.staking_reward_participant_list.map(
        (e) => (e ? StakingRewardParticipant.toJSON(e) : undefined)
      );
    } else {
      obj.staking_reward_participant_list = [];
    }
    if (message.pending_unlock_participant_list) {
      obj.pending_unlock_participant_list = message.pending_unlock_participant_list.map(
        (e) => (e ? PendingUnlockParticipant.toJSON(e) : undefined)
      );
    } else {
      obj.pending_unlock_participant_list = [];
    }
    if (message.trading_reward_leaderboard_list) {
      obj.trading_reward_leaderboard_list = message.trading_reward_leaderboard_list.map(
        (e) => (e ? TradingRewardLeaderboard.toJSON(e) : undefined)
      );
    } else {
      obj.trading_reward_leaderboard_list = [];
    }
    if (message.trading_reward_candidate_list) {
      obj.trading_reward_candidate_list = message.trading_reward_candidate_list.map(
        (e) => (e ? TradingRewardCandidate.toJSON(e) : undefined)
      );
    } else {
      obj.trading_reward_candidate_list = [];
    }
    if (message.market_id_trading_reward_id_list) {
      obj.market_id_trading_reward_id_list = message.market_id_trading_reward_id_list.map(
        (e) => (e ? MarketIdTradingRewardId.toJSON(e) : undefined)
      );
    } else {
      obj.market_id_trading_reward_id_list = [];
    }
    if (message.pending_trading_reward_expiration_list) {
      obj.pending_trading_reward_expiration_list = message.pending_trading_reward_expiration_list.map(
        (e) => (e ? TradingRewardExpiration.toJSON(e) : undefined)
      );
    } else {
      obj.pending_trading_reward_expiration_list = [];
    }
    if (message.active_trading_reward_expiration_list) {
      obj.active_trading_reward_expiration_list = message.active_trading_reward_expiration_list.map(
        (e) => (e ? TradingRewardExpiration.toJSON(e) : undefined)
      );
    } else {
      obj.active_trading_reward_expiration_list = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.staking_reward_list = [];
    message.active_trading_reward_list = [];
    message.pending_trading_reward_list = [];
    message.staking_reward_participant_list = [];
    message.pending_unlock_participant_list = [];
    message.trading_reward_leaderboard_list = [];
    message.trading_reward_candidate_list = [];
    message.market_id_trading_reward_id_list = [];
    message.pending_trading_reward_expiration_list = [];
    message.active_trading_reward_expiration_list = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    if (
      object.staking_reward_list !== undefined &&
      object.staking_reward_list !== null
    ) {
      for (const e of object.staking_reward_list) {
        message.staking_reward_list.push(StakingReward.fromPartial(e));
      }
    }
    if (
      object.staking_rewards_counter !== undefined &&
      object.staking_rewards_counter !== null
    ) {
      message.staking_rewards_counter = object.staking_rewards_counter;
    } else {
      message.staking_rewards_counter = 0;
    }
    if (
      object.trading_rewards_counter !== undefined &&
      object.trading_rewards_counter !== null
    ) {
      message.trading_rewards_counter = object.trading_rewards_counter;
    } else {
      message.trading_rewards_counter = 0;
    }
    if (
      object.active_trading_reward_list !== undefined &&
      object.active_trading_reward_list !== null
    ) {
      for (const e of object.active_trading_reward_list) {
        message.active_trading_reward_list.push(TradingReward.fromPartial(e));
      }
    }
    if (
      object.pending_trading_reward_list !== undefined &&
      object.pending_trading_reward_list !== null
    ) {
      for (const e of object.pending_trading_reward_list) {
        message.pending_trading_reward_list.push(TradingReward.fromPartial(e));
      }
    }
    if (
      object.staking_reward_participant_list !== undefined &&
      object.staking_reward_participant_list !== null
    ) {
      for (const e of object.staking_reward_participant_list) {
        message.staking_reward_participant_list.push(
          StakingRewardParticipant.fromPartial(e)
        );
      }
    }
    if (
      object.pending_unlock_participant_list !== undefined &&
      object.pending_unlock_participant_list !== null
    ) {
      for (const e of object.pending_unlock_participant_list) {
        message.pending_unlock_participant_list.push(
          PendingUnlockParticipant.fromPartial(e)
        );
      }
    }
    if (
      object.trading_reward_leaderboard_list !== undefined &&
      object.trading_reward_leaderboard_list !== null
    ) {
      for (const e of object.trading_reward_leaderboard_list) {
        message.trading_reward_leaderboard_list.push(
          TradingRewardLeaderboard.fromPartial(e)
        );
      }
    }
    if (
      object.trading_reward_candidate_list !== undefined &&
      object.trading_reward_candidate_list !== null
    ) {
      for (const e of object.trading_reward_candidate_list) {
        message.trading_reward_candidate_list.push(
          TradingRewardCandidate.fromPartial(e)
        );
      }
    }
    if (
      object.market_id_trading_reward_id_list !== undefined &&
      object.market_id_trading_reward_id_list !== null
    ) {
      for (const e of object.market_id_trading_reward_id_list) {
        message.market_id_trading_reward_id_list.push(
          MarketIdTradingRewardId.fromPartial(e)
        );
      }
    }
    if (
      object.pending_trading_reward_expiration_list !== undefined &&
      object.pending_trading_reward_expiration_list !== null
    ) {
      for (const e of object.pending_trading_reward_expiration_list) {
        message.pending_trading_reward_expiration_list.push(
          TradingRewardExpiration.fromPartial(e)
        );
      }
    }
    if (
      object.active_trading_reward_expiration_list !== undefined &&
      object.active_trading_reward_expiration_list !== null
    ) {
      for (const e of object.active_trading_reward_expiration_list) {
        message.active_trading_reward_expiration_list.push(
          TradingRewardExpiration.fromPartial(e)
        );
      }
    }
    return message;
  },
};

declare var self: any | undefined;
declare var window: any | undefined;
var globalThis: any = (() => {
  if (typeof globalThis !== "undefined") return globalThis;
  if (typeof self !== "undefined") return self;
  if (typeof window !== "undefined") return window;
  if (typeof global !== "undefined") return global;
  throw "Unable to locate global object";
})();

type Builtin = Date | Function | Uint8Array | string | number | undefined;
export type DeepPartial<T> = T extends Builtin
  ? T
  : T extends Array<infer U>
  ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U>
  ? ReadonlyArray<DeepPartial<U>>
  : T extends {}
  ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (util.Long !== Long) {
  util.Long = Long as any;
  configure();
}
