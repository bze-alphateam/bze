/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.v1.rewards";

export interface TradingReward {
  rewardId: string;
  prizeAmount: string;
  prizeDenom: string;
  duration: number;
  marketId: string;
  slots: number;
  expireAt: number;
}

export interface TradingRewardExpiration {
  rewardId: string;
  expireAt: number;
}

export interface TradingRewardLeaderboard {
  rewardId: string;
  list: TradingRewardLeaderboardEntry[];
}

export interface TradingRewardLeaderboardEntry {
  amount: string;
  address: string;
  createdAt: number;
}

export interface TradingRewardCandidate {
  rewardId: string;
  amount: string;
  address: string;
}

export interface MarketIdTradingRewardId {
  rewardId: string;
  marketId: string;
}

const baseTradingReward: object = {
  rewardId: "",
  prizeAmount: "",
  prizeDenom: "",
  duration: 0,
  marketId: "",
  slots: 0,
  expireAt: 0,
};

export const TradingReward = {
  encode(message: TradingReward, writer: Writer = Writer.create()): Writer {
    if (message.rewardId !== "") {
      writer.uint32(10).string(message.rewardId);
    }
    if (message.prizeAmount !== "") {
      writer.uint32(18).string(message.prizeAmount);
    }
    if (message.prizeDenom !== "") {
      writer.uint32(26).string(message.prizeDenom);
    }
    if (message.duration !== 0) {
      writer.uint32(32).uint32(message.duration);
    }
    if (message.marketId !== "") {
      writer.uint32(42).string(message.marketId);
    }
    if (message.slots !== 0) {
      writer.uint32(48).uint32(message.slots);
    }
    if (message.expireAt !== 0) {
      writer.uint32(56).uint32(message.expireAt);
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
          message.rewardId = reader.string();
          break;
        case 2:
          message.prizeAmount = reader.string();
          break;
        case 3:
          message.prizeDenom = reader.string();
          break;
        case 4:
          message.duration = reader.uint32();
          break;
        case 5:
          message.marketId = reader.string();
          break;
        case 6:
          message.slots = reader.uint32();
          break;
        case 7:
          message.expireAt = reader.uint32();
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
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = Number(object.duration);
    } else {
      message.duration = 0;
    }
    if (object.marketId !== undefined && object.marketId !== null) {
      message.marketId = String(object.marketId);
    } else {
      message.marketId = "";
    }
    if (object.slots !== undefined && object.slots !== null) {
      message.slots = Number(object.slots);
    } else {
      message.slots = 0;
    }
    if (object.expireAt !== undefined && object.expireAt !== null) {
      message.expireAt = Number(object.expireAt);
    } else {
      message.expireAt = 0;
    }
    return message;
  },

  toJSON(message: TradingReward): unknown {
    const obj: any = {};
    message.rewardId !== undefined && (obj.rewardId = message.rewardId);
    message.prizeAmount !== undefined &&
      (obj.prizeAmount = message.prizeAmount);
    message.prizeDenom !== undefined && (obj.prizeDenom = message.prizeDenom);
    message.duration !== undefined && (obj.duration = message.duration);
    message.marketId !== undefined && (obj.marketId = message.marketId);
    message.slots !== undefined && (obj.slots = message.slots);
    message.expireAt !== undefined && (obj.expireAt = message.expireAt);
    return obj;
  },

  fromPartial(object: DeepPartial<TradingReward>): TradingReward {
    const message = { ...baseTradingReward } as TradingReward;
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
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = object.duration;
    } else {
      message.duration = 0;
    }
    if (object.marketId !== undefined && object.marketId !== null) {
      message.marketId = object.marketId;
    } else {
      message.marketId = "";
    }
    if (object.slots !== undefined && object.slots !== null) {
      message.slots = object.slots;
    } else {
      message.slots = 0;
    }
    if (object.expireAt !== undefined && object.expireAt !== null) {
      message.expireAt = object.expireAt;
    } else {
      message.expireAt = 0;
    }
    return message;
  },
};

const baseTradingRewardExpiration: object = { rewardId: "", expireAt: 0 };

export const TradingRewardExpiration = {
  encode(
    message: TradingRewardExpiration,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.rewardId !== "") {
      writer.uint32(10).string(message.rewardId);
    }
    if (message.expireAt !== 0) {
      writer.uint32(16).uint32(message.expireAt);
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
          message.rewardId = reader.string();
          break;
        case 2:
          message.expireAt = reader.uint32();
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
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = String(object.rewardId);
    } else {
      message.rewardId = "";
    }
    if (object.expireAt !== undefined && object.expireAt !== null) {
      message.expireAt = Number(object.expireAt);
    } else {
      message.expireAt = 0;
    }
    return message;
  },

  toJSON(message: TradingRewardExpiration): unknown {
    const obj: any = {};
    message.rewardId !== undefined && (obj.rewardId = message.rewardId);
    message.expireAt !== undefined && (obj.expireAt = message.expireAt);
    return obj;
  },

  fromPartial(
    object: DeepPartial<TradingRewardExpiration>
  ): TradingRewardExpiration {
    const message = {
      ...baseTradingRewardExpiration,
    } as TradingRewardExpiration;
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = object.rewardId;
    } else {
      message.rewardId = "";
    }
    if (object.expireAt !== undefined && object.expireAt !== null) {
      message.expireAt = object.expireAt;
    } else {
      message.expireAt = 0;
    }
    return message;
  },
};

const baseTradingRewardLeaderboard: object = { rewardId: "" };

export const TradingRewardLeaderboard = {
  encode(
    message: TradingRewardLeaderboard,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.rewardId !== "") {
      writer.uint32(10).string(message.rewardId);
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
          message.rewardId = reader.string();
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
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = String(object.rewardId);
    } else {
      message.rewardId = "";
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
    message.rewardId !== undefined && (obj.rewardId = message.rewardId);
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
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = object.rewardId;
    } else {
      message.rewardId = "";
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
  createdAt: 0,
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
    if (message.createdAt !== 0) {
      writer.uint32(24).int64(message.createdAt);
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
          message.createdAt = longToNumber(reader.int64() as Long);
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
    if (object.createdAt !== undefined && object.createdAt !== null) {
      message.createdAt = Number(object.createdAt);
    } else {
      message.createdAt = 0;
    }
    return message;
  },

  toJSON(message: TradingRewardLeaderboardEntry): unknown {
    const obj: any = {};
    message.amount !== undefined && (obj.amount = message.amount);
    message.address !== undefined && (obj.address = message.address);
    message.createdAt !== undefined && (obj.createdAt = message.createdAt);
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
    if (object.createdAt !== undefined && object.createdAt !== null) {
      message.createdAt = object.createdAt;
    } else {
      message.createdAt = 0;
    }
    return message;
  },
};

const baseTradingRewardCandidate: object = {
  rewardId: "",
  amount: "",
  address: "",
};

export const TradingRewardCandidate = {
  encode(
    message: TradingRewardCandidate,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.rewardId !== "") {
      writer.uint32(10).string(message.rewardId);
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
          message.rewardId = reader.string();
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
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    return message;
  },

  toJSON(message: TradingRewardCandidate): unknown {
    const obj: any = {};
    message.rewardId !== undefined && (obj.rewardId = message.rewardId);
    message.amount !== undefined && (obj.amount = message.amount);
    message.address !== undefined && (obj.address = message.address);
    return obj;
  },

  fromPartial(
    object: DeepPartial<TradingRewardCandidate>
  ): TradingRewardCandidate {
    const message = { ...baseTradingRewardCandidate } as TradingRewardCandidate;
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
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    return message;
  },
};

const baseMarketIdTradingRewardId: object = { rewardId: "", marketId: "" };

export const MarketIdTradingRewardId = {
  encode(
    message: MarketIdTradingRewardId,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.rewardId !== "") {
      writer.uint32(10).string(message.rewardId);
    }
    if (message.marketId !== "") {
      writer.uint32(18).string(message.marketId);
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
          message.rewardId = reader.string();
          break;
        case 2:
          message.marketId = reader.string();
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
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = String(object.rewardId);
    } else {
      message.rewardId = "";
    }
    if (object.marketId !== undefined && object.marketId !== null) {
      message.marketId = String(object.marketId);
    } else {
      message.marketId = "";
    }
    return message;
  },

  toJSON(message: MarketIdTradingRewardId): unknown {
    const obj: any = {};
    message.rewardId !== undefined && (obj.rewardId = message.rewardId);
    message.marketId !== undefined && (obj.marketId = message.marketId);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MarketIdTradingRewardId>
  ): MarketIdTradingRewardId {
    const message = {
      ...baseMarketIdTradingRewardId,
    } as MarketIdTradingRewardId;
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = object.rewardId;
    } else {
      message.rewardId = "";
    }
    if (object.marketId !== undefined && object.marketId !== null) {
      message.marketId = object.marketId;
    } else {
      message.marketId = "";
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
