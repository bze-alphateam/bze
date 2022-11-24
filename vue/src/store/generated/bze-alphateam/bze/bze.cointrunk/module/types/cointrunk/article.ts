/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.cointrunk";

export interface Article {
  title: string;
  url: string;
  picture: string;
  publisher: string;
  paid: boolean;
}

const baseArticle: object = {
  title: "",
  url: "",
  picture: "",
  publisher: "",
  paid: false,
};

export const Article = {
  encode(message: Article, writer: Writer = Writer.create()): Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.url !== "") {
      writer.uint32(18).string(message.url);
    }
    if (message.picture !== "") {
      writer.uint32(26).string(message.picture);
    }
    if (message.publisher !== "") {
      writer.uint32(34).string(message.publisher);
    }
    if (message.paid === true) {
      writer.uint32(40).bool(message.paid);
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
          message.title = reader.string();
          break;
        case 2:
          message.url = reader.string();
          break;
        case 3:
          message.picture = reader.string();
          break;
        case 4:
          message.publisher = reader.string();
          break;
        case 5:
          message.paid = reader.bool();
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
    return message;
  },

  toJSON(message: Article): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.url !== undefined && (obj.url = message.url);
    message.picture !== undefined && (obj.picture = message.picture);
    message.publisher !== undefined && (obj.publisher = message.publisher);
    message.paid !== undefined && (obj.paid = message.paid);
    return obj;
  },

  fromPartial(object: DeepPartial<Article>): Article {
    const message = { ...baseArticle } as Article;
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
