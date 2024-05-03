/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.v1.rewards";

export interface StakingRewardParticipant {
  address: string;
  reward_id: string;
  /** stake[address] */
  amount: string;
  /** S0[address] */
  joined_at: string;
}

export interface PendingUnlockParticipant {
  index: string;
  address: string;
  amount: string;
  denom: string;
}

const baseStakingRewardParticipant: object = {
  address: "",
  reward_id: "",
  amount: "",
  joined_at: "",
};

export const StakingRewardParticipant = {
  encode(
    message: StakingRewardParticipant,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.reward_id !== "") {
      writer.uint32(18).string(message.reward_id);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.joined_at !== "") {
      writer.uint32(34).string(message.joined_at);
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
          message.reward_id = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.joined_at = reader.string();
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
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = String(object.reward_id);
    } else {
      message.reward_id = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    if (object.joined_at !== undefined && object.joined_at !== null) {
      message.joined_at = String(object.joined_at);
    } else {
      message.joined_at = "";
    }
    return message;
  },

  toJSON(message: StakingRewardParticipant): unknown {
    const obj: any = {};
    message.address !== undefined && (obj.address = message.address);
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    message.amount !== undefined && (obj.amount = message.amount);
    message.joined_at !== undefined && (obj.joined_at = message.joined_at);
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
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = object.reward_id;
    } else {
      message.reward_id = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    if (object.joined_at !== undefined && object.joined_at !== null) {
      message.joined_at = object.joined_at;
    } else {
      message.joined_at = "";
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
