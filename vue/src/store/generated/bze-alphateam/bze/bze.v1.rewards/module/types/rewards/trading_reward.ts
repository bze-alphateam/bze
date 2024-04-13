/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.v1.rewards";

export interface TradingReward {
  reward_id: string;
  prize_amount: string;
  prize_denom: string;
  duration: number;
  market_id: string;
  slots: number;
  expire_at: number;
}

export interface TradingRewardExpiration {
  reward_id: string;
  expire_at: number;
}

export interface TradingRewardLeaderboard {
  reward_id: string;
  list: TradingRewardLeaderboardEntry[];
}

export interface TradingRewardLeaderboardEntry {
  amount: string;
  address: string;
  created_at: number;
}

export interface TradingRewardCandidate {
  reward_id: string;
  amount: string;
  address: string;
}

export interface MarketIdTradingRewardId {
  reward_id: string;
  market_id: string;
}

const baseTradingReward: object = {
  reward_id: "",
  prize_amount: "",
  prize_denom: "",
  duration: 0,
  market_id: "",
  slots: 0,
  expire_at: 0,
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
    if (message.expire_at !== 0) {
      writer.uint32(56).uint32(message.expire_at);
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
        case 7:
          message.expire_at = reader.uint32();
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
    if (object.expire_at !== undefined && object.expire_at !== null) {
      message.expire_at = Number(object.expire_at);
    } else {
      message.expire_at = 0;
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
    message.expire_at !== undefined && (obj.expire_at = message.expire_at);
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
    if (object.expire_at !== undefined && object.expire_at !== null) {
      message.expire_at = object.expire_at;
    } else {
      message.expire_at = 0;
    }
    return message;
  },
};

const baseTradingRewardExpiration: object = { reward_id: "", expire_at: 0 };

export const TradingRewardExpiration = {
  encode(
    message: TradingRewardExpiration,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.reward_id !== "") {
      writer.uint32(10).string(message.reward_id);
    }
    if (message.expire_at !== 0) {
      writer.uint32(16).uint32(message.expire_at);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): TradingRewardExpiration {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseTradingRewardExpiration,
    } as TradingRewardExpiration;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reward_id = reader.string();
          break;
        case 2:
          message.expire_at = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TradingRewardExpiration {
    const message = {
      ...baseTradingRewardExpiration,
    } as TradingRewardExpiration;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = String(object.reward_id);
    } else {
      message.reward_id = "";
    }
    if (object.expire_at !== undefined && object.expire_at !== null) {
      message.expire_at = Number(object.expire_at);
    } else {
      message.expire_at = 0;
    }
    return message;
  },

  toJSON(message: TradingRewardExpiration): unknown {
    const obj: any = {};
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    message.expire_at !== undefined && (obj.expire_at = message.expire_at);
    return obj;
  },

  fromPartial(
    object: DeepPartial<TradingRewardExpiration>
  ): TradingRewardExpiration {
    const message = {
      ...baseTradingRewardExpiration,
    } as TradingRewardExpiration;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = object.reward_id;
    } else {
      message.reward_id = "";
    }
    if (object.expire_at !== undefined && object.expire_at !== null) {
      message.expire_at = object.expire_at;
    } else {
      message.expire_at = 0;
    }
    return message;
  },
};

const baseTradingRewardLeaderboard: object = { reward_id: "" };

export const TradingRewardLeaderboard = {
  encode(
    message: TradingRewardLeaderboard,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.reward_id !== "") {
      writer.uint32(10).string(message.reward_id);
    }
    for (const v of message.list) {
      TradingRewardLeaderboardEntry.encode(
        v!,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): TradingRewardLeaderboard {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseTradingRewardLeaderboard,
    } as TradingRewardLeaderboard;
    message.list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reward_id = reader.string();
          break;
        case 2:
          message.list.push(
            TradingRewardLeaderboardEntry.decode(reader, reader.uint32())
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TradingRewardLeaderboard {
    const message = {
      ...baseTradingRewardLeaderboard,
    } as TradingRewardLeaderboard;
    message.list = [];
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = String(object.reward_id);
    } else {
      message.reward_id = "";
    }
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(TradingRewardLeaderboardEntry.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: TradingRewardLeaderboard): unknown {
    const obj: any = {};
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    if (message.list) {
      obj.list = message.list.map((e) =>
        e ? TradingRewardLeaderboardEntry.toJSON(e) : undefined
      );
    } else {
      obj.list = [];
    }
    return obj;
  },

  fromPartial(
    object: DeepPartial<TradingRewardLeaderboard>
  ): TradingRewardLeaderboard {
    const message = {
      ...baseTradingRewardLeaderboard,
    } as TradingRewardLeaderboard;
    message.list = [];
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = object.reward_id;
    } else {
      message.reward_id = "";
    }
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(TradingRewardLeaderboardEntry.fromPartial(e));
      }
    }
    return message;
  },
};

const baseTradingRewardLeaderboardEntry: object = {
  amount: "",
  address: "",
  created_at: 0,
};

export const TradingRewardLeaderboardEntry = {
  encode(
    message: TradingRewardLeaderboardEntry,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.amount !== "") {
      writer.uint32(10).string(message.amount);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    if (message.created_at !== 0) {
      writer.uint32(24).int64(message.created_at);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): TradingRewardLeaderboardEntry {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseTradingRewardLeaderboardEntry,
    } as TradingRewardLeaderboardEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.amount = reader.string();
          break;
        case 2:
          message.address = reader.string();
          break;
        case 3:
          message.created_at = longToNumber(reader.int64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TradingRewardLeaderboardEntry {
    const message = {
      ...baseTradingRewardLeaderboardEntry,
    } as TradingRewardLeaderboardEntry;
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    if (object.created_at !== undefined && object.created_at !== null) {
      message.created_at = Number(object.created_at);
    } else {
      message.created_at = 0;
    }
    return message;
  },

  toJSON(message: TradingRewardLeaderboardEntry): unknown {
    const obj: any = {};
    message.amount !== undefined && (obj.amount = message.amount);
    message.address !== undefined && (obj.address = message.address);
    message.created_at !== undefined && (obj.created_at = message.created_at);
    return obj;
  },

  fromPartial(
    object: DeepPartial<TradingRewardLeaderboardEntry>
  ): TradingRewardLeaderboardEntry {
    const message = {
      ...baseTradingRewardLeaderboardEntry,
    } as TradingRewardLeaderboardEntry;
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    if (object.created_at !== undefined && object.created_at !== null) {
      message.created_at = object.created_at;
    } else {
      message.created_at = 0;
    }
    return message;
  },
};

const baseTradingRewardCandidate: object = {
  reward_id: "",
  amount: "",
  address: "",
};

export const TradingRewardCandidate = {
  encode(
    message: TradingRewardCandidate,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.reward_id !== "") {
      writer.uint32(10).string(message.reward_id);
    }
    if (message.amount !== "") {
      writer.uint32(18).string(message.amount);
    }
    if (message.address !== "") {
      writer.uint32(26).string(message.address);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): TradingRewardCandidate {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseTradingRewardCandidate } as TradingRewardCandidate;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reward_id = reader.string();
          break;
        case 2:
          message.amount = reader.string();
          break;
        case 3:
          message.address = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TradingRewardCandidate {
    const message = { ...baseTradingRewardCandidate } as TradingRewardCandidate;
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
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    return message;
  },

  toJSON(message: TradingRewardCandidate): unknown {
    const obj: any = {};
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    message.amount !== undefined && (obj.amount = message.amount);
    message.address !== undefined && (obj.address = message.address);
    return obj;
  },

  fromPartial(
    object: DeepPartial<TradingRewardCandidate>
  ): TradingRewardCandidate {
    const message = { ...baseTradingRewardCandidate } as TradingRewardCandidate;
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
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    return message;
  },
};

const baseMarketIdTradingRewardId: object = { reward_id: "", market_id: "" };

export const MarketIdTradingRewardId = {
  encode(
    message: MarketIdTradingRewardId,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.reward_id !== "") {
      writer.uint32(10).string(message.reward_id);
    }
    if (message.market_id !== "") {
      writer.uint32(18).string(message.market_id);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MarketIdTradingRewardId {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMarketIdTradingRewardId,
    } as MarketIdTradingRewardId;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reward_id = reader.string();
          break;
        case 2:
          message.market_id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MarketIdTradingRewardId {
    const message = {
      ...baseMarketIdTradingRewardId,
    } as MarketIdTradingRewardId;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = String(object.reward_id);
    } else {
      message.reward_id = "";
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = String(object.market_id);
    } else {
      message.market_id = "";
    }
    return message;
  },

  toJSON(message: MarketIdTradingRewardId): unknown {
    const obj: any = {};
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    message.market_id !== undefined && (obj.market_id = message.market_id);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MarketIdTradingRewardId>
  ): MarketIdTradingRewardId {
    const message = {
      ...baseMarketIdTradingRewardId,
    } as MarketIdTradingRewardId;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = object.reward_id;
    } else {
      message.reward_id = "";
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = object.market_id;
    } else {
      message.market_id = "";
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
