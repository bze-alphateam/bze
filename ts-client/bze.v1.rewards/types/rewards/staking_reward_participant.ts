/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.v1.rewards";

export interface StakingRewardParticipant {
  address: string;
  rewardId: string;
  /** stake[address] */
  amount: string;
  /** S0[address] */
  joinedAt: string;
}

export interface PendingUnlockParticipant {
  index: string;
  address: string;
  amount: string;
  denom: string;
}

const baseStakingRewardParticipant: object = {
  address: "",
  rewardId: "",
  amount: "",
  joinedAt: "",
};

export const StakingRewardParticipant = {
  encode(
    message: StakingRewardParticipant,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.rewardId !== "") {
      writer.uint32(18).string(message.rewardId);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.joinedAt !== "") {
      writer.uint32(34).string(message.joinedAt);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): StakingRewardParticipant {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseStakingRewardParticipant,
    } as StakingRewardParticipant;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.address = reader.string();
          break;
        case 2:
          message.rewardId = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.joinedAt = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): StakingRewardParticipant {
    const message = {
      ...baseStakingRewardParticipant,
    } as StakingRewardParticipant;
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = String(object.rewardId);
    } else {
      message.rewardId = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    if (object.joinedAt !== undefined && object.joinedAt !== null) {
      message.joinedAt = String(object.joinedAt);
    } else {
      message.joinedAt = "";
    }
    return message;
  },

  toJSON(message: StakingRewardParticipant): unknown {
    const obj: any = {};
    message.address !== undefined && (obj.address = message.address);
    message.rewardId !== undefined && (obj.rewardId = message.rewardId);
    message.amount !== undefined && (obj.amount = message.amount);
    message.joinedAt !== undefined && (obj.joinedAt = message.joinedAt);
    return obj;
  },

  fromPartial(
    object: DeepPartial<StakingRewardParticipant>
  ): StakingRewardParticipant {
    const message = {
      ...baseStakingRewardParticipant,
    } as StakingRewardParticipant;
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = object.rewardId;
    } else {
      message.rewardId = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    if (object.joinedAt !== undefined && object.joinedAt !== null) {
      message.joinedAt = object.joinedAt;
    } else {
      message.joinedAt = "";
    }
    return message;
  },
};

const basePendingUnlockParticipant: object = {
  index: "",
  address: "",
  amount: "",
  denom: "",
};

export const PendingUnlockParticipant = {
  encode(
    message: PendingUnlockParticipant,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.denom !== "") {
      writer.uint32(34).string(message.denom);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): PendingUnlockParticipant {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePendingUnlockParticipant,
    } as PendingUnlockParticipant;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        case 2:
          message.address = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.denom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PendingUnlockParticipant {
    const message = {
      ...basePendingUnlockParticipant,
    } as PendingUnlockParticipant;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    return message;
  },

  toJSON(message: PendingUnlockParticipant): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.address !== undefined && (obj.address = message.address);
    message.amount !== undefined && (obj.amount = message.amount);
    message.denom !== undefined && (obj.denom = message.denom);
    return obj;
  },

  fromPartial(
    object: DeepPartial<PendingUnlockParticipant>
  ): PendingUnlockParticipant {
    const message = {
      ...basePendingUnlockParticipant,
    } as PendingUnlockParticipant;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    return message;
  },
};

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
