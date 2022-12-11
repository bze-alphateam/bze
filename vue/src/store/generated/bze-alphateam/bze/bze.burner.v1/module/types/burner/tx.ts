/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";

export const protobufPackage = "bze.burner.v1";

export interface MsgFundBurner {
  creator: string;
  amount: string;
}

export interface MsgFundBurnerResponse {}

const baseMsgFundBurner: object = { creator: "", amount: "" };

export const MsgFundBurner = {
  encode(message: MsgFundBurner, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.amount !== "") {
      writer.uint32(18).string(message.amount);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgFundBurner {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgFundBurner } as MsgFundBurner;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgFundBurner {
    const message = { ...baseMsgFundBurner } as MsgFundBurner;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    return message;
  },

  toJSON(message: MsgFundBurner): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.amount !== undefined && (obj.amount = message.amount);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgFundBurner>): MsgFundBurner {
    const message = { ...baseMsgFundBurner } as MsgFundBurner;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    return message;
  },
};

const baseMsgFundBurnerResponse: object = {};

export const MsgFundBurnerResponse = {
  encode(_: MsgFundBurnerResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgFundBurnerResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgFundBurnerResponse } as MsgFundBurnerResponse;
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

  fromJSON(_: any): MsgFundBurnerResponse {
    const message = { ...baseMsgFundBurnerResponse } as MsgFundBurnerResponse;
    return message;
  },

  toJSON(_: MsgFundBurnerResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgFundBurnerResponse>): MsgFundBurnerResponse {
    const message = { ...baseMsgFundBurnerResponse } as MsgFundBurnerResponse;
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  /** this line is used by starport scaffolding # proto/tx/rpc */
  FundBurner(request: MsgFundBurner): Promise<MsgFundBurnerResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  FundBurner(request: MsgFundBurner): Promise<MsgFundBurnerResponse> {
    const data = MsgFundBurner.encode(request).finish();
    const promise = this.rpc.request("bze.burner.v1.Msg", "FundBurner", data);
    return promise.then((data) =>
      MsgFundBurnerResponse.decode(new Reader(data))
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
