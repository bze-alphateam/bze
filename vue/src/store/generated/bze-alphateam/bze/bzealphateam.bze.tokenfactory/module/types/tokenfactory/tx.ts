/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";

export const protobufPackage = "bzealphateam.bze.tokenfactory";

export interface MsgCreateDenom {
  creator: string;
  subdenom: string;
}

export interface MsgCreateDenomResponse {
  new_denom: string;
}

const baseMsgCreateDenom: object = { creator: "", subdenom: "" };

export const MsgCreateDenom = {
  encode(message: MsgCreateDenom, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.subdenom !== "") {
      writer.uint32(18).string(message.subdenom);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgCreateDenom {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgCreateDenom } as MsgCreateDenom;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.subdenom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgCreateDenom {
    const message = { ...baseMsgCreateDenom } as MsgCreateDenom;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.subdenom !== undefined && object.subdenom !== null) {
      message.subdenom = String(object.subdenom);
    } else {
      message.subdenom = "";
    }
    return message;
  },

  toJSON(message: MsgCreateDenom): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.subdenom !== undefined && (obj.subdenom = message.subdenom);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgCreateDenom>): MsgCreateDenom {
    const message = { ...baseMsgCreateDenom } as MsgCreateDenom;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.subdenom !== undefined && object.subdenom !== null) {
      message.subdenom = object.subdenom;
    } else {
      message.subdenom = "";
    }
    return message;
  },
};

const baseMsgCreateDenomResponse: object = { new_denom: "" };

export const MsgCreateDenomResponse = {
  encode(
    message: MsgCreateDenomResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.new_denom !== "") {
      writer.uint32(10).string(message.new_denom);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgCreateDenomResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgCreateDenomResponse } as MsgCreateDenomResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.new_denom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgCreateDenomResponse {
    const message = { ...baseMsgCreateDenomResponse } as MsgCreateDenomResponse;
    if (object.new_denom !== undefined && object.new_denom !== null) {
      message.new_denom = String(object.new_denom);
    } else {
      message.new_denom = "";
    }
    return message;
  },

  toJSON(message: MsgCreateDenomResponse): unknown {
    const obj: any = {};
    message.new_denom !== undefined && (obj.new_denom = message.new_denom);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgCreateDenomResponse>
  ): MsgCreateDenomResponse {
    const message = { ...baseMsgCreateDenomResponse } as MsgCreateDenomResponse;
    if (object.new_denom !== undefined && object.new_denom !== null) {
      message.new_denom = object.new_denom;
    } else {
      message.new_denom = "";
    }
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  /** this line is used by starport scaffolding # proto/tx/rpc */
  CreateDenom(request: MsgCreateDenom): Promise<MsgCreateDenomResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  CreateDenom(request: MsgCreateDenom): Promise<MsgCreateDenomResponse> {
    const data = MsgCreateDenom.encode(request).finish();
    const promise = this.rpc.request(
      "bzealphateam.bze.tokenfactory.Msg",
      "CreateDenom",
      data
    );
    return promise.then((data) =>
      MsgCreateDenomResponse.decode(new Reader(data))
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
