/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";

export const protobufPackage = "bze.cointrunk";

export interface MsgAddArticle {
  publisher: string;
  title: string;
  url: string;
  picture: string;
}

export interface MsgAddArticleResponse {}

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

/** Msg defines the Msg service. */
export interface Msg {
  /** this line is used by starport scaffolding # proto/tx/rpc */
  AddArticle(request: MsgAddArticle): Promise<MsgAddArticleResponse>;
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
}

interface Rpc {
  request(
    service: string,
    method: string,
    data: Uint8Array
  ): Promise<Uint8Array>;
}

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
