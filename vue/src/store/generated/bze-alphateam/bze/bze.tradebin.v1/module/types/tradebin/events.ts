/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.tradebin.v1";

export interface OrderCreateMessageEvent {
  creator: string;
  market_id: string;
  order_type: string;
  amount: string;
  price: string;
}

export interface OrderCancelMessageEvent {
  creator: string;
  marketId: string;
  orderId: string;
  order_type: string;
}

export interface MarketCreatedEvent {
  creator: string;
  base: string;
  quote: string;
}

export interface OrderExecutedEvent {
  id: string;
  market_id: string;
  order_type: string;
  amount: string;
  price: string;
}

export interface OrderCanceledEvent {
  id: string;
  market_id: string;
  order_type: string;
  amount: string;
  price: string;
}

export interface OrderSavedEvent {
  id: string;
  market_id: string;
  order_type: string;
  amount: string;
  price: string;
}

const baseOrderCreateMessageEvent: object = {
  creator: "",
  market_id: "",
  order_type: "",
  amount: "",
  price: "",
};

export const OrderCreateMessageEvent = {
  encode(
    message: OrderCreateMessageEvent,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
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
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): OrderCreateMessageEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseOrderCreateMessageEvent,
    } as OrderCreateMessageEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
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
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): OrderCreateMessageEvent {
    const message = {
      ...baseOrderCreateMessageEvent,
    } as OrderCreateMessageEvent;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
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
    return message;
  },

  toJSON(message: OrderCreateMessageEvent): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.market_id !== undefined && (obj.market_id = message.market_id);
    message.order_type !== undefined && (obj.order_type = message.order_type);
    message.amount !== undefined && (obj.amount = message.amount);
    message.price !== undefined && (obj.price = message.price);
    return obj;
  },

  fromPartial(
    object: DeepPartial<OrderCreateMessageEvent>
  ): OrderCreateMessageEvent {
    const message = {
      ...baseOrderCreateMessageEvent,
    } as OrderCreateMessageEvent;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
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
    return message;
  },
};

const baseOrderCancelMessageEvent: object = {
  creator: "",
  marketId: "",
  orderId: "",
  order_type: "",
};

export const OrderCancelMessageEvent = {
  encode(
    message: OrderCancelMessageEvent,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.marketId !== "") {
      writer.uint32(18).string(message.marketId);
    }
    if (message.orderId !== "") {
      writer.uint32(26).string(message.orderId);
    }
    if (message.order_type !== "") {
      writer.uint32(34).string(message.order_type);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): OrderCancelMessageEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseOrderCancelMessageEvent,
    } as OrderCancelMessageEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.marketId = reader.string();
          break;
        case 3:
          message.orderId = reader.string();
          break;
        case 4:
          message.order_type = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): OrderCancelMessageEvent {
    const message = {
      ...baseOrderCancelMessageEvent,
    } as OrderCancelMessageEvent;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.marketId !== undefined && object.marketId !== null) {
      message.marketId = String(object.marketId);
    } else {
      message.marketId = "";
    }
    if (object.orderId !== undefined && object.orderId !== null) {
      message.orderId = String(object.orderId);
    } else {
      message.orderId = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = String(object.order_type);
    } else {
      message.order_type = "";
    }
    return message;
  },

  toJSON(message: OrderCancelMessageEvent): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.marketId !== undefined && (obj.marketId = message.marketId);
    message.orderId !== undefined && (obj.orderId = message.orderId);
    message.order_type !== undefined && (obj.order_type = message.order_type);
    return obj;
  },

  fromPartial(
    object: DeepPartial<OrderCancelMessageEvent>
  ): OrderCancelMessageEvent {
    const message = {
      ...baseOrderCancelMessageEvent,
    } as OrderCancelMessageEvent;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.marketId !== undefined && object.marketId !== null) {
      message.marketId = object.marketId;
    } else {
      message.marketId = "";
    }
    if (object.orderId !== undefined && object.orderId !== null) {
      message.orderId = object.orderId;
    } else {
      message.orderId = "";
    }
    if (object.order_type !== undefined && object.order_type !== null) {
      message.order_type = object.order_type;
    } else {
      message.order_type = "";
    }
    return message;
  },
};

const baseMarketCreatedEvent: object = { creator: "", base: "", quote: "" };

export const MarketCreatedEvent = {
  encode(
    message: MarketCreatedEvent,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.base !== "") {
      writer.uint32(18).string(message.base);
    }
    if (message.quote !== "") {
      writer.uint32(26).string(message.quote);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MarketCreatedEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMarketCreatedEvent } as MarketCreatedEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.base = reader.string();
          break;
        case 3:
          message.quote = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MarketCreatedEvent {
    const message = { ...baseMarketCreatedEvent } as MarketCreatedEvent;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.base !== undefined && object.base !== null) {
      message.base = String(object.base);
    } else {
      message.base = "";
    }
    if (object.quote !== undefined && object.quote !== null) {
      message.quote = String(object.quote);
    } else {
      message.quote = "";
    }
    return message;
  },

  toJSON(message: MarketCreatedEvent): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.base !== undefined && (obj.base = message.base);
    message.quote !== undefined && (obj.quote = message.quote);
    return obj;
  },

  fromPartial(object: DeepPartial<MarketCreatedEvent>): MarketCreatedEvent {
    const message = { ...baseMarketCreatedEvent } as MarketCreatedEvent;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.base !== undefined && object.base !== null) {
      message.base = object.base;
    } else {
      message.base = "";
    }
    if (object.quote !== undefined && object.quote !== null) {
      message.quote = object.quote;
    } else {
      message.quote = "";
    }
    return message;
  },
};

const baseOrderExecutedEvent: object = {
  id: "",
  market_id: "",
  order_type: "",
  amount: "",
  price: "",
};

export const OrderExecutedEvent = {
  encode(
    message: OrderExecutedEvent,
    writer: Writer = Writer.create()
  ): Writer {
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
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): OrderExecutedEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseOrderExecutedEvent } as OrderExecutedEvent;
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
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): OrderExecutedEvent {
    const message = { ...baseOrderExecutedEvent } as OrderExecutedEvent;
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
    return message;
  },

  toJSON(message: OrderExecutedEvent): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.market_id !== undefined && (obj.market_id = message.market_id);
    message.order_type !== undefined && (obj.order_type = message.order_type);
    message.amount !== undefined && (obj.amount = message.amount);
    message.price !== undefined && (obj.price = message.price);
    return obj;
  },

  fromPartial(object: DeepPartial<OrderExecutedEvent>): OrderExecutedEvent {
    const message = { ...baseOrderExecutedEvent } as OrderExecutedEvent;
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
    return message;
  },
};

const baseOrderCanceledEvent: object = {
  id: "",
  market_id: "",
  order_type: "",
  amount: "",
  price: "",
};

export const OrderCanceledEvent = {
  encode(
    message: OrderCanceledEvent,
    writer: Writer = Writer.create()
  ): Writer {
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
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): OrderCanceledEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseOrderCanceledEvent } as OrderCanceledEvent;
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
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): OrderCanceledEvent {
    const message = { ...baseOrderCanceledEvent } as OrderCanceledEvent;
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
    return message;
  },

  toJSON(message: OrderCanceledEvent): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.market_id !== undefined && (obj.market_id = message.market_id);
    message.order_type !== undefined && (obj.order_type = message.order_type);
    message.amount !== undefined && (obj.amount = message.amount);
    message.price !== undefined && (obj.price = message.price);
    return obj;
  },

  fromPartial(object: DeepPartial<OrderCanceledEvent>): OrderCanceledEvent {
    const message = { ...baseOrderCanceledEvent } as OrderCanceledEvent;
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
    return message;
  },
};

const baseOrderSavedEvent: object = {
  id: "",
  market_id: "",
  order_type: "",
  amount: "",
  price: "",
};

export const OrderSavedEvent = {
  encode(message: OrderSavedEvent, writer: Writer = Writer.create()): Writer {
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
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): OrderSavedEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseOrderSavedEvent } as OrderSavedEvent;
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
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): OrderSavedEvent {
    const message = { ...baseOrderSavedEvent } as OrderSavedEvent;
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
    return message;
  },

  toJSON(message: OrderSavedEvent): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.market_id !== undefined && (obj.market_id = message.market_id);
    message.order_type !== undefined && (obj.order_type = message.order_type);
    message.amount !== undefined && (obj.amount = message.amount);
    message.price !== undefined && (obj.price = message.price);
    return obj;
  },

  fromPartial(object: DeepPartial<OrderSavedEvent>): OrderSavedEvent {
    const message = { ...baseOrderSavedEvent } as OrderSavedEvent;
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
