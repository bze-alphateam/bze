/* eslint-disable */
import { Params } from "../cointrunk/params";
import { Publisher } from "../cointrunk/publisher";
import { AcceptedDomain } from "../cointrunk/accepted_domain";
import { Article } from "../cointrunk/article";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.cointrunk";

/** GenesisState defines the cointrunk module's genesis state. */
export interface GenesisState {
  params: Params | undefined;
  PublisherList: Publisher[];
  AcceptedDomainList: AcceptedDomain[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  ArticleList: Article[];
}

const baseGenesisState: object = {};

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
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.PublisherList = [];
    message.AcceptedDomainList = [];
    message.ArticleList = [];
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
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.PublisherList = [];
    message.AcceptedDomainList = [];
    message.ArticleList = [];
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
