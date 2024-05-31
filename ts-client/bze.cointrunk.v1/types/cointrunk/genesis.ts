/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Params } from "../cointrunk/params";
import { Publisher } from "../cointrunk/publisher";
import { AcceptedDomain } from "../cointrunk/accepted_domain";
import { Article } from "../cointrunk/article";

export const protobufPackage = "bze.cointrunk.v1";

/** GenesisState defines the cointrunk module's genesis state. */
export interface GenesisState {
  params: Params | undefined;
  publisherList: Publisher[];
  acceptedDomainList: AcceptedDomain[];
  articleList: Article[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  articlesCounter: number;
}

const baseGenesisState: object = { articlesCounter: 0 };

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.publisherList) {
      Publisher.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.acceptedDomainList) {
      AcceptedDomain.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.articleList) {
      Article.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    if (message.articlesCounter !== 0) {
      writer.uint32(40).uint64(message.articlesCounter);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.publisherList = [];
    message.acceptedDomainList = [];
    message.articleList = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.publisherList.push(Publisher.decode(reader, reader.uint32()));
          break;
        case 3:
          message.acceptedDomainList.push(
            AcceptedDomain.decode(reader, reader.uint32())
          );
          break;
        case 4:
          message.articleList.push(Article.decode(reader, reader.uint32()));
          break;
        case 5:
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
    message.publisherList = [];
    message.acceptedDomainList = [];
    message.articleList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    if (object.publisherList !== undefined && object.publisherList !== null) {
      for (const e of object.publisherList) {
        message.publisherList.push(Publisher.fromJSON(e));
      }
    }
    if (
      object.acceptedDomainList !== undefined &&
      object.acceptedDomainList !== null
    ) {
      for (const e of object.acceptedDomainList) {
        message.acceptedDomainList.push(AcceptedDomain.fromJSON(e));
      }
    }
    if (object.articleList !== undefined && object.articleList !== null) {
      for (const e of object.articleList) {
        message.articleList.push(Article.fromJSON(e));
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
    if (message.publisherList) {
      obj.publisherList = message.publisherList.map((e) =>
        e ? Publisher.toJSON(e) : undefined
      );
    } else {
      obj.publisherList = [];
    }
    if (message.acceptedDomainList) {
      obj.acceptedDomainList = message.acceptedDomainList.map((e) =>
        e ? AcceptedDomain.toJSON(e) : undefined
      );
    } else {
      obj.acceptedDomainList = [];
    }
    if (message.articleList) {
      obj.articleList = message.articleList.map((e) =>
        e ? Article.toJSON(e) : undefined
      );
    } else {
      obj.articleList = [];
    }
    message.articlesCounter !== undefined &&
      (obj.articlesCounter = message.articlesCounter);
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.publisherList = [];
    message.acceptedDomainList = [];
    message.articleList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    if (object.publisherList !== undefined && object.publisherList !== null) {
      for (const e of object.publisherList) {
        message.publisherList.push(Publisher.fromPartial(e));
      }
    }
    if (
      object.acceptedDomainList !== undefined &&
      object.acceptedDomainList !== null
    ) {
      for (const e of object.acceptedDomainList) {
        message.acceptedDomainList.push(AcceptedDomain.fromPartial(e));
      }
    }
    if (object.articleList !== undefined && object.articleList !== null) {
      for (const e of object.articleList) {
        message.articleList.push(Article.fromPartial(e));
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
