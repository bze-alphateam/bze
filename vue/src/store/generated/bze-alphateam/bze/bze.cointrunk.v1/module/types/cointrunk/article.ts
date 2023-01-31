/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.cointrunk.v1";

export interface Article {
  id: number;
  title: string;
  url: string;
  picture: string;
  publisher: string;
  paid: boolean;
  created_at: number;
}

const baseArticle: object = {
  id: 0,
  title: "",
  url: "",
  picture: "",
  publisher: "",
  paid: false,
  created_at: 0,
};

export const Article = {
  encode(message: Article, writer: Writer = Writer.create()): Writer {
    if (message.id !== 0) {
      writer.uint32(8).uint64(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.url !== "") {
      writer.uint32(26).string(message.url);
    }
    if (message.picture !== "") {
      writer.uint32(34).string(message.picture);
    }
    if (message.publisher !== "") {
      writer.uint32(42).string(message.publisher);
    }
    if (message.paid === true) {
      writer.uint32(48).bool(message.paid);
    }
    if (message.created_at !== 0) {
      writer.uint32(56).int64(message.created_at);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): Article {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseArticle } as Article;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = longToNumber(reader.uint64() as Long);
          break;
        case 2:
          message.title = reader.string();
          break;
        case 3:
          message.url = reader.string();
          break;
        case 4:
          message.picture = reader.string();
          break;
        case 5:
          message.publisher = reader.string();
          break;
        case 6:
          message.paid = reader.bool();
          break;
        case 7:
          message.created_at = longToNumber(reader.int64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Article {
    const message = { ...baseArticle } as Article;
    if (object.id !== undefined && object.id !== null) {
      message.id = Number(object.id);
    } else {
      message.id = 0;
    }
    if (object.title !== undefined && object.title !== null) {
      message.title = String(object.title);
    } else {
      message.title = "";
    }
    if (object.url !== undefined && object.url !== null) {
      message.url = String(object.url);
    } else {
      message.url = "";
    }
    if (object.picture !== undefined && object.picture !== null) {
      message.picture = String(object.picture);
    } else {
      message.picture = "";
    }
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = String(object.publisher);
    } else {
      message.publisher = "";
    }
    if (object.paid !== undefined && object.paid !== null) {
      message.paid = Boolean(object.paid);
    } else {
      message.paid = false;
    }
    if (object.created_at !== undefined && object.created_at !== null) {
      message.created_at = Number(object.created_at);
    } else {
      message.created_at = 0;
    }
    return message;
  },

  toJSON(message: Article): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    message.url !== undefined && (obj.url = message.url);
    message.picture !== undefined && (obj.picture = message.picture);
    message.publisher !== undefined && (obj.publisher = message.publisher);
    message.paid !== undefined && (obj.paid = message.paid);
    message.created_at !== undefined && (obj.created_at = message.created_at);
    return obj;
  },

  fromPartial(object: DeepPartial<Article>): Article {
    const message = { ...baseArticle } as Article;
    if (object.id !== undefined && object.id !== null) {
      message.id = object.id;
    } else {
      message.id = 0;
    }
    if (object.title !== undefined && object.title !== null) {
      message.title = object.title;
    } else {
      message.title = "";
    }
    if (object.url !== undefined && object.url !== null) {
      message.url = object.url;
    } else {
      message.url = "";
    }
    if (object.picture !== undefined && object.picture !== null) {
      message.picture = object.picture;
    } else {
      message.picture = "";
    }
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = object.publisher;
    } else {
      message.publisher = "";
    }
    if (object.paid !== undefined && object.paid !== null) {
      message.paid = object.paid;
    } else {
      message.paid = false;
    }
    if (object.created_at !== undefined && object.created_at !== null) {
      message.created_at = object.created_at;
    } else {
      message.created_at = 0;
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
