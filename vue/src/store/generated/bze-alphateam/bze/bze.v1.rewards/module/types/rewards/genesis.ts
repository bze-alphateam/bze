/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Params } from "../rewards/params";
import { StakingReward } from "../rewards/staking_reward";

export const protobufPackage = "bze.v1.rewards";

/** GenesisState defines the rewards module's genesis state. */
export interface GenesisState {
  params: Params | undefined;
  staking_reward_list: StakingReward[];
  staking_rewards_counter: number;
  trading_rewards_counter: number;
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
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.staking_reward_list = [];
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
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.staking_reward_list = [];
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
