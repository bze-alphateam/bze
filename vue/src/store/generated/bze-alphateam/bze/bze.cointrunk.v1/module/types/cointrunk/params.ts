/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Coin } from "../cosmos/base/v1beta1/coin";

export const protobufPackage = "bze.cointrunk.v1";

/** Params defines the parameters for the module. */
export interface PublisherRespectParams {
  tax: string;
  denom: string;
}

export interface Params {
  anon_article_limit: number;
  anon_article_cost: Coin | undefined;
  publisher_respect_params: PublisherRespectParams | undefined;
}

const basePublisherRespectParams: object = { tax: "", denom: "" };

export const PublisherRespectParams = {
  encode(
    message: PublisherRespectParams,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.tax !== "") {
      writer.uint32(10).string(message.tax);
    }
    if (message.denom !== "") {
      writer.uint32(42).string(message.denom);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): PublisherRespectParams {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePublisherRespectParams } as PublisherRespectParams;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tax = reader.string();
          break;
        case 5:
          message.denom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PublisherRespectParams {
    const message = { ...basePublisherRespectParams } as PublisherRespectParams;
    if (object.tax !== undefined && object.tax !== null) {
      message.tax = String(object.tax);
    } else {
      message.tax = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    return message;
  },

  toJSON(message: PublisherRespectParams): unknown {
    const obj: any = {};
    message.tax !== undefined && (obj.tax = message.tax);
    message.denom !== undefined && (obj.denom = message.denom);
    return obj;
  },

  fromPartial(
    object: DeepPartial<PublisherRespectParams>
  ): PublisherRespectParams {
    const message = { ...basePublisherRespectParams } as PublisherRespectParams;
    if (object.tax !== undefined && object.tax !== null) {
      message.tax = object.tax;
    } else {
      message.tax = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    return message;
  },
};

const baseParams: object = { anon_article_limit: 0 };

export const Params = {
  encode(message: Params, writer: Writer = Writer.create()): Writer {
    if (message.anon_article_limit !== 0) {
      writer.uint32(8).uint64(message.anon_article_limit);
    }
    if (message.anon_article_cost !== undefined) {
      Coin.encode(message.anon_article_cost, writer.uint32(18).fork()).ldelim();
    }
    if (message.publisher_respect_params !== undefined) {
      PublisherRespectParams.encode(
        message.publisher_respect_params,
        writer.uint32(26).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): Params {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseParams } as Params;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.anon_article_limit = longToNumber(reader.uint64() as Long);
          break;
        case 2:
          message.anon_article_cost = Coin.decode(reader, reader.uint32());
          break;
        case 3:
          message.publisher_respect_params = PublisherRespectParams.decode(
            reader,
            reader.uint32()
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Params {
    const message = { ...baseParams } as Params;
    if (
      object.anon_article_limit !== undefined &&
      object.anon_article_limit !== null
    ) {
      message.anon_article_limit = Number(object.anon_article_limit);
    } else {
      message.anon_article_limit = 0;
    }
    if (
      object.anon_article_cost !== undefined &&
      object.anon_article_cost !== null
    ) {
      message.anon_article_cost = Coin.fromJSON(object.anon_article_cost);
    } else {
      message.anon_article_cost = undefined;
    }
    if (
      object.publisher_respect_params !== undefined &&
      object.publisher_respect_params !== null
    ) {
      message.publisher_respect_params = PublisherRespectParams.fromJSON(
        object.publisher_respect_params
      );
    } else {
      message.publisher_respect_params = undefined;
    }
    return message;
  },

  toJSON(message: Params): unknown {
    const obj: any = {};
    message.anon_article_limit !== undefined &&
      (obj.anon_article_limit = message.anon_article_limit);
    message.anon_article_cost !== undefined &&
      (obj.anon_article_cost = message.anon_article_cost
        ? Coin.toJSON(message.anon_article_cost)
        : undefined);
    message.publisher_respect_params !== undefined &&
      (obj.publisher_respect_params = message.publisher_respect_params
        ? PublisherRespectParams.toJSON(message.publisher_respect_params)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<Params>): Params {
    const message = { ...baseParams } as Params;
    if (
      object.anon_article_limit !== undefined &&
      object.anon_article_limit !== null
    ) {
      message.anon_article_limit = object.anon_article_limit;
    } else {
      message.anon_article_limit = 0;
    }
    if (
      object.anon_article_cost !== undefined &&
      object.anon_article_cost !== null
    ) {
      message.anon_article_cost = Coin.fromPartial(object.anon_article_cost);
    } else {
      message.anon_article_cost = undefined;
    }
    if (
      object.publisher_respect_params !== undefined &&
      object.publisher_respect_params !== null
    ) {
      message.publisher_respect_params = PublisherRespectParams.fromPartial(
        object.publisher_respect_params
      );
    } else {
      message.publisher_respect_params = undefined;
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
