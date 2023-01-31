/* eslint-disable */
import { Reader, util, configure, Writer } from "protobufjs/minimal";
import * as Long from "long";

export const protobufPackage = "bze.cointrunk";

export interface MsgAddArticle {
  publisher: string;
  title: string;
  url: string;
  picture: string;
}

export interface MsgAddArticleResponse {}

export interface MsgPayPublisherRespect {
  creator: string;
  address: string;
  amount: string;
}

export interface MsgPayPublisherRespectResponse {
  respectPaid: number;
  publisherReward: string;
  communityPoolFunds: string;
}

const baseMsgAddArticle: object = {
  publisher: "",
  title: "",
  url: "",
  picture: "",
};

export const MsgAddArticle = {
  encode(message: MsgAddArticle, writer: Writer = Writer.create()): Writer {
    if (message.publisher !== "") {
      writer.uint32(10).string(message.publisher);
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
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgAddArticle {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgAddArticle } as MsgAddArticle;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.publisher = reader.string();
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
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgAddArticle {
    const message = { ...baseMsgAddArticle } as MsgAddArticle;
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = String(object.publisher);
    } else {
      message.publisher = "";
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
    return message;
  },

  toJSON(message: MsgAddArticle): unknown {
    const obj: any = {};
    message.publisher !== undefined && (obj.publisher = message.publisher);
    message.title !== undefined && (obj.title = message.title);
    message.url !== undefined && (obj.url = message.url);
    message.picture !== undefined && (obj.picture = message.picture);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgAddArticle>): MsgAddArticle {
    const message = { ...baseMsgAddArticle } as MsgAddArticle;
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = object.publisher;
    } else {
      message.publisher = "";
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
    return message;
  },
};

const baseMsgAddArticleResponse: object = {};

export const MsgAddArticleResponse = {
  encode(_: MsgAddArticleResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgAddArticleResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgAddArticleResponse } as MsgAddArticleResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): MsgAddArticleResponse {
    const message = { ...baseMsgAddArticleResponse } as MsgAddArticleResponse;
    return message;
  },

  toJSON(_: MsgAddArticleResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgAddArticleResponse>): MsgAddArticleResponse {
    const message = { ...baseMsgAddArticleResponse } as MsgAddArticleResponse;
    return message;
  },
};

const baseMsgPayPublisherRespect: object = {
  creator: "",
  address: "",
  amount: "",
};

export const MsgPayPublisherRespect = {
  encode(
    message: MsgPayPublisherRespect,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgPayPublisherRespect {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgPayPublisherRespect } as MsgPayPublisherRespect;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.address = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgPayPublisherRespect {
    const message = { ...baseMsgPayPublisherRespect } as MsgPayPublisherRespect;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    return message;
  },

  toJSON(message: MsgPayPublisherRespect): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.address !== undefined && (obj.address = message.address);
    message.amount !== undefined && (obj.amount = message.amount);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgPayPublisherRespect>
  ): MsgPayPublisherRespect {
    const message = { ...baseMsgPayPublisherRespect } as MsgPayPublisherRespect;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    return message;
  },
};

const baseMsgPayPublisherRespectResponse: object = {
  respectPaid: 0,
  publisherReward: "",
  communityPoolFunds: "",
};

export const MsgPayPublisherRespectResponse = {
  encode(
    message: MsgPayPublisherRespectResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.respectPaid !== 0) {
      writer.uint32(8).uint64(message.respectPaid);
    }
    if (message.publisherReward !== "") {
      writer.uint32(18).string(message.publisherReward);
    }
    if (message.communityPoolFunds !== "") {
      writer.uint32(26).string(message.communityPoolFunds);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgPayPublisherRespectResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgPayPublisherRespectResponse,
    } as MsgPayPublisherRespectResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.respectPaid = longToNumber(reader.uint64() as Long);
          break;
        case 2:
          message.publisherReward = reader.string();
          break;
        case 3:
          message.communityPoolFunds = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgPayPublisherRespectResponse {
    const message = {
      ...baseMsgPayPublisherRespectResponse,
    } as MsgPayPublisherRespectResponse;
    if (object.respectPaid !== undefined && object.respectPaid !== null) {
      message.respectPaid = Number(object.respectPaid);
    } else {
      message.respectPaid = 0;
    }
    if (
      object.publisherReward !== undefined &&
      object.publisherReward !== null
    ) {
      message.publisherReward = String(object.publisherReward);
    } else {
      message.publisherReward = "";
    }
    if (
      object.communityPoolFunds !== undefined &&
      object.communityPoolFunds !== null
    ) {
      message.communityPoolFunds = String(object.communityPoolFunds);
    } else {
      message.communityPoolFunds = "";
    }
    return message;
  },

  toJSON(message: MsgPayPublisherRespectResponse): unknown {
    const obj: any = {};
    message.respectPaid !== undefined &&
      (obj.respectPaid = message.respectPaid);
    message.publisherReward !== undefined &&
      (obj.publisherReward = message.publisherReward);
    message.communityPoolFunds !== undefined &&
      (obj.communityPoolFunds = message.communityPoolFunds);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgPayPublisherRespectResponse>
  ): MsgPayPublisherRespectResponse {
    const message = {
      ...baseMsgPayPublisherRespectResponse,
    } as MsgPayPublisherRespectResponse;
    if (object.respectPaid !== undefined && object.respectPaid !== null) {
      message.respectPaid = object.respectPaid;
    } else {
      message.respectPaid = 0;
    }
    if (
      object.publisherReward !== undefined &&
      object.publisherReward !== null
    ) {
      message.publisherReward = object.publisherReward;
    } else {
      message.publisherReward = "";
    }
    if (
      object.communityPoolFunds !== undefined &&
      object.communityPoolFunds !== null
    ) {
      message.communityPoolFunds = object.communityPoolFunds;
    } else {
      message.communityPoolFunds = "";
    }
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  AddArticle(request: MsgAddArticle): Promise<MsgAddArticleResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  PayPublisherRespect(
    request: MsgPayPublisherRespect
  ): Promise<MsgPayPublisherRespectResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  AddArticle(request: MsgAddArticle): Promise<MsgAddArticleResponse> {
    const data = MsgAddArticle.encode(request).finish();
    const promise = this.rpc.request("bze.cointrunk.Msg", "AddArticle", data);
    return promise.then((data) =>
      MsgAddArticleResponse.decode(new Reader(data))
    );
  }

  PayPublisherRespect(
    request: MsgPayPublisherRespect
  ): Promise<MsgPayPublisherRespectResponse> {
    const data = MsgPayPublisherRespect.encode(request).finish();
    const promise = this.rpc.request(
      "bze.cointrunk.Msg",
      "PayPublisherRespect",
      data
    );
    return promise.then((data) =>
      MsgPayPublisherRespectResponse.decode(new Reader(data))
    );
  }
}

interface Rpc {
  request(
    service: string,
    method: string,
    data: Uint8Array
  ): Promise<Uint8Array>;
}

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
