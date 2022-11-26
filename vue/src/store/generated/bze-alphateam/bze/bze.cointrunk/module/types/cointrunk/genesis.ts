/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Params } from "../cointrunk/params";
import { Publisher } from "../cointrunk/publisher";
import { AcceptedDomain } from "../cointrunk/accepted_domain";
import { Article } from "../cointrunk/article";
import { BurnedCoins } from "../cointrunk/burned_coins";

export const protobufPackage = "bze.cointrunk";

/** GenesisState defines the cointrunk module's genesis state. */
export interface GenesisState {
  params: Params | undefined;
  PublisherList: Publisher[];
  AcceptedDomainList: AcceptedDomain[];
  ArticleList: Article[];
  BurnedCoinsList: BurnedCoins[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  articlesCounter: number;
}

const baseGenesisState: object = { articlesCounter: 0 };

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.PublisherList) {
      Publisher.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.AcceptedDomainList) {
      AcceptedDomain.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.ArticleList) {
      Article.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.BurnedCoinsList) {
      BurnedCoins.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    if (message.articlesCounter !== 0) {
      writer.uint32(48).uint64(message.articlesCounter);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.PublisherList = [];
    message.AcceptedDomainList = [];
    message.ArticleList = [];
    message.BurnedCoinsList = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.PublisherList.push(Publisher.decode(reader, reader.uint32()));
          break;
        case 3:
          message.AcceptedDomainList.push(
            AcceptedDomain.decode(reader, reader.uint32())
          );
          break;
        case 4:
          message.ArticleList.push(Article.decode(reader, reader.uint32()));
          break;
        case 5:
          message.BurnedCoinsList.push(
            BurnedCoins.decode(reader, reader.uint32())
          );
          break;
        case 6:
          message.articlesCounter = longToNumber(reader.uint64() as Long);
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
    message.PublisherList = [];
    message.AcceptedDomainList = [];
    message.ArticleList = [];
    message.BurnedCoinsList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    if (object.PublisherList !== undefined && object.PublisherList !== null) {
      for (const e of object.PublisherList) {
        message.PublisherList.push(Publisher.fromJSON(e));
      }
    }
    if (
      object.AcceptedDomainList !== undefined &&
      object.AcceptedDomainList !== null
    ) {
      for (const e of object.AcceptedDomainList) {
        message.AcceptedDomainList.push(AcceptedDomain.fromJSON(e));
      }
    }
    if (object.ArticleList !== undefined && object.ArticleList !== null) {
      for (const e of object.ArticleList) {
        message.ArticleList.push(Article.fromJSON(e));
      }
    }
    if (
      object.BurnedCoinsList !== undefined &&
      object.BurnedCoinsList !== null
    ) {
      for (const e of object.BurnedCoinsList) {
        message.BurnedCoinsList.push(BurnedCoins.fromJSON(e));
      }
    }
    if (
      object.articlesCounter !== undefined &&
      object.articlesCounter !== null
    ) {
      message.articlesCounter = Number(object.articlesCounter);
    } else {
      message.articlesCounter = 0;
    }
    return message;
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined &&
      (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    if (message.PublisherList) {
      obj.PublisherList = message.PublisherList.map((e) =>
        e ? Publisher.toJSON(e) : undefined
      );
    } else {
      obj.PublisherList = [];
    }
    if (message.AcceptedDomainList) {
      obj.AcceptedDomainList = message.AcceptedDomainList.map((e) =>
        e ? AcceptedDomain.toJSON(e) : undefined
      );
    } else {
      obj.AcceptedDomainList = [];
    }
    if (message.ArticleList) {
      obj.ArticleList = message.ArticleList.map((e) =>
        e ? Article.toJSON(e) : undefined
      );
    } else {
      obj.ArticleList = [];
    }
    if (message.BurnedCoinsList) {
      obj.BurnedCoinsList = message.BurnedCoinsList.map((e) =>
        e ? BurnedCoins.toJSON(e) : undefined
      );
    } else {
      obj.BurnedCoinsList = [];
    }
    message.articlesCounter !== undefined &&
      (obj.articlesCounter = message.articlesCounter);
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.PublisherList = [];
    message.AcceptedDomainList = [];
    message.ArticleList = [];
    message.BurnedCoinsList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    if (object.PublisherList !== undefined && object.PublisherList !== null) {
      for (const e of object.PublisherList) {
        message.PublisherList.push(Publisher.fromPartial(e));
      }
    }
    if (
      object.AcceptedDomainList !== undefined &&
      object.AcceptedDomainList !== null
    ) {
      for (const e of object.AcceptedDomainList) {
        message.AcceptedDomainList.push(AcceptedDomain.fromPartial(e));
      }
    }
    if (object.ArticleList !== undefined && object.ArticleList !== null) {
      for (const e of object.ArticleList) {
        message.ArticleList.push(Article.fromPartial(e));
      }
    }
    if (
      object.BurnedCoinsList !== undefined &&
      object.BurnedCoinsList !== null
    ) {
      for (const e of object.BurnedCoinsList) {
        message.BurnedCoinsList.push(BurnedCoins.fromPartial(e));
      }
    }
    if (
      object.articlesCounter !== undefined &&
      object.articlesCounter !== null
    ) {
      message.articlesCounter = object.articlesCounter;
    } else {
      message.articlesCounter = 0;
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
