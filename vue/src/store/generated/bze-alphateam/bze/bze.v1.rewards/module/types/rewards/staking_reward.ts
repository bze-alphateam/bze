/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.v1.rewards";

export interface StakingReward {
  reward_id: string;
  prize_amount: string;
  prize_denom: string;
  staking_denom: string;
  duration: number;
  payouts: number;
  min_stake: number;
  lock: number;
}

const baseStakingReward: object = {
  reward_id: "",
  prize_amount: "",
  prize_denom: "",
  staking_denom: "",
  duration: 0,
  payouts: 0,
  min_stake: 0,
  lock: 0,
};

export const StakingReward = {
  encode(message: StakingReward, writer: Writer = Writer.create()): Writer {
    if (message.reward_id !== "") {
      writer.uint32(10).string(message.reward_id);
    }
    if (message.prize_amount !== "") {
      writer.uint32(18).string(message.prize_amount);
    }
    if (message.prize_denom !== "") {
      writer.uint32(26).string(message.prize_denom);
    }
    if (message.staking_denom !== "") {
      writer.uint32(34).string(message.staking_denom);
    }
    if (message.duration !== 0) {
      writer.uint32(40).uint32(message.duration);
    }
    if (message.payouts !== 0) {
      writer.uint32(48).uint32(message.payouts);
    }
    if (message.min_stake !== 0) {
      writer.uint32(56).uint64(message.min_stake);
    }
    if (message.lock !== 0) {
      writer.uint32(64).uint32(message.lock);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): StakingReward {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseStakingReward } as StakingReward;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reward_id = reader.string();
          break;
        case 2:
          message.prize_amount = reader.string();
          break;
        case 3:
          message.prize_denom = reader.string();
          break;
        case 4:
          message.staking_denom = reader.string();
          break;
        case 5:
          message.duration = reader.uint32();
          break;
        case 6:
          message.payouts = reader.uint32();
          break;
        case 7:
          message.min_stake = longToNumber(reader.uint64() as Long);
          break;
        case 8:
          message.lock = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): StakingReward {
    const message = { ...baseStakingReward } as StakingReward;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = String(object.reward_id);
    } else {
      message.reward_id = "";
    }
    if (object.prize_amount !== undefined && object.prize_amount !== null) {
      message.prize_amount = String(object.prize_amount);
    } else {
      message.prize_amount = "";
    }
    if (object.prize_denom !== undefined && object.prize_denom !== null) {
      message.prize_denom = String(object.prize_denom);
    } else {
      message.prize_denom = "";
    }
    if (object.staking_denom !== undefined && object.staking_denom !== null) {
      message.staking_denom = String(object.staking_denom);
    } else {
      message.staking_denom = "";
    }
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = Number(object.duration);
    } else {
      message.duration = 0;
    }
    if (object.payouts !== undefined && object.payouts !== null) {
      message.payouts = Number(object.payouts);
    } else {
      message.payouts = 0;
    }
    if (object.min_stake !== undefined && object.min_stake !== null) {
      message.min_stake = Number(object.min_stake);
    } else {
      message.min_stake = 0;
    }
    if (object.lock !== undefined && object.lock !== null) {
      message.lock = Number(object.lock);
    } else {
      message.lock = 0;
    }
    return message;
  },

  toJSON(message: StakingReward): unknown {
    const obj: any = {};
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    message.prize_amount !== undefined &&
      (obj.prize_amount = message.prize_amount);
    message.prize_denom !== undefined &&
      (obj.prize_denom = message.prize_denom);
    message.staking_denom !== undefined &&
      (obj.staking_denom = message.staking_denom);
    message.duration !== undefined && (obj.duration = message.duration);
    message.payouts !== undefined && (obj.payouts = message.payouts);
    message.min_stake !== undefined && (obj.min_stake = message.min_stake);
    message.lock !== undefined && (obj.lock = message.lock);
    return obj;
  },

  fromPartial(object: DeepPartial<StakingReward>): StakingReward {
    const message = { ...baseStakingReward } as StakingReward;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = object.reward_id;
    } else {
      message.reward_id = "";
    }
    if (object.prize_amount !== undefined && object.prize_amount !== null) {
      message.prize_amount = object.prize_amount;
    } else {
      message.prize_amount = "";
    }
    if (object.prize_denom !== undefined && object.prize_denom !== null) {
      message.prize_denom = object.prize_denom;
    } else {
      message.prize_denom = "";
    }
    if (object.staking_denom !== undefined && object.staking_denom !== null) {
      message.staking_denom = object.staking_denom;
    } else {
      message.staking_denom = "";
    }
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = object.duration;
    } else {
      message.duration = 0;
    }
    if (object.payouts !== undefined && object.payouts !== null) {
      message.payouts = object.payouts;
    } else {
      message.payouts = 0;
    }
    if (object.min_stake !== undefined && object.min_stake !== null) {
      message.min_stake = object.min_stake;
    } else {
      message.min_stake = 0;
    }
    if (object.lock !== undefined && object.lock !== null) {
      message.lock = object.lock;
    } else {
      message.lock = 0;
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
