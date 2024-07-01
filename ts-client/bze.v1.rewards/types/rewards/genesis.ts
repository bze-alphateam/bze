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
  stakingRewardList: StakingReward[];
  stakingRewardsCounter: number;
  tradingRewardsCounter: number;
  activeTradingRewardList: TradingReward[];
  pendingTradingRewardList: TradingReward[];
  stakingRewardParticipantList: StakingRewardParticipant[];
  pendingUnlockParticipantList: PendingUnlockParticipant[];
  tradingRewardLeaderboardList: TradingRewardLeaderboard[];
  tradingRewardCandidateList: TradingRewardCandidate[];
  marketIdTradingRewardIdList: MarketIdTradingRewardId[];
  pendingTradingRewardExpirationList: TradingRewardExpiration[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  activeTradingRewardExpirationList: TradingRewardExpiration[];
}

const baseGenesisState: object = {
  stakingRewardsCounter: 0,
  tradingRewardsCounter: 0,
};

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.stakingRewardList) {
      StakingReward.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.stakingRewardsCounter !== 0) {
      writer.uint32(24).uint64(message.stakingRewardsCounter);
    }
    if (message.tradingRewardsCounter !== 0) {
      writer.uint32(32).uint64(message.tradingRewardsCounter);
    }
    for (const v of message.activeTradingRewardList) {
      TradingReward.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.pendingTradingRewardList) {
      TradingReward.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.stakingRewardParticipantList) {
      StakingRewardParticipant.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    for (const v of message.pendingUnlockParticipantList) {
      PendingUnlockParticipant.encode(v!, writer.uint32(66).fork()).ldelim();
    }
    for (const v of message.tradingRewardLeaderboardList) {
      TradingRewardLeaderboard.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    for (const v of message.tradingRewardCandidateList) {
      TradingRewardCandidate.encode(v!, writer.uint32(82).fork()).ldelim();
    }
    for (const v of message.marketIdTradingRewardIdList) {
      MarketIdTradingRewardId.encode(v!, writer.uint32(90).fork()).ldelim();
    }
    for (const v of message.pendingTradingRewardExpirationList) {
      TradingRewardExpiration.encode(v!, writer.uint32(98).fork()).ldelim();
    }
    for (const v of message.activeTradingRewardExpirationList) {
      TradingRewardExpiration.encode(v!, writer.uint32(106).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.stakingRewardList = [];
    message.activeTradingRewardList = [];
    message.pendingTradingRewardList = [];
    message.stakingRewardParticipantList = [];
    message.pendingUnlockParticipantList = [];
    message.tradingRewardLeaderboardList = [];
    message.tradingRewardCandidateList = [];
    message.marketIdTradingRewardIdList = [];
    message.pendingTradingRewardExpirationList = [];
    message.activeTradingRewardExpirationList = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.stakingRewardList.push(
            StakingReward.decode(reader, reader.uint32())
          );
          break;
        case 3:
          message.stakingRewardsCounter = longToNumber(reader.uint64() as Long);
          break;
        case 4:
          message.tradingRewardsCounter = longToNumber(reader.uint64() as Long);
          break;
        case 5:
          message.activeTradingRewardList.push(
            TradingReward.decode(reader, reader.uint32())
          );
          break;
        case 6:
          message.pendingTradingRewardList.push(
            TradingReward.decode(reader, reader.uint32())
          );
          break;
        case 7:
          message.stakingRewardParticipantList.push(
            StakingRewardParticipant.decode(reader, reader.uint32())
          );
          break;
        case 8:
          message.pendingUnlockParticipantList.push(
            PendingUnlockParticipant.decode(reader, reader.uint32())
          );
          break;
        case 9:
          message.tradingRewardLeaderboardList.push(
            TradingRewardLeaderboard.decode(reader, reader.uint32())
          );
          break;
        case 10:
          message.tradingRewardCandidateList.push(
            TradingRewardCandidate.decode(reader, reader.uint32())
          );
          break;
        case 11:
          message.marketIdTradingRewardIdList.push(
            MarketIdTradingRewardId.decode(reader, reader.uint32())
          );
          break;
        case 12:
          message.pendingTradingRewardExpirationList.push(
            TradingRewardExpiration.decode(reader, reader.uint32())
          );
          break;
        case 13:
          message.activeTradingRewardExpirationList.push(
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
    message.stakingRewardList = [];
    message.activeTradingRewardList = [];
    message.pendingTradingRewardList = [];
    message.stakingRewardParticipantList = [];
    message.pendingUnlockParticipantList = [];
    message.tradingRewardLeaderboardList = [];
    message.tradingRewardCandidateList = [];
    message.marketIdTradingRewardIdList = [];
    message.pendingTradingRewardExpirationList = [];
    message.activeTradingRewardExpirationList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    if (
      object.stakingRewardList !== undefined &&
      object.stakingRewardList !== null
    ) {
      for (const e of object.stakingRewardList) {
        message.stakingRewardList.push(StakingReward.fromJSON(e));
      }
    }
    if (
      object.stakingRewardsCounter !== undefined &&
      object.stakingRewardsCounter !== null
    ) {
      message.stakingRewardsCounter = Number(object.stakingRewardsCounter);
    } else {
      message.stakingRewardsCounter = 0;
    }
    if (
      object.tradingRewardsCounter !== undefined &&
      object.tradingRewardsCounter !== null
    ) {
      message.tradingRewardsCounter = Number(object.tradingRewardsCounter);
    } else {
      message.tradingRewardsCounter = 0;
    }
    if (
      object.activeTradingRewardList !== undefined &&
      object.activeTradingRewardList !== null
    ) {
      for (const e of object.activeTradingRewardList) {
        message.activeTradingRewardList.push(TradingReward.fromJSON(e));
      }
    }
    if (
      object.pendingTradingRewardList !== undefined &&
      object.pendingTradingRewardList !== null
    ) {
      for (const e of object.pendingTradingRewardList) {
        message.pendingTradingRewardList.push(TradingReward.fromJSON(e));
      }
    }
    if (
      object.stakingRewardParticipantList !== undefined &&
      object.stakingRewardParticipantList !== null
    ) {
      for (const e of object.stakingRewardParticipantList) {
        message.stakingRewardParticipantList.push(
          StakingRewardParticipant.fromJSON(e)
        );
      }
    }
    if (
      object.pendingUnlockParticipantList !== undefined &&
      object.pendingUnlockParticipantList !== null
    ) {
      for (const e of object.pendingUnlockParticipantList) {
        message.pendingUnlockParticipantList.push(
          PendingUnlockParticipant.fromJSON(e)
        );
      }
    }
    if (
      object.tradingRewardLeaderboardList !== undefined &&
      object.tradingRewardLeaderboardList !== null
    ) {
      for (const e of object.tradingRewardLeaderboardList) {
        message.tradingRewardLeaderboardList.push(
          TradingRewardLeaderboard.fromJSON(e)
        );
      }
    }
    if (
      object.tradingRewardCandidateList !== undefined &&
      object.tradingRewardCandidateList !== null
    ) {
      for (const e of object.tradingRewardCandidateList) {
        message.tradingRewardCandidateList.push(
          TradingRewardCandidate.fromJSON(e)
        );
      }
    }
    if (
      object.marketIdTradingRewardIdList !== undefined &&
      object.marketIdTradingRewardIdList !== null
    ) {
      for (const e of object.marketIdTradingRewardIdList) {
        message.marketIdTradingRewardIdList.push(
          MarketIdTradingRewardId.fromJSON(e)
        );
      }
    }
    if (
      object.pendingTradingRewardExpirationList !== undefined &&
      object.pendingTradingRewardExpirationList !== null
    ) {
      for (const e of object.pendingTradingRewardExpirationList) {
        message.pendingTradingRewardExpirationList.push(
          TradingRewardExpiration.fromJSON(e)
        );
      }
    }
    if (
      object.activeTradingRewardExpirationList !== undefined &&
      object.activeTradingRewardExpirationList !== null
    ) {
      for (const e of object.activeTradingRewardExpirationList) {
        message.activeTradingRewardExpirationList.push(
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
    if (message.stakingRewardList) {
      obj.stakingRewardList = message.stakingRewardList.map((e) =>
        e ? StakingReward.toJSON(e) : undefined
      );
    } else {
      obj.stakingRewardList = [];
    }
    message.stakingRewardsCounter !== undefined &&
      (obj.stakingRewardsCounter = message.stakingRewardsCounter);
    message.tradingRewardsCounter !== undefined &&
      (obj.tradingRewardsCounter = message.tradingRewardsCounter);
    if (message.activeTradingRewardList) {
      obj.activeTradingRewardList = message.activeTradingRewardList.map((e) =>
        e ? TradingReward.toJSON(e) : undefined
      );
    } else {
      obj.activeTradingRewardList = [];
    }
    if (message.pendingTradingRewardList) {
      obj.pendingTradingRewardList = message.pendingTradingRewardList.map((e) =>
        e ? TradingReward.toJSON(e) : undefined
      );
    } else {
      obj.pendingTradingRewardList = [];
    }
    if (message.stakingRewardParticipantList) {
      obj.stakingRewardParticipantList = message.stakingRewardParticipantList.map(
        (e) => (e ? StakingRewardParticipant.toJSON(e) : undefined)
      );
    } else {
      obj.stakingRewardParticipantList = [];
    }
    if (message.pendingUnlockParticipantList) {
      obj.pendingUnlockParticipantList = message.pendingUnlockParticipantList.map(
        (e) => (e ? PendingUnlockParticipant.toJSON(e) : undefined)
      );
    } else {
      obj.pendingUnlockParticipantList = [];
    }
    if (message.tradingRewardLeaderboardList) {
      obj.tradingRewardLeaderboardList = message.tradingRewardLeaderboardList.map(
        (e) => (e ? TradingRewardLeaderboard.toJSON(e) : undefined)
      );
    } else {
      obj.tradingRewardLeaderboardList = [];
    }
    if (message.tradingRewardCandidateList) {
      obj.tradingRewardCandidateList = message.tradingRewardCandidateList.map(
        (e) => (e ? TradingRewardCandidate.toJSON(e) : undefined)
      );
    } else {
      obj.tradingRewardCandidateList = [];
    }
    if (message.marketIdTradingRewardIdList) {
      obj.marketIdTradingRewardIdList = message.marketIdTradingRewardIdList.map(
        (e) => (e ? MarketIdTradingRewardId.toJSON(e) : undefined)
      );
    } else {
      obj.marketIdTradingRewardIdList = [];
    }
    if (message.pendingTradingRewardExpirationList) {
      obj.pendingTradingRewardExpirationList = message.pendingTradingRewardExpirationList.map(
        (e) => (e ? TradingRewardExpiration.toJSON(e) : undefined)
      );
    } else {
      obj.pendingTradingRewardExpirationList = [];
    }
    if (message.activeTradingRewardExpirationList) {
      obj.activeTradingRewardExpirationList = message.activeTradingRewardExpirationList.map(
        (e) => (e ? TradingRewardExpiration.toJSON(e) : undefined)
      );
    } else {
      obj.activeTradingRewardExpirationList = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.stakingRewardList = [];
    message.activeTradingRewardList = [];
    message.pendingTradingRewardList = [];
    message.stakingRewardParticipantList = [];
    message.pendingUnlockParticipantList = [];
    message.tradingRewardLeaderboardList = [];
    message.tradingRewardCandidateList = [];
    message.marketIdTradingRewardIdList = [];
    message.pendingTradingRewardExpirationList = [];
    message.activeTradingRewardExpirationList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    if (
      object.stakingRewardList !== undefined &&
      object.stakingRewardList !== null
    ) {
      for (const e of object.stakingRewardList) {
        message.stakingRewardList.push(StakingReward.fromPartial(e));
      }
    }
    if (
      object.stakingRewardsCounter !== undefined &&
      object.stakingRewardsCounter !== null
    ) {
      message.stakingRewardsCounter = object.stakingRewardsCounter;
    } else {
      message.stakingRewardsCounter = 0;
    }
    if (
      object.tradingRewardsCounter !== undefined &&
      object.tradingRewardsCounter !== null
    ) {
      message.tradingRewardsCounter = object.tradingRewardsCounter;
    } else {
      message.tradingRewardsCounter = 0;
    }
    if (
      object.activeTradingRewardList !== undefined &&
      object.activeTradingRewardList !== null
    ) {
      for (const e of object.activeTradingRewardList) {
        message.activeTradingRewardList.push(TradingReward.fromPartial(e));
      }
    }
    if (
      object.pendingTradingRewardList !== undefined &&
      object.pendingTradingRewardList !== null
    ) {
      for (const e of object.pendingTradingRewardList) {
        message.pendingTradingRewardList.push(TradingReward.fromPartial(e));
      }
    }
    if (
      object.stakingRewardParticipantList !== undefined &&
      object.stakingRewardParticipantList !== null
    ) {
      for (const e of object.stakingRewardParticipantList) {
        message.stakingRewardParticipantList.push(
          StakingRewardParticipant.fromPartial(e)
        );
      }
    }
    if (
      object.pendingUnlockParticipantList !== undefined &&
      object.pendingUnlockParticipantList !== null
    ) {
      for (const e of object.pendingUnlockParticipantList) {
        message.pendingUnlockParticipantList.push(
          PendingUnlockParticipant.fromPartial(e)
        );
      }
    }
    if (
      object.tradingRewardLeaderboardList !== undefined &&
      object.tradingRewardLeaderboardList !== null
    ) {
      for (const e of object.tradingRewardLeaderboardList) {
        message.tradingRewardLeaderboardList.push(
          TradingRewardLeaderboard.fromPartial(e)
        );
      }
    }
    if (
      object.tradingRewardCandidateList !== undefined &&
      object.tradingRewardCandidateList !== null
    ) {
      for (const e of object.tradingRewardCandidateList) {
        message.tradingRewardCandidateList.push(
          TradingRewardCandidate.fromPartial(e)
        );
      }
    }
    if (
      object.marketIdTradingRewardIdList !== undefined &&
      object.marketIdTradingRewardIdList !== null
    ) {
      for (const e of object.marketIdTradingRewardIdList) {
        message.marketIdTradingRewardIdList.push(
          MarketIdTradingRewardId.fromPartial(e)
        );
      }
    }
    if (
      object.pendingTradingRewardExpirationList !== undefined &&
      object.pendingTradingRewardExpirationList !== null
    ) {
      for (const e of object.pendingTradingRewardExpirationList) {
        message.pendingTradingRewardExpirationList.push(
          TradingRewardExpiration.fromPartial(e)
        );
      }
    }
    if (
      object.activeTradingRewardExpirationList !== undefined &&
      object.activeTradingRewardExpirationList !== null
    ) {
      for (const e of object.activeTradingRewardExpirationList) {
        message.activeTradingRewardExpirationList.push(
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
