/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Params } from "../tradebin/params";
import { Market } from "../tradebin/market";
import { QueueMessage } from "../tradebin/queue_message";
import { Order, AggregatedOrder, HistoryOrder } from "../tradebin/order";

export const protobufPackage = "bze.tradebin.v1";

/** GenesisState defines the tradebin module's genesis state. */
export interface GenesisState {
  params: Params | undefined;
  market_list: Market[];
  queue_message_list: QueueMessage[];
  order_list: Order[];
  aggregated_order_list: AggregatedOrder[];
  history_order_list: HistoryOrder[];
  order_counter: number;
}

const baseGenesisState: object = { order_counter: 0 };

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.market_list) {
      Market.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.queue_message_list) {
      QueueMessage.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.order_list) {
      Order.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.aggregated_order_list) {
      AggregatedOrder.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.history_order_list) {
      HistoryOrder.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    if (message.order_counter !== 0) {
      writer.uint32(56).int64(message.order_counter);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.market_list = [];
    message.queue_message_list = [];
    message.order_list = [];
    message.aggregated_order_list = [];
    message.history_order_list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.market_list.push(Market.decode(reader, reader.uint32()));
          break;
        case 3:
          message.queue_message_list.push(
            QueueMessage.decode(reader, reader.uint32())
          );
          break;
        case 4:
          message.order_list.push(Order.decode(reader, reader.uint32()));
          break;
        case 5:
          message.aggregated_order_list.push(
            AggregatedOrder.decode(reader, reader.uint32())
          );
          break;
        case 6:
          message.history_order_list.push(
            HistoryOrder.decode(reader, reader.uint32())
          );
          break;
        case 7:
          message.order_counter = longToNumber(reader.int64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.market_list = [];
    message.queue_message_list = [];
    message.order_list = [];
    message.aggregated_order_list = [];
    message.history_order_list = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    if (object.market_list !== undefined && object.market_list !== null) {
      for (const e of object.market_list) {
        message.market_list.push(Market.fromJSON(e));
      }
    }
    if (
      object.queue_message_list !== undefined &&
      object.queue_message_list !== null
    ) {
      for (const e of object.queue_message_list) {
        message.queue_message_list.push(QueueMessage.fromJSON(e));
      }
    }
    if (object.order_list !== undefined && object.order_list !== null) {
      for (const e of object.order_list) {
        message.order_list.push(Order.fromJSON(e));
      }
    }
    if (
      object.aggregated_order_list !== undefined &&
      object.aggregated_order_list !== null
    ) {
      for (const e of object.aggregated_order_list) {
        message.aggregated_order_list.push(AggregatedOrder.fromJSON(e));
      }
    }
    if (
      object.history_order_list !== undefined &&
      object.history_order_list !== null
    ) {
      for (const e of object.history_order_list) {
        message.history_order_list.push(HistoryOrder.fromJSON(e));
      }
    }
    if (object.order_counter !== undefined && object.order_counter !== null) {
      message.order_counter = Number(object.order_counter);
    } else {
      message.order_counter = 0;
    }
    return message;
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined &&
      (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    if (message.market_list) {
      obj.market_list = message.market_list.map((e) =>
        e ? Market.toJSON(e) : undefined
      );
    } else {
      obj.market_list = [];
    }
    if (message.queue_message_list) {
      obj.queue_message_list = message.queue_message_list.map((e) =>
        e ? QueueMessage.toJSON(e) : undefined
      );
    } else {
      obj.queue_message_list = [];
    }
    if (message.order_list) {
      obj.order_list = message.order_list.map((e) =>
        e ? Order.toJSON(e) : undefined
      );
    } else {
      obj.order_list = [];
    }
    if (message.aggregated_order_list) {
      obj.aggregated_order_list = message.aggregated_order_list.map((e) =>
        e ? AggregatedOrder.toJSON(e) : undefined
      );
    } else {
      obj.aggregated_order_list = [];
    }
    if (message.history_order_list) {
      obj.history_order_list = message.history_order_list.map((e) =>
        e ? HistoryOrder.toJSON(e) : undefined
      );
    } else {
      obj.history_order_list = [];
    }
    message.order_counter !== undefined &&
      (obj.order_counter = message.order_counter);
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.market_list = [];
    message.queue_message_list = [];
    message.order_list = [];
    message.aggregated_order_list = [];
    message.history_order_list = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    if (object.market_list !== undefined && object.market_list !== null) {
      for (const e of object.market_list) {
        message.market_list.push(Market.fromPartial(e));
      }
    }
    if (
      object.queue_message_list !== undefined &&
      object.queue_message_list !== null
    ) {
      for (const e of object.queue_message_list) {
        message.queue_message_list.push(QueueMessage.fromPartial(e));
      }
    }
    if (object.order_list !== undefined && object.order_list !== null) {
      for (const e of object.order_list) {
        message.order_list.push(Order.fromPartial(e));
      }
    }
    if (
      object.aggregated_order_list !== undefined &&
      object.aggregated_order_list !== null
    ) {
      for (const e of object.aggregated_order_list) {
        message.aggregated_order_list.push(AggregatedOrder.fromPartial(e));
      }
    }
    if (
      object.history_order_list !== undefined &&
      object.history_order_list !== null
    ) {
      for (const e of object.history_order_list) {
        message.history_order_list.push(HistoryOrder.fromPartial(e));
      }
    }
    if (object.order_counter !== undefined && object.order_counter !== null) {
      message.order_counter = object.order_counter;
    } else {
      message.order_counter = 0;
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
