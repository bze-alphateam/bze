/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.tokenfactory.v1";

/** Params defines the parameters for the module. */
export interface Params {
  createDenomFee: string;
}

const baseParams: object = { createDenomFee: "" };

export const Params = {
  encode(message: Params, writer: Writer = Writer.create()): Writer {
    if (message.createDenomFee !== "") {
      writer.uint32(10).string(message.createDenomFee);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): Params {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseParams } as Params;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.createDenomFee = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Params {
    const message = { ...baseParams } as Params;
    if (object.createDenomFee !== undefined && object.createDenomFee !== null) {
      message.createDenomFee = String(object.createDenomFee);
    } else {
      message.createDenomFee = "";
    }
    return message;
  },

  toJSON(message: Params): unknown {
    const obj: any = {};
    message.createDenomFee !== undefined &&
      (obj.createDenomFee = message.createDenomFee);
    return obj;
  },

  fromPartial(object: DeepPartial<Params>): Params {
    const message = { ...baseParams } as Params;
    if (object.createDenomFee !== undefined && object.createDenomFee !== null) {
      message.createDenomFee = object.createDenomFee;
    } else {
      message.createDenomFee = "";
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
