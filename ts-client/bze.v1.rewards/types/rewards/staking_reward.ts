/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.v1.rewards";

export interface StakingReward {
  rewardId: string;
  prizeAmount: string;
  prizeDenom: string;
  stakingDenom: string;
  duration: number;
  payouts: number;
  minStake: number;
  lock: number;
  /** T */
  stakedAmount: string;
  /** S */
  distributedStake: string;
}

const baseStakingReward: object = {
  rewardId: "",
  prizeAmount: "",
  prizeDenom: "",
  stakingDenom: "",
  duration: 0,
  payouts: 0,
  minStake: 0,
  lock: 0,
  stakedAmount: "",
  distributedStake: "",
};

export const StakingReward = {
  encode(message: StakingReward, writer: Writer = Writer.create()): Writer {
    if (message.rewardId !== "") {
      writer.uint32(10).string(message.rewardId);
    }
    if (message.prizeAmount !== "") {
      writer.uint32(18).string(message.prizeAmount);
    }
    if (message.prizeDenom !== "") {
      writer.uint32(26).string(message.prizeDenom);
    }
    if (message.stakingDenom !== "") {
      writer.uint32(34).string(message.stakingDenom);
    }
    if (message.duration !== 0) {
      writer.uint32(40).uint32(message.duration);
    }
    if (message.payouts !== 0) {
      writer.uint32(48).uint32(message.payouts);
    }
    if (message.minStake !== 0) {
      writer.uint32(56).uint64(message.minStake);
    }
    if (message.lock !== 0) {
      writer.uint32(64).uint32(message.lock);
    }
    if (message.stakedAmount !== "") {
      writer.uint32(74).string(message.stakedAmount);
    }
    if (message.distributedStake !== "") {
      writer.uint32(82).string(message.distributedStake);
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
          message.rewardId = reader.string();
          break;
        case 2:
          message.prizeAmount = reader.string();
          break;
        case 3:
          message.prizeDenom = reader.string();
          break;
        case 4:
          message.stakingDenom = reader.string();
          break;
        case 5:
          message.duration = reader.uint32();
          break;
        case 6:
          message.payouts = reader.uint32();
          break;
        case 7:
          message.minStake = longToNumber(reader.uint64() as Long);
          break;
        case 8:
          message.lock = reader.uint32();
          break;
        case 9:
          message.stakedAmount = reader.string();
          break;
        case 10:
          message.distributedStake = reader.string();
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
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = String(object.rewardId);
    } else {
      message.rewardId = "";
    }
    if (object.prizeAmount !== undefined && object.prizeAmount !== null) {
      message.prizeAmount = String(object.prizeAmount);
    } else {
      message.prizeAmount = "";
    }
    if (object.prizeDenom !== undefined && object.prizeDenom !== null) {
      message.prizeDenom = String(object.prizeDenom);
    } else {
      message.prizeDenom = "";
    }
    if (object.stakingDenom !== undefined && object.stakingDenom !== null) {
      message.stakingDenom = String(object.stakingDenom);
    } else {
      message.stakingDenom = "";
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
    if (object.minStake !== undefined && object.minStake !== null) {
      message.minStake = Number(object.minStake);
    } else {
      message.minStake = 0;
    }
    if (object.lock !== undefined && object.lock !== null) {
      message.lock = Number(object.lock);
    } else {
      message.lock = 0;
    }
    if (object.stakedAmount !== undefined && object.stakedAmount !== null) {
      message.stakedAmount = String(object.stakedAmount);
    } else {
      message.stakedAmount = "";
    }
    if (
      object.distributedStake !== undefined &&
      object.distributedStake !== null
    ) {
      message.distributedStake = String(object.distributedStake);
    } else {
      message.distributedStake = "";
    }
    return message;
  },

  toJSON(message: StakingReward): unknown {
    const obj: any = {};
    message.rewardId !== undefined && (obj.rewardId = message.rewardId);
    message.prizeAmount !== undefined &&
      (obj.prizeAmount = message.prizeAmount);
    message.prizeDenom !== undefined && (obj.prizeDenom = message.prizeDenom);
    message.stakingDenom !== undefined &&
      (obj.stakingDenom = message.stakingDenom);
    message.duration !== undefined && (obj.duration = message.duration);
    message.payouts !== undefined && (obj.payouts = message.payouts);
    message.minStake !== undefined && (obj.minStake = message.minStake);
    message.lock !== undefined && (obj.lock = message.lock);
    message.stakedAmount !== undefined &&
      (obj.stakedAmount = message.stakedAmount);
    message.distributedStake !== undefined &&
      (obj.distributedStake = message.distributedStake);
    return obj;
  },

  fromPartial(object: DeepPartial<StakingReward>): StakingReward {
    const message = { ...baseStakingReward } as StakingReward;
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = object.rewardId;
    } else {
      message.rewardId = "";
    }
    if (object.prizeAmount !== undefined && object.prizeAmount !== null) {
      message.prizeAmount = object.prizeAmount;
    } else {
      message.prizeAmount = "";
    }
    if (object.prizeDenom !== undefined && object.prizeDenom !== null) {
      message.prizeDenom = object.prizeDenom;
    } else {
      message.prizeDenom = "";
    }
    if (object.stakingDenom !== undefined && object.stakingDenom !== null) {
      message.stakingDenom = object.stakingDenom;
    } else {
      message.stakingDenom = "";
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
    if (object.minStake !== undefined && object.minStake !== null) {
      message.minStake = object.minStake;
    } else {
      message.minStake = 0;
    }
    if (object.lock !== undefined && object.lock !== null) {
      message.lock = object.lock;
    } else {
      message.lock = 0;
    }
    if (object.stakedAmount !== undefined && object.stakedAmount !== null) {
      message.stakedAmount = object.stakedAmount;
    } else {
      message.stakedAmount = "";
    }
    if (
      object.distributedStake !== undefined &&
      object.distributedStake !== null
    ) {
      message.distributedStake = object.distributedStake;
    } else {
      message.distributedStake = "";
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
