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
  anonArticleLimit: number;
  anonArticleCost: Coin | undefined;
  publisherRespectParams: PublisherRespectParams | undefined;
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

const baseParams: object = { anonArticleLimit: 0 };

export const Params = {
  encode(message: Params, writer: Writer = Writer.create()): Writer {
    if (message.anonArticleLimit !== 0) {
      writer.uint32(8).uint64(message.anonArticleLimit);
    }
    if (message.anonArticleCost !== undefined) {
      Coin.encode(message.anonArticleCost, writer.uint32(18).fork()).ldelim();
    }
    if (message.publisherRespectParams !== undefined) {
      PublisherRespectParams.encode(
        message.publisherRespectParams,
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
          message.anonArticleLimit = longToNumber(reader.uint64() as Long);
          break;
        case 2:
          message.anonArticleCost = Coin.decode(reader, reader.uint32());
          break;
        case 3:
          message.publisherRespectParams = PublisherRespectParams.decode(
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
      object.anonArticleLimit !== undefined &&
      object.anonArticleLimit !== null
    ) {
      message.anonArticleLimit = Number(object.anonArticleLimit);
    } else {
      message.anonArticleLimit = 0;
    }
    if (
      object.anonArticleCost !== undefined &&
      object.anonArticleCost !== null
    ) {
      message.anonArticleCost = Coin.fromJSON(object.anonArticleCost);
    } else {
      message.anonArticleCost = undefined;
    }
    if (
      object.publisherRespectParams !== undefined &&
      object.publisherRespectParams !== null
    ) {
      message.publisherRespectParams = PublisherRespectParams.fromJSON(
        object.publisherRespectParams
      );
    } else {
      message.publisherRespectParams = undefined;
    }
    return message;
  },

  toJSON(message: Params): unknown {
    const obj: any = {};
    message.anonArticleLimit !== undefined &&
      (obj.anonArticleLimit = message.anonArticleLimit);
    message.anonArticleCost !== undefined &&
      (obj.anonArticleCost = message.anonArticleCost
        ? Coin.toJSON(message.anonArticleCost)
        : undefined);
    message.publisherRespectParams !== undefined &&
      (obj.publisherRespectParams = message.publisherRespectParams
        ? PublisherRespectParams.toJSON(message.publisherRespectParams)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<Params>): Params {
    const message = { ...baseParams } as Params;
    if (
      object.anonArticleLimit !== undefined &&
      object.anonArticleLimit !== null
    ) {
      message.anonArticleLimit = object.anonArticleLimit;
    } else {
      message.anonArticleLimit = 0;
    }
    if (
      object.anonArticleCost !== undefined &&
      object.anonArticleCost !== null
    ) {
      message.anonArticleCost = Coin.fromPartial(object.anonArticleCost);
    } else {
      message.anonArticleCost = undefined;
    }
    if (
      object.publisherRespectParams !== undefined &&
      object.publisherRespectParams !== null
    ) {
      message.publisherRespectParams = PublisherRespectParams.fromPartial(
        object.publisherRespectParams
      );
    } else {
      message.publisherRespectParams = undefined;
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
