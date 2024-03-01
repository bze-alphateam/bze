/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.tradebin.v1";

export interface QueueMessage {
  message_id: string;
  market_id: string;
  message_type: string;
  amount: number;
  price: string;
  created_at: number;
  order_id: string;
  order_type: string;
}

const baseQueueMessage: object = {
  message_id: "",
  market_id: "",
  message_type: "",
  amount: 0,
  price: "",
  created_at: 0,
  order_id: "",
  order_type: "",
};

export const QueueMessage = {
  encode(message: QueueMessage, writer: Writer = Writer.create()): Writer {
    if (message.message_id !== "") {
      writer.uint32(10).string(message.message_id);
    }
    if (message.market_id !== "") {
      writer.uint32(18).string(message.market_id);
    }
    if (message.message_type !== "") {
      writer.uint32(26).string(message.message_type);
    }
    if (message.amount !== 0) {
      writer.uint32(32).int64(message.amount);
    }
    if (message.price !== "") {
      writer.uint32(42).string(message.price);
    }
    if (message.created_at !== 0) {
      writer.uint32(48).int64(message.created_at);
    }
    if (message.order_id !== "") {
      writer.uint32(58).string(message.order_id);
    }
    if (message.order_type !== "") {
      writer.uint32(66).string(message.order_type);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueueMessage {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueueMessage } as QueueMessage;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.message_id = reader.string();
          break;
        case 2:
          message.market_id = reader.string();
          break;
        case 3:
          message.message_type = reader.string();
          break;
        case 4:
          message.amount = longToNumber(reader.int64() as Long);
          break;
        case 5:
          message.price = reader.string();
          break;
        case 6:
          message.created_at = longToNumber(reader.int64() as Long);
          break;
        case 7:
          message.order_id = reader.string();
          break;
        case 8:
          message.order_type = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueueMessage {
    const message = { ...baseQueueMessage } as QueueMessage;
    if (object.message_id !== undefined && object.message_id !== null) {
      message.message_id = String(object.message_id);
    } else {
      message.message_id = "";
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = String(object.market_id);
    } else {
      message.market_id = "";
    }
    if (object.message_type !== undefined && object.message_type !== null) {
      message.message_type = String(object.message_type);
    } else {
      message.message_type = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = Number(object.amount);
    } else {
      message.amount = 0;
    }
    if (object.price !== undefined && object.price !== null) {
      message.price = String(object.price);
    } else {
      message.price = "";
    }
    if (object.created_at !== undefined && object.created_at !== null) {
      message.created_at = Number(object.created_at);
    } else {
      message.created_at = 0;
    }
    if (object.order_id !== undefined && object.order_id !== null) {
      message.order_id = String(object.order_id);
    } else {
      message.order_id = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = String(object.order_type);
    } else {
      message.order_type = "";
    }
    return message;
  },

  toJSON(message: QueueMessage): unknown {
    const obj: any = {};
    message.message_id !== undefined && (obj.message_id = message.message_id);
    message.market_id !== undefined && (obj.market_id = message.market_id);
    message.message_type !== undefined &&
      (obj.message_type = message.message_type);
    message.amount !== undefined && (obj.amount = message.amount);
    message.price !== undefined && (obj.price = message.price);
    message.created_at !== undefined && (obj.created_at = message.created_at);
    message.order_id !== undefined && (obj.order_id = message.order_id);
    message.order_type !== undefined && (obj.order_type = message.order_type);
    return obj;
  },

  fromPartial(object: DeepPartial<QueueMessage>): QueueMessage {
    const message = { ...baseQueueMessage } as QueueMessage;
    if (object.message_id !== undefined && object.message_id !== null) {
      message.message_id = object.message_id;
    } else {
      message.message_id = "";
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = object.market_id;
    } else {
      message.market_id = "";
    }
    if (object.message_type !== undefined && object.message_type !== null) {
      message.message_type = object.message_type;
    } else {
      message.message_type = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = 0;
    }
    if (object.price !== undefined && object.price !== null) {
      message.price = object.price;
    } else {
      message.price = "";
    }
    if (object.created_at !== undefined && object.created_at !== null) {
      message.created_at = object.created_at;
    } else {
      message.created_at = 0;
    }
    if (object.order_id !== undefined && object.order_id !== null) {
      message.order_id = object.order_id;
    } else {
      message.order_id = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = object.order_type;
    } else {
      message.order_type = "";
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
