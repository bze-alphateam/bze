/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.burner.v1";

export interface Raffle {
  pot: string;
  duration: number;
  chances: number;
  ratio: string;
  endAt: number;
  winners: number;
  ticketPrice: string;
  denom: string;
}

export interface RaffleDeleteHook {
  denom: string;
  endAt: number;
}

const baseRaffle: object = {
  pot: "",
  duration: 0,
  chances: 0,
  ratio: "",
  endAt: 0,
  winners: 0,
  ticketPrice: "",
  denom: "",
};

export const Raffle = {
  encode(message: Raffle, writer: Writer = Writer.create()): Writer {
    if (message.pot !== "") {
      writer.uint32(10).string(message.pot);
    }
    if (message.duration !== 0) {
      writer.uint32(16).uint64(message.duration);
    }
    if (message.chances !== 0) {
      writer.uint32(24).uint64(message.chances);
    }
    if (message.ratio !== "") {
      writer.uint32(34).string(message.ratio);
    }
    if (message.endAt !== 0) {
      writer.uint32(40).uint64(message.endAt);
    }
    if (message.winners !== 0) {
      writer.uint32(48).uint64(message.winners);
    }
    if (message.ticketPrice !== "") {
      writer.uint32(58).string(message.ticketPrice);
    }
    if (message.denom !== "") {
      writer.uint32(66).string(message.denom);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): Raffle {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseRaffle } as Raffle;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pot = reader.string();
          break;
        case 2:
          message.duration = longToNumber(reader.uint64() as Long);
          break;
        case 3:
          message.chances = longToNumber(reader.uint64() as Long);
          break;
        case 4:
          message.ratio = reader.string();
          break;
        case 5:
          message.endAt = longToNumber(reader.uint64() as Long);
          break;
        case 6:
          message.winners = longToNumber(reader.uint64() as Long);
          break;
        case 7:
          message.ticketPrice = reader.string();
          break;
        case 8:
          message.denom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Raffle {
    const message = { ...baseRaffle } as Raffle;
    if (object.pot !== undefined && object.pot !== null) {
      message.pot = String(object.pot);
    } else {
      message.pot = "";
    }
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = Number(object.duration);
    } else {
      message.duration = 0;
    }
    if (object.chances !== undefined && object.chances !== null) {
      message.chances = Number(object.chances);
    } else {
      message.chances = 0;
    }
    if (object.ratio !== undefined && object.ratio !== null) {
      message.ratio = String(object.ratio);
    } else {
      message.ratio = "";
    }
    if (object.endAt !== undefined && object.endAt !== null) {
      message.endAt = Number(object.endAt);
    } else {
      message.endAt = 0;
    }
    if (object.winners !== undefined && object.winners !== null) {
      message.winners = Number(object.winners);
    } else {
      message.winners = 0;
    }
    if (object.ticketPrice !== undefined && object.ticketPrice !== null) {
      message.ticketPrice = String(object.ticketPrice);
    } else {
      message.ticketPrice = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    return message;
  },

  toJSON(message: Raffle): unknown {
    const obj: any = {};
    message.pot !== undefined && (obj.pot = message.pot);
    message.duration !== undefined && (obj.duration = message.duration);
    message.chances !== undefined && (obj.chances = message.chances);
    message.ratio !== undefined && (obj.ratio = message.ratio);
    message.endAt !== undefined && (obj.endAt = message.endAt);
    message.winners !== undefined && (obj.winners = message.winners);
    message.ticketPrice !== undefined &&
      (obj.ticketPrice = message.ticketPrice);
    message.denom !== undefined && (obj.denom = message.denom);
    return obj;
  },

  fromPartial(object: DeepPartial<Raffle>): Raffle {
    const message = { ...baseRaffle } as Raffle;
    if (object.pot !== undefined && object.pot !== null) {
      message.pot = object.pot;
    } else {
      message.pot = "";
    }
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = object.duration;
    } else {
      message.duration = 0;
    }
    if (object.chances !== undefined && object.chances !== null) {
      message.chances = object.chances;
    } else {
      message.chances = 0;
    }
    if (object.ratio !== undefined && object.ratio !== null) {
      message.ratio = object.ratio;
    } else {
      message.ratio = "";
    }
    if (object.endAt !== undefined && object.endAt !== null) {
      message.endAt = object.endAt;
    } else {
      message.endAt = 0;
    }
    if (object.winners !== undefined && object.winners !== null) {
      message.winners = object.winners;
    } else {
      message.winners = 0;
    }
    if (object.ticketPrice !== undefined && object.ticketPrice !== null) {
      message.ticketPrice = object.ticketPrice;
    } else {
      message.ticketPrice = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    return message;
  },
};

const baseRaffleDeleteHook: object = { denom: "", endAt: 0 };

export const RaffleDeleteHook = {
  encode(message: RaffleDeleteHook, writer: Writer = Writer.create()): Writer {
    if (message.denom !== "") {
      writer.uint32(10).string(message.denom);
    }
    if (message.endAt !== 0) {
      writer.uint32(16).uint64(message.endAt);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): RaffleDeleteHook {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseRaffleDeleteHook } as RaffleDeleteHook;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.denom = reader.string();
          break;
        case 2:
          message.endAt = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RaffleDeleteHook {
    const message = { ...baseRaffleDeleteHook } as RaffleDeleteHook;
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    if (object.endAt !== undefined && object.endAt !== null) {
      message.endAt = Number(object.endAt);
    } else {
      message.endAt = 0;
    }
    return message;
  },

  toJSON(message: RaffleDeleteHook): unknown {
    const obj: any = {};
    message.denom !== undefined && (obj.denom = message.denom);
    message.endAt !== undefined && (obj.endAt = message.endAt);
    return obj;
  },

  fromPartial(object: DeepPartial<RaffleDeleteHook>): RaffleDeleteHook {
    const message = { ...baseRaffleDeleteHook } as RaffleDeleteHook;
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    if (object.endAt !== undefined && object.endAt !== null) {
      message.endAt = object.endAt;
    } else {
      message.endAt = 0;
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
