/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bzealphateam.bze.cointrunk";

export interface AddAcceptedDomainProposal {
  title: string;
  description: string;
  domain: string;
  active: boolean;
}

const baseAddAcceptedDomainProposal: object = {
  title: "",
  description: "",
  domain: "",
  active: false,
};

export const AddAcceptedDomainProposal = {
  encode(
    message: AddAcceptedDomainProposal,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(18).string(message.description);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    if (message.active === true) {
      writer.uint32(32).bool(message.active);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): AddAcceptedDomainProposal {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseAddAcceptedDomainProposal,
    } as AddAcceptedDomainProposal;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.title = reader.string();
          break;
        case 2:
          message.description = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.active = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): AddAcceptedDomainProposal {
    const message = {
      ...baseAddAcceptedDomainProposal,
    } as AddAcceptedDomainProposal;
    if (object.title !== undefined && object.title !== null) {
      message.title = String(object.title);
    } else {
      message.title = "";
    }
    if (object.description !== undefined && object.description !== null) {
      message.description = String(object.description);
    } else {
      message.description = "";
    }
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

  toJSON(message: AddAcceptedDomainProposal): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined &&
      (obj.description = message.description);
    message.domain !== undefined && (obj.domain = message.domain);
    message.active !== undefined && (obj.active = message.active);
    return obj;
  },

  fromPartial(
    object: DeepPartial<AddAcceptedDomainProposal>
  ): AddAcceptedDomainProposal {
    const message = {
      ...baseAddAcceptedDomainProposal,
    } as AddAcceptedDomainProposal;
    if (object.title !== undefined && object.title !== null) {
      message.title = object.title;
    } else {
      message.title = "";
    }
    if (object.description !== undefined && object.description !== null) {
      message.description = object.description;
    } else {
      message.description = "";
    }
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
