/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.burner.v1";

export interface Raffle {
  pot: string;
  duration: number;
  chances: number;
  ratio: string;
  end_at: number;
  winners: number;
  ticket_price: string;
  denom: string;
  total_won: string;
}

export interface RaffleDeleteHook {
  denom: string;
  end_at: number;
}

export interface RaffleWinner {
  index: string;
  denom: string;
  amount: string;
  winner: string;
}

export interface RaffleParticipant {
  index: number;
  denom: string;
  participant: string;
  execute_at: number;
}

const baseRaffle: object = {
  pot: "",
  duration: 0,
  chances: 0,
  ratio: "",
  end_at: 0,
  winners: 0,
  ticket_price: "",
  denom: "",
  total_won: "",
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
    if (message.end_at !== 0) {
      writer.uint32(40).uint64(message.end_at);
    }
    if (message.winners !== 0) {
      writer.uint32(48).uint64(message.winners);
    }
    if (message.ticket_price !== "") {
      writer.uint32(58).string(message.ticket_price);
    }
    if (message.denom !== "") {
      writer.uint32(66).string(message.denom);
    }
    if (message.total_won !== "") {
      writer.uint32(74).string(message.total_won);
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
          message.end_at = longToNumber(reader.uint64() as Long);
          break;
        case 6:
          message.winners = longToNumber(reader.uint64() as Long);
          break;
        case 7:
          message.ticket_price = reader.string();
          break;
        case 8:
          message.denom = reader.string();
          break;
        case 9:
          message.total_won = reader.string();
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
    if (object.end_at !== undefined && object.end_at !== null) {
      message.end_at = Number(object.end_at);
    } else {
      message.end_at = 0;
    }
    if (object.winners !== undefined && object.winners !== null) {
      message.winners = Number(object.winners);
    } else {
      message.winners = 0;
    }
    if (object.ticket_price !== undefined && object.ticket_price !== null) {
      message.ticket_price = String(object.ticket_price);
    } else {
      message.ticket_price = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    if (object.total_won !== undefined && object.total_won !== null) {
      message.total_won = String(object.total_won);
    } else {
      message.total_won = "";
    }
    return message;
  },

  toJSON(message: Raffle): unknown {
    const obj: any = {};
    message.pot !== undefined && (obj.pot = message.pot);
    message.duration !== undefined && (obj.duration = message.duration);
    message.chances !== undefined && (obj.chances = message.chances);
    message.ratio !== undefined && (obj.ratio = message.ratio);
    message.end_at !== undefined && (obj.end_at = message.end_at);
    message.winners !== undefined && (obj.winners = message.winners);
    message.ticket_price !== undefined &&
      (obj.ticket_price = message.ticket_price);
    message.denom !== undefined && (obj.denom = message.denom);
    message.total_won !== undefined && (obj.total_won = message.total_won);
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
    if (object.end_at !== undefined && object.end_at !== null) {
      message.end_at = object.end_at;
    } else {
      message.end_at = 0;
    }
    if (object.winners !== undefined && object.winners !== null) {
      message.winners = object.winners;
    } else {
      message.winners = 0;
    }
    if (object.ticket_price !== undefined && object.ticket_price !== null) {
      message.ticket_price = object.ticket_price;
    } else {
      message.ticket_price = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    if (object.total_won !== undefined && object.total_won !== null) {
      message.total_won = object.total_won;
    } else {
      message.total_won = "";
    }
    return message;
  },
};

const baseRaffleDeleteHook: object = { denom: "", end_at: 0 };

export const RaffleDeleteHook = {
  encode(message: RaffleDeleteHook, writer: Writer = Writer.create()): Writer {
    if (message.denom !== "") {
      writer.uint32(10).string(message.denom);
    }
    if (message.end_at !== 0) {
      writer.uint32(16).uint64(message.end_at);
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
          message.end_at = longToNumber(reader.uint64() as Long);
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
    if (object.end_at !== undefined && object.end_at !== null) {
      message.end_at = Number(object.end_at);
    } else {
      message.end_at = 0;
    }
    return message;
  },

  toJSON(message: RaffleDeleteHook): unknown {
    const obj: any = {};
    message.denom !== undefined && (obj.denom = message.denom);
    message.end_at !== undefined && (obj.end_at = message.end_at);
    return obj;
  },

  fromPartial(object: DeepPartial<RaffleDeleteHook>): RaffleDeleteHook {
    const message = { ...baseRaffleDeleteHook } as RaffleDeleteHook;
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    if (object.end_at !== undefined && object.end_at !== null) {
      message.end_at = object.end_at;
    } else {
      message.end_at = 0;
    }
    return message;
  },
};

const baseRaffleWinner: object = {
  index: "",
  denom: "",
  amount: "",
  winner: "",
};

export const RaffleWinner = {
  encode(message: RaffleWinner, writer: Writer = Writer.create()): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.denom !== "") {
      writer.uint32(18).string(message.denom);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.winner !== "") {
      writer.uint32(34).string(message.winner);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): RaffleWinner {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseRaffleWinner } as RaffleWinner;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        case 2:
          message.denom = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.winner = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RaffleWinner {
    const message = { ...baseRaffleWinner } as RaffleWinner;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    if (object.winner !== undefined && object.winner !== null) {
      message.winner = String(object.winner);
    } else {
      message.winner = "";
    }
    return message;
  },

  toJSON(message: RaffleWinner): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.denom !== undefined && (obj.denom = message.denom);
    message.amount !== undefined && (obj.amount = message.amount);
    message.winner !== undefined && (obj.winner = message.winner);
    return obj;
  },

  fromPartial(object: DeepPartial<RaffleWinner>): RaffleWinner {
    const message = { ...baseRaffleWinner } as RaffleWinner;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    if (object.winner !== undefined && object.winner !== null) {
      message.winner = object.winner;
    } else {
      message.winner = "";
    }
    return message;
  },
};

const baseRaffleParticipant: object = {
  index: 0,
  denom: "",
  participant: "",
  execute_at: 0,
};

export const RaffleParticipant = {
  encode(message: RaffleParticipant, writer: Writer = Writer.create()): Writer {
    if (message.index !== 0) {
      writer.uint32(8).uint64(message.index);
    }
    if (message.denom !== "") {
      writer.uint32(18).string(message.denom);
    }
    if (message.participant !== "") {
      writer.uint32(26).string(message.participant);
    }
    if (message.execute_at !== 0) {
      writer.uint32(32).int64(message.execute_at);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): RaffleParticipant {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseRaffleParticipant } as RaffleParticipant;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = longToNumber(reader.uint64() as Long);
          break;
        case 2:
          message.denom = reader.string();
          break;
        case 3:
          message.participant = reader.string();
          break;
        case 4:
          message.execute_at = longToNumber(reader.int64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RaffleParticipant {
    const message = { ...baseRaffleParticipant } as RaffleParticipant;
    if (object.index !== undefined && object.index !== null) {
      message.index = Number(object.index);
    } else {
      message.index = 0;
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    if (object.participant !== undefined && object.participant !== null) {
      message.participant = String(object.participant);
    } else {
      message.participant = "";
    }
    if (object.execute_at !== undefined && object.execute_at !== null) {
      message.execute_at = Number(object.execute_at);
    } else {
      message.execute_at = 0;
    }
    return message;
  },

  toJSON(message: RaffleParticipant): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.denom !== undefined && (obj.denom = message.denom);
    message.participant !== undefined &&
      (obj.participant = message.participant);
    message.execute_at !== undefined && (obj.execute_at = message.execute_at);
    return obj;
  },

  fromPartial(object: DeepPartial<RaffleParticipant>): RaffleParticipant {
    const message = { ...baseRaffleParticipant } as RaffleParticipant;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = 0;
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    if (object.participant !== undefined && object.participant !== null) {
      message.participant = object.participant;
    } else {
      message.participant = "";
    }
    if (object.execute_at !== undefined && object.execute_at !== null) {
      message.execute_at = object.execute_at;
    } else {
      message.execute_at = 0;
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
