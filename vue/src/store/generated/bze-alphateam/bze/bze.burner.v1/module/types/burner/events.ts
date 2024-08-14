/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.burner.v1";

export interface CoinsBurnedEvent {
  burned: string;
}

export interface FundBurnerEvent {
  from: string;
  amount: string;
}

export interface RaffleWinnerEvent {
  denom: string;
  winner: string;
  amount: string;
}

export interface RaffleFinishedEvent {
  denom: string;
}

const baseCoinsBurnedEvent: object = { burned: "" };

export const CoinsBurnedEvent = {
  encode(message: CoinsBurnedEvent, writer: Writer = Writer.create()): Writer {
    if (message.burned !== "") {
      writer.uint32(10).string(message.burned);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): CoinsBurnedEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseCoinsBurnedEvent } as CoinsBurnedEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.burned = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CoinsBurnedEvent {
    const message = { ...baseCoinsBurnedEvent } as CoinsBurnedEvent;
    if (object.burned !== undefined && object.burned !== null) {
      message.burned = String(object.burned);
    } else {
      message.burned = "";
    }
    return message;
  },

  toJSON(message: CoinsBurnedEvent): unknown {
    const obj: any = {};
    message.burned !== undefined && (obj.burned = message.burned);
    return obj;
  },

  fromPartial(object: DeepPartial<CoinsBurnedEvent>): CoinsBurnedEvent {
    const message = { ...baseCoinsBurnedEvent } as CoinsBurnedEvent;
    if (object.burned !== undefined && object.burned !== null) {
      message.burned = object.burned;
    } else {
      message.burned = "";
    }
    return message;
  },
};

const baseFundBurnerEvent: object = { from: "", amount: "" };

export const FundBurnerEvent = {
  encode(message: FundBurnerEvent, writer: Writer = Writer.create()): Writer {
    if (message.from !== "") {
      writer.uint32(10).string(message.from);
    }
    if (message.amount !== "") {
      writer.uint32(18).string(message.amount);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): FundBurnerEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseFundBurnerEvent } as FundBurnerEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.from = reader.string();
          break;
        case 2:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): FundBurnerEvent {
    const message = { ...baseFundBurnerEvent } as FundBurnerEvent;
    if (object.from !== undefined && object.from !== null) {
      message.from = String(object.from);
    } else {
      message.from = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    return message;
  },

  toJSON(message: FundBurnerEvent): unknown {
    const obj: any = {};
    message.from !== undefined && (obj.from = message.from);
    message.amount !== undefined && (obj.amount = message.amount);
    return obj;
  },

  fromPartial(object: DeepPartial<FundBurnerEvent>): FundBurnerEvent {
    const message = { ...baseFundBurnerEvent } as FundBurnerEvent;
    if (object.from !== undefined && object.from !== null) {
      message.from = object.from;
    } else {
      message.from = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    return message;
  },
};

const baseRaffleWinnerEvent: object = { denom: "", winner: "", amount: "" };

export const RaffleWinnerEvent = {
  encode(message: RaffleWinnerEvent, writer: Writer = Writer.create()): Writer {
    if (message.denom !== "") {
      writer.uint32(10).string(message.denom);
    }
    if (message.winner !== "") {
      writer.uint32(18).string(message.winner);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): RaffleWinnerEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseRaffleWinnerEvent } as RaffleWinnerEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.denom = reader.string();
          break;
        case 2:
          message.winner = reader.string();
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

  fromJSON(object: any): RaffleWinnerEvent {
    const message = { ...baseRaffleWinnerEvent } as RaffleWinnerEvent;
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    if (object.winner !== undefined && object.winner !== null) {
      message.winner = String(object.winner);
    } else {
      message.winner = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    return message;
  },

  toJSON(message: RaffleWinnerEvent): unknown {
    const obj: any = {};
    message.denom !== undefined && (obj.denom = message.denom);
    message.winner !== undefined && (obj.winner = message.winner);
    message.amount !== undefined && (obj.amount = message.amount);
    return obj;
  },

  fromPartial(object: DeepPartial<RaffleWinnerEvent>): RaffleWinnerEvent {
    const message = { ...baseRaffleWinnerEvent } as RaffleWinnerEvent;
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    if (object.winner !== undefined && object.winner !== null) {
      message.winner = object.winner;
    } else {
      message.winner = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    return message;
  },
};

const baseRaffleFinishedEvent: object = { denom: "" };

export const RaffleFinishedEvent = {
  encode(
    message: RaffleFinishedEvent,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.denom !== "") {
      writer.uint32(10).string(message.denom);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): RaffleFinishedEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseRaffleFinishedEvent } as RaffleFinishedEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.denom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RaffleFinishedEvent {
    const message = { ...baseRaffleFinishedEvent } as RaffleFinishedEvent;
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    return message;
  },

  toJSON(message: RaffleFinishedEvent): unknown {
    const obj: any = {};
    message.denom !== undefined && (obj.denom = message.denom);
    return obj;
  },

  fromPartial(object: DeepPartial<RaffleFinishedEvent>): RaffleFinishedEvent {
    const message = { ...baseRaffleFinishedEvent } as RaffleFinishedEvent;
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
