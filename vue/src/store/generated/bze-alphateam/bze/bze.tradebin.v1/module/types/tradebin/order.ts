/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.tradebin.v1";

export interface Order {
  id: string;
  market_id: string;
  order_type: string;
  amount: string;
  price: string;
  created_at: number;
  owner: string;
}

export interface OrderReference {
  id: string;
  market_id: string;
  order_type: string;
}

export interface AggregatedOrder {
  market_id: string;
  order_type: string;
  amount: string;
  price: string;
}

export interface HistoryOrder {
  market_id: string;
  order_type: string;
  amount: string;
  price: string;
  executed_at: number;
  maker: string;
  taker: string;
}

const baseOrder: object = {
  id: "",
  market_id: "",
  order_type: "",
  amount: "",
  price: "",
  created_at: 0,
  owner: "",
};

export const Order = {
  encode(message: Order, writer: Writer = Writer.create()): Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.market_id !== "") {
      writer.uint32(18).string(message.market_id);
    }
    if (message.order_type !== "") {
      writer.uint32(26).string(message.order_type);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    if (message.price !== "") {
      writer.uint32(42).string(message.price);
    }
    if (message.created_at !== 0) {
      writer.uint32(48).int64(message.created_at);
    }
    if (message.owner !== "") {
      writer.uint32(58).string(message.owner);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): Order {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseOrder } as Order;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.market_id = reader.string();
          break;
        case 3:
          message.order_type = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        case 5:
          message.price = reader.string();
          break;
        case 6:
          message.created_at = longToNumber(reader.int64() as Long);
          break;
        case 7:
          message.owner = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Order {
    const message = { ...baseOrder } as Order;
    if (object.id !== undefined && object.id !== null) {
      message.id = String(object.id);
    } else {
      message.id = "";
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = String(object.market_id);
    } else {
      message.market_id = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = String(object.order_type);
    } else {
      message.order_type = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
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
    if (object.owner !== undefined && object.owner !== null) {
      message.owner = String(object.owner);
    } else {
      message.owner = "";
    }
    return message;
  },

  toJSON(message: Order): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.market_id !== undefined && (obj.market_id = message.market_id);
    message.order_type !== undefined && (obj.order_type = message.order_type);
    message.amount !== undefined && (obj.amount = message.amount);
    message.price !== undefined && (obj.price = message.price);
    message.created_at !== undefined && (obj.created_at = message.created_at);
    message.owner !== undefined && (obj.owner = message.owner);
    return obj;
  },

  fromPartial(object: DeepPartial<Order>): Order {
    const message = { ...baseOrder } as Order;
    if (object.id !== undefined && object.id !== null) {
      message.id = object.id;
    } else {
      message.id = "";
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = object.market_id;
    } else {
      message.market_id = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = object.order_type;
    } else {
      message.order_type = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
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
    if (object.owner !== undefined && object.owner !== null) {
      message.owner = object.owner;
    } else {
      message.owner = "";
    }
    return message;
  },
};

const baseOrderReference: object = { id: "", market_id: "", order_type: "" };

export const OrderReference = {
  encode(message: OrderReference, writer: Writer = Writer.create()): Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.market_id !== "") {
      writer.uint32(18).string(message.market_id);
    }
    if (message.order_type !== "") {
      writer.uint32(26).string(message.order_type);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): OrderReference {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseOrderReference } as OrderReference;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.market_id = reader.string();
          break;
        case 3:
          message.order_type = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): OrderReference {
    const message = { ...baseOrderReference } as OrderReference;
    if (object.id !== undefined && object.id !== null) {
      message.id = String(object.id);
    } else {
      message.id = "";
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = String(object.market_id);
    } else {
      message.market_id = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = String(object.order_type);
    } else {
      message.order_type = "";
    }
    return message;
  },

  toJSON(message: OrderReference): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.market_id !== undefined && (obj.market_id = message.market_id);
    message.order_type !== undefined && (obj.order_type = message.order_type);
    return obj;
  },

  fromPartial(object: DeepPartial<OrderReference>): OrderReference {
    const message = { ...baseOrderReference } as OrderReference;
    if (object.id !== undefined && object.id !== null) {
      message.id = object.id;
    } else {
      message.id = "";
    }
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = object.market_id;
    } else {
      message.market_id = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = object.order_type;
    } else {
      message.order_type = "";
    }
    return message;
  },
};

const baseAggregatedOrder: object = {
  market_id: "",
  order_type: "",
  amount: "",
  price: "",
};

export const AggregatedOrder = {
  encode(message: AggregatedOrder, writer: Writer = Writer.create()): Writer {
    if (message.market_id !== "") {
      writer.uint32(10).string(message.market_id);
    }
    if (message.order_type !== "") {
      writer.uint32(18).string(message.order_type);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.price !== "") {
      writer.uint32(34).string(message.price);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): AggregatedOrder {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseAggregatedOrder } as AggregatedOrder;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.market_id = reader.string();
          break;
        case 2:
          message.order_type = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.price = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): AggregatedOrder {
    const message = { ...baseAggregatedOrder } as AggregatedOrder;
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = String(object.market_id);
    } else {
      message.market_id = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = String(object.order_type);
    } else {
      message.order_type = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    if (object.price !== undefined && object.price !== null) {
      message.price = String(object.price);
    } else {
      message.price = "";
    }
    return message;
  },

  toJSON(message: AggregatedOrder): unknown {
    const obj: any = {};
    message.market_id !== undefined && (obj.market_id = message.market_id);
    message.order_type !== undefined && (obj.order_type = message.order_type);
    message.amount !== undefined && (obj.amount = message.amount);
    message.price !== undefined && (obj.price = message.price);
    return obj;
  },

  fromPartial(object: DeepPartial<AggregatedOrder>): AggregatedOrder {
    const message = { ...baseAggregatedOrder } as AggregatedOrder;
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = object.market_id;
    } else {
      message.market_id = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = object.order_type;
    } else {
      message.order_type = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    if (object.price !== undefined && object.price !== null) {
      message.price = object.price;
    } else {
      message.price = "";
    }
    return message;
  },
};

const baseHistoryOrder: object = {
  market_id: "",
  order_type: "",
  amount: "",
  price: "",
  executed_at: 0,
  maker: "",
  taker: "",
};

export const HistoryOrder = {
  encode(message: HistoryOrder, writer: Writer = Writer.create()): Writer {
    if (message.market_id !== "") {
      writer.uint32(10).string(message.market_id);
    }
    if (message.order_type !== "") {
      writer.uint32(18).string(message.order_type);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.price !== "") {
      writer.uint32(34).string(message.price);
    }
    if (message.executed_at !== 0) {
      writer.uint32(40).int64(message.executed_at);
    }
    if (message.maker !== "") {
      writer.uint32(50).string(message.maker);
    }
    if (message.taker !== "") {
      writer.uint32(58).string(message.taker);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): HistoryOrder {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseHistoryOrder } as HistoryOrder;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.market_id = reader.string();
          break;
        case 2:
          message.order_type = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.price = reader.string();
          break;
        case 5:
          message.executed_at = longToNumber(reader.int64() as Long);
          break;
        case 6:
          message.maker = reader.string();
          break;
        case 7:
          message.taker = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): HistoryOrder {
    const message = { ...baseHistoryOrder } as HistoryOrder;
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = String(object.market_id);
    } else {
      message.market_id = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = String(object.order_type);
    } else {
      message.order_type = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    if (object.price !== undefined && object.price !== null) {
      message.price = String(object.price);
    } else {
      message.price = "";
    }
    if (object.executed_at !== undefined && object.executed_at !== null) {
      message.executed_at = Number(object.executed_at);
    } else {
      message.executed_at = 0;
    }
    if (object.maker !== undefined && object.maker !== null) {
      message.maker = String(object.maker);
    } else {
      message.maker = "";
    }
    if (object.taker !== undefined && object.taker !== null) {
      message.taker = String(object.taker);
    } else {
      message.taker = "";
    }
    return message;
  },

  toJSON(message: HistoryOrder): unknown {
    const obj: any = {};
    message.market_id !== undefined && (obj.market_id = message.market_id);
    message.order_type !== undefined && (obj.order_type = message.order_type);
    message.amount !== undefined && (obj.amount = message.amount);
    message.price !== undefined && (obj.price = message.price);
    message.executed_at !== undefined &&
      (obj.executed_at = message.executed_at);
    message.maker !== undefined && (obj.maker = message.maker);
    message.taker !== undefined && (obj.taker = message.taker);
    return obj;
  },

  fromPartial(object: DeepPartial<HistoryOrder>): HistoryOrder {
    const message = { ...baseHistoryOrder } as HistoryOrder;
    if (object.market_id !== undefined && object.market_id !== null) {
      message.market_id = object.market_id;
    } else {
      message.market_id = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = object.order_type;
    } else {
      message.order_type = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    if (object.price !== undefined && object.price !== null) {
      message.price = object.price;
    } else {
      message.price = "";
    }
    if (object.executed_at !== undefined && object.executed_at !== null) {
      message.executed_at = object.executed_at;
    } else {
      message.executed_at = 0;
    }
    if (object.maker !== undefined && object.maker !== null) {
      message.maker = object.maker;
    } else {
      message.maker = "";
    }
    if (object.taker !== undefined && object.taker !== null) {
      message.taker = object.taker;
    } else {
      message.taker = "";
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
