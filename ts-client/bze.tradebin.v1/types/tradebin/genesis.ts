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
  marketList: Market[];
  queueMessageList: QueueMessage[];
  orderList: Order[];
  aggregatedOrderList: AggregatedOrder[];
  historyOrderList: HistoryOrder[];
  orderCounter: number;
}

const baseGenesisState: object = { orderCounter: 0 };

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.marketList) {
      Market.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.queueMessageList) {
      QueueMessage.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.orderList) {
      Order.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.aggregatedOrderList) {
      AggregatedOrder.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.historyOrderList) {
      HistoryOrder.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    if (message.orderCounter !== 0) {
      writer.uint32(56).int64(message.orderCounter);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.marketList = [];
    message.queueMessageList = [];
    message.orderList = [];
    message.aggregatedOrderList = [];
    message.historyOrderList = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.marketList.push(Market.decode(reader, reader.uint32()));
          break;
        case 3:
          message.queueMessageList.push(
            QueueMessage.decode(reader, reader.uint32())
          );
          break;
        case 4:
          message.orderList.push(Order.decode(reader, reader.uint32()));
          break;
        case 5:
          message.aggregatedOrderList.push(
            AggregatedOrder.decode(reader, reader.uint32())
          );
          break;
        case 6:
          message.historyOrderList.push(
            HistoryOrder.decode(reader, reader.uint32())
          );
          break;
        case 7:
          message.orderCounter = longToNumber(reader.int64() as Long);
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
    message.marketList = [];
    message.queueMessageList = [];
    message.orderList = [];
    message.aggregatedOrderList = [];
    message.historyOrderList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    if (object.marketList !== undefined && object.marketList !== null) {
      for (const e of object.marketList) {
        message.marketList.push(Market.fromJSON(e));
      }
    }
    if (
      object.queueMessageList !== undefined &&
      object.queueMessageList !== null
    ) {
      for (const e of object.queueMessageList) {
        message.queueMessageList.push(QueueMessage.fromJSON(e));
      }
    }
    if (object.orderList !== undefined && object.orderList !== null) {
      for (const e of object.orderList) {
        message.orderList.push(Order.fromJSON(e));
      }
    }
    if (
      object.aggregatedOrderList !== undefined &&
      object.aggregatedOrderList !== null
    ) {
      for (const e of object.aggregatedOrderList) {
        message.aggregatedOrderList.push(AggregatedOrder.fromJSON(e));
      }
    }
    if (
      object.historyOrderList !== undefined &&
      object.historyOrderList !== null
    ) {
      for (const e of object.historyOrderList) {
        message.historyOrderList.push(HistoryOrder.fromJSON(e));
      }
    }
    if (object.orderCounter !== undefined && object.orderCounter !== null) {
      message.orderCounter = Number(object.orderCounter);
    } else {
      message.orderCounter = 0;
    }
    return message;
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined &&
      (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    if (message.marketList) {
      obj.marketList = message.marketList.map((e) =>
        e ? Market.toJSON(e) : undefined
      );
    } else {
      obj.marketList = [];
    }
    if (message.queueMessageList) {
      obj.queueMessageList = message.queueMessageList.map((e) =>
        e ? QueueMessage.toJSON(e) : undefined
      );
    } else {
      obj.queueMessageList = [];
    }
    if (message.orderList) {
      obj.orderList = message.orderList.map((e) =>
        e ? Order.toJSON(e) : undefined
      );
    } else {
      obj.orderList = [];
    }
    if (message.aggregatedOrderList) {
      obj.aggregatedOrderList = message.aggregatedOrderList.map((e) =>
        e ? AggregatedOrder.toJSON(e) : undefined
      );
    } else {
      obj.aggregatedOrderList = [];
    }
    if (message.historyOrderList) {
      obj.historyOrderList = message.historyOrderList.map((e) =>
        e ? HistoryOrder.toJSON(e) : undefined
      );
    } else {
      obj.historyOrderList = [];
    }
    message.orderCounter !== undefined &&
      (obj.orderCounter = message.orderCounter);
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.marketList = [];
    message.queueMessageList = [];
    message.orderList = [];
    message.aggregatedOrderList = [];
    message.historyOrderList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    if (object.marketList !== undefined && object.marketList !== null) {
      for (const e of object.marketList) {
        message.marketList.push(Market.fromPartial(e));
      }
    }
    if (
      object.queueMessageList !== undefined &&
      object.queueMessageList !== null
    ) {
      for (const e of object.queueMessageList) {
        message.queueMessageList.push(QueueMessage.fromPartial(e));
      }
    }
    if (object.orderList !== undefined && object.orderList !== null) {
      for (const e of object.orderList) {
        message.orderList.push(Order.fromPartial(e));
      }
    }
    if (
      object.aggregatedOrderList !== undefined &&
      object.aggregatedOrderList !== null
    ) {
      for (const e of object.aggregatedOrderList) {
        message.aggregatedOrderList.push(AggregatedOrder.fromPartial(e));
      }
    }
    if (
      object.historyOrderList !== undefined &&
      object.historyOrderList !== null
    ) {
      for (const e of object.historyOrderList) {
        message.historyOrderList.push(HistoryOrder.fromPartial(e));
      }
    }
    if (object.orderCounter !== undefined && object.orderCounter !== null) {
      message.orderCounter = object.orderCounter;
    } else {
      message.orderCounter = 0;
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
