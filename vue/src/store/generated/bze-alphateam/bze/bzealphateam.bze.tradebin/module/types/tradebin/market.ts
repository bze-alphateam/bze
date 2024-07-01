/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bzealphateam.bze.tradebin";

export interface Market {
  asset1: string;
  asset2: string;
  creator: string;
}

const baseMarket: object = { asset1: "", asset2: "", creator: "" };

export const Market = {
  encode(message: Market, writer: Writer = Writer.create()): Writer {
    if (message.asset1 !== "") {
      writer.uint32(10).string(message.asset1);
    }
    if (message.asset2 !== "") {
      writer.uint32(18).string(message.asset2);
    }
    if (message.creator !== "") {
      writer.uint32(26).string(message.creator);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): Market {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMarket } as Market;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.asset1 = reader.string();
          break;
        case 2:
          message.asset2 = reader.string();
          break;
        case 3:
          message.creator = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Market {
    const message = { ...baseMarket } as Market;
    if (object.asset1 !== undefined && object.asset1 !== null) {
      message.asset1 = String(object.asset1);
    } else {
      message.asset1 = "";
    }
    if (object.asset2 !== undefined && object.asset2 !== null) {
      message.asset2 = String(object.asset2);
    } else {
      message.asset2 = "";
    }
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    return message;
  },

  toJSON(message: Market): unknown {
    const obj: any = {};
    message.asset1 !== undefined && (obj.asset1 = message.asset1);
    message.asset2 !== undefined && (obj.asset2 = message.asset2);
    message.creator !== undefined && (obj.creator = message.creator);
    return obj;
  },

  fromPartial(object: DeepPartial<Market>): Market {
    const message = { ...baseMarket } as Market;
    if (object.asset1 !== undefined && object.asset1 !== null) {
      message.asset1 = object.asset1;
    } else {
      message.asset1 = "";
    }
    if (object.asset2 !== undefined && object.asset2 !== null) {
      message.asset2 = object.asset2;
    } else {
      message.asset2 = "";
    }
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
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
