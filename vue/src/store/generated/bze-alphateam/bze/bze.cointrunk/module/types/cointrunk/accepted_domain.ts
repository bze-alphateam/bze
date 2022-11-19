/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.cointrunk";

export interface AcceptedDomain {
  domain: string;
  active: boolean;
}

const baseAcceptedDomain: object = { domain: "", active: false };

export const AcceptedDomain = {
  encode(message: AcceptedDomain, writer: Writer = Writer.create()): Writer {
    if (message.domain !== "") {
      writer.uint32(10).string(message.domain);
    }
    if (message.active === true) {
      writer.uint32(16).bool(message.active);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): AcceptedDomain {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseAcceptedDomain } as AcceptedDomain;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.domain = reader.string();
          break;
        case 2:
          message.active = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): AcceptedDomain {
    const message = { ...baseAcceptedDomain } as AcceptedDomain;
    if (object.domain !== undefined && object.domain !== null) {
      message.domain = String(object.domain);
    } else {
      message.domain = "";
    }
    if (object.active !== undefined && object.active !== null) {
      message.active = Boolean(object.active);
    } else {
      message.active = false;
    }
    return message;
  },

  toJSON(message: AcceptedDomain): unknown {
    const obj: any = {};
    message.domain !== undefined && (obj.domain = message.domain);
    message.active !== undefined && (obj.active = message.active);
    return obj;
  },

  fromPartial(object: DeepPartial<AcceptedDomain>): AcceptedDomain {
    const message = { ...baseAcceptedDomain } as AcceptedDomain;
    if (object.domain !== undefined && object.domain !== null) {
      message.domain = object.domain;
    } else {
      message.domain = "";
    }
    if (object.active !== undefined && object.active !== null) {
      message.active = object.active;
    } else {
      message.active = false;
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
