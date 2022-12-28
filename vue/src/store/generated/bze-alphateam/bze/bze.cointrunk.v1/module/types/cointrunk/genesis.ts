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
  publisher_list: Publisher[];
  accepted_domain_list: AcceptedDomain[];
  article_list: Article[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  articles_counter: number;
}

const baseGenesisState: object = { articles_counter: 0 };

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.publisher_list) {
      Publisher.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.accepted_domain_list) {
      AcceptedDomain.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.article_list) {
      Article.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    if (message.articles_counter !== 0) {
      writer.uint32(40).uint64(message.articles_counter);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.publisher_list = [];
    message.accepted_domain_list = [];
    message.article_list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.publisher_list.push(
            Publisher.decode(reader, reader.uint32())
          );
          break;
        case 3:
          message.accepted_domain_list.push(
            AcceptedDomain.decode(reader, reader.uint32())
          );
          break;
        case 4:
          message.article_list.push(Article.decode(reader, reader.uint32()));
          break;
        case 5:
          message.articles_counter = longToNumber(reader.uint64() as Long);
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
    message.publisher_list = [];
    message.accepted_domain_list = [];
    message.article_list = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    if (object.publisher_list !== undefined && object.publisher_list !== null) {
      for (const e of object.publisher_list) {
        message.publisher_list.push(Publisher.fromJSON(e));
      }
    }
    if (
      object.accepted_domain_list !== undefined &&
      object.accepted_domain_list !== null
    ) {
      for (const e of object.accepted_domain_list) {
        message.accepted_domain_list.push(AcceptedDomain.fromJSON(e));
      }
    }
    if (object.article_list !== undefined && object.article_list !== null) {
      for (const e of object.article_list) {
        message.article_list.push(Article.fromJSON(e));
      }
    }
    if (
      object.articles_counter !== undefined &&
      object.articles_counter !== null
    ) {
      message.articles_counter = Number(object.articles_counter);
    } else {
      message.articles_counter = 0;
    }
    return message;
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined &&
      (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    if (message.publisher_list) {
      obj.publisher_list = message.publisher_list.map((e) =>
        e ? Publisher.toJSON(e) : undefined
      );
    } else {
      obj.publisher_list = [];
    }
    if (message.accepted_domain_list) {
      obj.accepted_domain_list = message.accepted_domain_list.map((e) =>
        e ? AcceptedDomain.toJSON(e) : undefined
      );
    } else {
      obj.accepted_domain_list = [];
    }
    if (message.article_list) {
      obj.article_list = message.article_list.map((e) =>
        e ? Article.toJSON(e) : undefined
      );
    } else {
      obj.article_list = [];
    }
    message.articles_counter !== undefined &&
      (obj.articles_counter = message.articles_counter);
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.publisher_list = [];
    message.accepted_domain_list = [];
    message.article_list = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    if (object.publisher_list !== undefined && object.publisher_list !== null) {
      for (const e of object.publisher_list) {
        message.publisher_list.push(Publisher.fromPartial(e));
      }
    }
    if (
      object.accepted_domain_list !== undefined &&
      object.accepted_domain_list !== null
    ) {
      for (const e of object.accepted_domain_list) {
        message.accepted_domain_list.push(AcceptedDomain.fromPartial(e));
      }
    }
    if (object.article_list !== undefined && object.article_list !== null) {
      for (const e of object.article_list) {
        message.article_list.push(Article.fromPartial(e));
      }
    }
    if (
      object.articles_counter !== undefined &&
      object.articles_counter !== null
    ) {
      message.articles_counter = object.articles_counter;
    } else {
      message.articles_counter = 0;
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
