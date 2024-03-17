/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";

export const protobufPackage = "bze.v1.rewards";

export interface MsgCreateStakingReward {
  /** msg creator */
  creator: string;
  /** the amount paid as prize for each epoch (duration) */
  prize_amount: string;
  /** the denom paid as prize */
  prize_denom: string;
  /** the denom a user has to stake in order to qualify */
  staking_denom: string;
  /** the number of days the rewards are paid */
  duration: string;
  /** the minimum amount of staking denom a user has to stake in order to qualify */
  min_stake: string;
  /** the number of days the funds are locked upon exiting stake */
  lock: string;
}

export interface MsgCreateStakingRewardResponse {
  reward_id: string;
}

export interface MsgUpdateStakingReward {
  creator: string;
  reward_id: string;
  /** the number of days the rewards are paid */
  duration: string;
}

export interface MsgUpdateStakingRewardResponse {}

export interface MsgCreateTradingReward {
  creator: string;
  /** the amount paid as prize for each slot */
  prize_amount: string;
  /** the denom paid as prize */
  prize_denom: string;
  duration: string;
  market_id: string;
  slots: string;
}

export interface MsgCreateTradingRewardResponse {
  reward_id: string;
}

export interface MsgJoinStaking {
  creator: string;
  reward_id: string;
  amount: string;
}

export interface MsgJoinStakingResponse {}

export interface MsgExitStaking {
  creator: string;
  rewardId: string;
}

export interface MsgExitStakingResponse {}

const baseMsgCreateStakingReward: object = {
  creator: "",
  prize_amount: "",
  prize_denom: "",
  staking_denom: "",
  duration: "",
  min_stake: "",
  lock: "",
};

export const MsgCreateStakingReward = {
  encode(
    message: MsgCreateStakingReward,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
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
    if (message.duration !== "") {
      writer.uint32(42).string(message.duration);
    }
    if (message.min_stake !== "") {
      writer.uint32(50).string(message.min_stake);
    }
    if (message.lock !== "") {
      writer.uint32(58).string(message.lock);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgCreateStakingReward {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgCreateStakingReward } as MsgCreateStakingReward;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
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
          message.duration = reader.string();
          break;
        case 6:
          message.min_stake = reader.string();
          break;
        case 7:
          message.lock = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgCreateStakingReward {
    const message = { ...baseMsgCreateStakingReward } as MsgCreateStakingReward;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
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
      message.duration = String(object.duration);
    } else {
      message.duration = "";
    }
    if (object.min_stake !== undefined && object.min_stake !== null) {
      message.min_stake = String(object.min_stake);
    } else {
      message.min_stake = "";
    }
    if (object.lock !== undefined && object.lock !== null) {
      message.lock = String(object.lock);
    } else {
      message.lock = "";
    }
    return message;
  },

  toJSON(message: MsgCreateStakingReward): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.prize_amount !== undefined &&
      (obj.prize_amount = message.prize_amount);
    message.prize_denom !== undefined &&
      (obj.prize_denom = message.prize_denom);
    message.staking_denom !== undefined &&
      (obj.staking_denom = message.staking_denom);
    message.duration !== undefined && (obj.duration = message.duration);
    message.min_stake !== undefined && (obj.min_stake = message.min_stake);
    message.lock !== undefined && (obj.lock = message.lock);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgCreateStakingReward>
  ): MsgCreateStakingReward {
    const message = { ...baseMsgCreateStakingReward } as MsgCreateStakingReward;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
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
      message.duration = "";
    }
    if (object.min_stake !== undefined && object.min_stake !== null) {
      message.min_stake = object.min_stake;
    } else {
      message.min_stake = "";
    }
    if (object.lock !== undefined && object.lock !== null) {
      message.lock = object.lock;
    } else {
      message.lock = "";
    }
    return message;
  },
};

const baseMsgCreateStakingRewardResponse: object = { reward_id: "" };

export const MsgCreateStakingRewardResponse = {
  encode(
    message: MsgCreateStakingRewardResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.reward_id !== "") {
      writer.uint32(10).string(message.reward_id);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgCreateStakingRewardResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgCreateStakingRewardResponse,
    } as MsgCreateStakingRewardResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reward_id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgCreateStakingRewardResponse {
    const message = {
      ...baseMsgCreateStakingRewardResponse,
    } as MsgCreateStakingRewardResponse;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = String(object.reward_id);
    } else {
      message.reward_id = "";
    }
    return message;
  },

  toJSON(message: MsgCreateStakingRewardResponse): unknown {
    const obj: any = {};
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgCreateStakingRewardResponse>
  ): MsgCreateStakingRewardResponse {
    const message = {
      ...baseMsgCreateStakingRewardResponse,
    } as MsgCreateStakingRewardResponse;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = object.reward_id;
    } else {
      message.reward_id = "";
    }
    return message;
  },
};

const baseMsgUpdateStakingReward: object = {
  creator: "",
  reward_id: "",
  duration: "",
};

export const MsgUpdateStakingReward = {
  encode(
    message: MsgUpdateStakingReward,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.reward_id !== "") {
      writer.uint32(18).string(message.reward_id);
    }
    if (message.duration !== "") {
      writer.uint32(26).string(message.duration);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgUpdateStakingReward {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgUpdateStakingReward } as MsgUpdateStakingReward;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.reward_id = reader.string();
          break;
        case 3:
          message.duration = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgUpdateStakingReward {
    const message = { ...baseMsgUpdateStakingReward } as MsgUpdateStakingReward;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = String(object.reward_id);
    } else {
      message.reward_id = "";
    }
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = String(object.duration);
    } else {
      message.duration = "";
    }
    return message;
  },

  toJSON(message: MsgUpdateStakingReward): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    message.duration !== undefined && (obj.duration = message.duration);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgUpdateStakingReward>
  ): MsgUpdateStakingReward {
    const message = { ...baseMsgUpdateStakingReward } as MsgUpdateStakingReward;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = object.reward_id;
    } else {
      message.reward_id = "";
    }
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = object.duration;
    } else {
      message.duration = "";
    }
    return message;
  },
};

const baseMsgUpdateStakingRewardResponse: object = {};

export const MsgUpdateStakingRewardResponse = {
  encode(
    _: MsgUpdateStakingRewardResponse,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgUpdateStakingRewardResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgUpdateStakingRewardResponse,
    } as MsgUpdateStakingRewardResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): MsgUpdateStakingRewardResponse {
    const message = {
      ...baseMsgUpdateStakingRewardResponse,
    } as MsgUpdateStakingRewardResponse;
    return message;
  },

  toJSON(_: MsgUpdateStakingRewardResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<MsgUpdateStakingRewardResponse>
  ): MsgUpdateStakingRewardResponse {
    const message = {
      ...baseMsgUpdateStakingRewardResponse,
    } as MsgUpdateStakingRewardResponse;
    return message;
  },
};

const baseMsgCreateTradingReward: object = {
  creator: "",
  prize_amount: "",
  prize_denom: "",
  duration: "",
  market_id: "",
  slots: "",
};

export const MsgCreateTradingReward = {
  encode(
    message: MsgCreateTradingReward,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.prize_amount !== "") {
      writer.uint32(18).string(message.prize_amount);
    }
    if (message.prize_denom !== "") {
      writer.uint32(26).string(message.prize_denom);
    }
    if (message.duration !== "") {
      writer.uint32(34).string(message.duration);
    }
    if (message.market_id !== "") {
      writer.uint32(42).string(message.market_id);
    }
    if (message.slots !== "") {
      writer.uint32(50).string(message.slots);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgCreateTradingReward {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgCreateTradingReward } as MsgCreateTradingReward;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.prize_amount = reader.string();
          break;
        case 3:
          message.prize_denom = reader.string();
          break;
        case 4:
          message.duration = reader.string();
          break;
        case 5:
          message.market_id = reader.string();
          break;
        case 6:
          message.slots = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgCreateTradingReward {
    const message = { ...baseMsgCreateTradingReward } as MsgCreateTradingReward;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
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
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = String(object.duration);
    } else {
      message.duration = "";
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = String(object.market_id);
    } else {
      message.market_id = "";
    }
    if (object.slots !== undefined && object.slots !== null) {
      message.slots = String(object.slots);
    } else {
      message.slots = "";
    }
    return message;
  },

  toJSON(message: MsgCreateTradingReward): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.prize_amount !== undefined &&
      (obj.prize_amount = message.prize_amount);
    message.prize_denom !== undefined &&
      (obj.prize_denom = message.prize_denom);
    message.duration !== undefined && (obj.duration = message.duration);
    message.market_id !== undefined && (obj.market_id = message.market_id);
    message.slots !== undefined && (obj.slots = message.slots);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgCreateTradingReward>
  ): MsgCreateTradingReward {
    const message = { ...baseMsgCreateTradingReward } as MsgCreateTradingReward;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
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
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = object.duration;
    } else {
      message.duration = "";
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = object.market_id;
    } else {
      message.market_id = "";
    }
    if (object.slots !== undefined && object.slots !== null) {
      message.slots = object.slots;
    } else {
      message.slots = "";
    }
    return message;
  },
};

const baseMsgCreateTradingRewardResponse: object = { reward_id: "" };

export const MsgCreateTradingRewardResponse = {
  encode(
    message: MsgCreateTradingRewardResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.reward_id !== "") {
      writer.uint32(10).string(message.reward_id);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgCreateTradingRewardResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgCreateTradingRewardResponse,
    } as MsgCreateTradingRewardResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reward_id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgCreateTradingRewardResponse {
    const message = {
      ...baseMsgCreateTradingRewardResponse,
    } as MsgCreateTradingRewardResponse;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = String(object.reward_id);
    } else {
      message.reward_id = "";
    }
    return message;
  },

  toJSON(message: MsgCreateTradingRewardResponse): unknown {
    const obj: any = {};
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgCreateTradingRewardResponse>
  ): MsgCreateTradingRewardResponse {
    const message = {
      ...baseMsgCreateTradingRewardResponse,
    } as MsgCreateTradingRewardResponse;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = object.reward_id;
    } else {
      message.reward_id = "";
    }
    return message;
  },
};

const baseMsgJoinStaking: object = { creator: "", reward_id: "", amount: "" };

export const MsgJoinStaking = {
  encode(message: MsgJoinStaking, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.reward_id !== "") {
      writer.uint32(18).string(message.reward_id);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgJoinStaking {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgJoinStaking } as MsgJoinStaking;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.reward_id = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgJoinStaking {
    const message = { ...baseMsgJoinStaking } as MsgJoinStaking;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
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
    return message;
  },

  toJSON(message: MsgJoinStaking): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    message.amount !== undefined && (obj.amount = message.amount);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgJoinStaking>): MsgJoinStaking {
    const message = { ...baseMsgJoinStaking } as MsgJoinStaking;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
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
    return message;
  },
};

const baseMsgJoinStakingResponse: object = {};

export const MsgJoinStakingResponse = {
  encode(_: MsgJoinStakingResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgJoinStakingResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgJoinStakingResponse } as MsgJoinStakingResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): MsgJoinStakingResponse {
    const message = { ...baseMsgJoinStakingResponse } as MsgJoinStakingResponse;
    return message;
  },

  toJSON(_: MsgJoinStakingResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgJoinStakingResponse>): MsgJoinStakingResponse {
    const message = { ...baseMsgJoinStakingResponse } as MsgJoinStakingResponse;
    return message;
  },
};

const baseMsgExitStaking: object = { creator: "", rewardId: "" };

export const MsgExitStaking = {
  encode(message: MsgExitStaking, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.rewardId !== "") {
      writer.uint32(18).string(message.rewardId);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgExitStaking {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgExitStaking } as MsgExitStaking;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.rewardId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgExitStaking {
    const message = { ...baseMsgExitStaking } as MsgExitStaking;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = String(object.rewardId);
    } else {
      message.rewardId = "";
    }
    return message;
  },

  toJSON(message: MsgExitStaking): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.rewardId !== undefined && (obj.rewardId = message.rewardId);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgExitStaking>): MsgExitStaking {
    const message = { ...baseMsgExitStaking } as MsgExitStaking;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = object.rewardId;
    } else {
      message.rewardId = "";
    }
    return message;
  },
};

const baseMsgExitStakingResponse: object = {};

export const MsgExitStakingResponse = {
  encode(_: MsgExitStakingResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgExitStakingResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgExitStakingResponse } as MsgExitStakingResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): MsgExitStakingResponse {
    const message = { ...baseMsgExitStakingResponse } as MsgExitStakingResponse;
    return message;
  },

  toJSON(_: MsgExitStakingResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgExitStakingResponse>): MsgExitStakingResponse {
    const message = { ...baseMsgExitStakingResponse } as MsgExitStakingResponse;
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  CreateStakingReward(
    request: MsgCreateStakingReward
  ): Promise<MsgCreateStakingRewardResponse>;
  UpdateStakingReward(
    request: MsgUpdateStakingReward
  ): Promise<MsgUpdateStakingRewardResponse>;
  CreateTradingReward(
    request: MsgCreateTradingReward
  ): Promise<MsgCreateTradingRewardResponse>;
  JoinStaking(request: MsgJoinStaking): Promise<MsgJoinStakingResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  ExitStaking(request: MsgExitStaking): Promise<MsgExitStakingResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  CreateStakingReward(
    request: MsgCreateStakingReward
  ): Promise<MsgCreateStakingRewardResponse> {
    const data = MsgCreateStakingReward.encode(request).finish();
    const promise = this.rpc.request(
      "bze.v1.rewards.Msg",
      "CreateStakingReward",
      data
    );
    return promise.then((data) =>
      MsgCreateStakingRewardResponse.decode(new Reader(data))
    );
  }

  UpdateStakingReward(
    request: MsgUpdateStakingReward
  ): Promise<MsgUpdateStakingRewardResponse> {
    const data = MsgUpdateStakingReward.encode(request).finish();
    const promise = this.rpc.request(
      "bze.v1.rewards.Msg",
      "UpdateStakingReward",
      data
    );
    return promise.then((data) =>
      MsgUpdateStakingRewardResponse.decode(new Reader(data))
    );
  }

  CreateTradingReward(
    request: MsgCreateTradingReward
  ): Promise<MsgCreateTradingRewardResponse> {
    const data = MsgCreateTradingReward.encode(request).finish();
    const promise = this.rpc.request(
      "bze.v1.rewards.Msg",
      "CreateTradingReward",
      data
    );
    return promise.then((data) =>
      MsgCreateTradingRewardResponse.decode(new Reader(data))
    );
  }

  JoinStaking(request: MsgJoinStaking): Promise<MsgJoinStakingResponse> {
    const data = MsgJoinStaking.encode(request).finish();
    const promise = this.rpc.request("bze.v1.rewards.Msg", "JoinStaking", data);
    return promise.then((data) =>
      MsgJoinStakingResponse.decode(new Reader(data))
    );
  }

  ExitStaking(request: MsgExitStaking): Promise<MsgExitStakingResponse> {
    const data = MsgExitStaking.encode(request).finish();
    const promise = this.rpc.request("bze.v1.rewards.Msg", "ExitStaking", data);
    return promise.then((data) =>
      MsgExitStakingResponse.decode(new Reader(data))
    );
  }
}

interface Rpc {
  request(
    service: string,
    method: string,
    data: Uint8Array
  ): Promise<Uint8Array>;
}

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
