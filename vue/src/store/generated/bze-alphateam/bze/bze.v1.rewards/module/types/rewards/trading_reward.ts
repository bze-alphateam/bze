/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.v1.rewards";

export interface TradingReward {
  reward_id: string;
  prize_amount: string;
  prize_denom: string;
  duration: number;
  market_id: string;
  slots: number;
}

const baseTradingReward: object = {
  reward_id: "",
  prize_amount: "",
  prize_denom: "",
  duration: 0,
  market_id: "",
  slots: 0,
};

export const TradingReward = {
  encode(message: TradingReward, writer: Writer = Writer.create()): Writer {
    if (message.reward_id !== "") {
      writer.uint32(10).string(message.reward_id);
    }
    if (message.prize_amount !== "") {
      writer.uint32(18).string(message.prize_amount);
    }
    if (message.prize_denom !== "") {
      writer.uint32(26).string(message.prize_denom);
    }
    if (message.duration !== 0) {
      writer.uint32(32).uint32(message.duration);
    }
    if (message.market_id !== "") {
      writer.uint32(42).string(message.market_id);
    }
    if (message.slots !== 0) {
      writer.uint32(48).uint32(message.slots);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): TradingReward {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseTradingReward } as TradingReward;
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
          message.duration = reader.uint32();
          break;
        case 5:
          message.market_id = reader.string();
          break;
        case 6:
          message.slots = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TradingReward {
    const message = { ...baseTradingReward } as TradingReward;
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
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = Number(object.duration);
    } else {
      message.duration = 0;
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = String(object.market_id);
    } else {
      message.market_id = "";
    }
    if (object.slots !== undefined && object.slots !== null) {
      message.slots = Number(object.slots);
    } else {
      message.slots = 0;
    }
    return message;
  },

  toJSON(message: TradingReward): unknown {
    const obj: any = {};
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    message.prize_amount !== undefined &&
      (obj.prize_amount = message.prize_amount);
    message.prize_denom !== undefined &&
      (obj.prize_denom = message.prize_denom);
    message.duration !== undefined && (obj.duration = message.duration);
    message.market_id !== undefined && (obj.market_id = message.market_id);
    message.slots !== undefined && (obj.slots = message.slots);
    return obj;
  },

  fromPartial(object: DeepPartial<TradingReward>): TradingReward {
    const message = { ...baseTradingReward } as TradingReward;
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
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = object.duration;
    } else {
      message.duration = 0;
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = object.market_id;
    } else {
      message.market_id = "";
    }
    if (object.slots !== undefined && object.slots !== null) {
      message.slots = object.slots;
    } else {
      message.slots = 0;
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
