/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.epochs.v1";

export interface EpochStartEvent {
  identifier: string;
  epoch: string;
}

export interface EpochEndEvent {
  identifier: string;
  epoch: string;
}

const baseEpochStartEvent: object = { identifier: "", epoch: "" };

export const EpochStartEvent = {
  encode(message: EpochStartEvent, writer: Writer = Writer.create()): Writer {
    if (message.identifier !== "") {
      writer.uint32(10).string(message.identifier);
    }
    if (message.epoch !== "") {
      writer.uint32(18).string(message.epoch);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EpochStartEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseEpochStartEvent } as EpochStartEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.identifier = reader.string();
          break;
        case 2:
          message.epoch = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): EpochStartEvent {
    const message = { ...baseEpochStartEvent } as EpochStartEvent;
    if (object.identifier !== undefined && object.identifier !== null) {
      message.identifier = String(object.identifier);
    } else {
      message.identifier = "";
    }
    if (object.epoch !== undefined && object.epoch !== null) {
      message.epoch = String(object.epoch);
    } else {
      message.epoch = "";
    }
    return message;
  },

  toJSON(message: EpochStartEvent): unknown {
    const obj: any = {};
    message.identifier !== undefined && (obj.identifier = message.identifier);
    message.epoch !== undefined && (obj.epoch = message.epoch);
    return obj;
  },

  fromPartial(object: DeepPartial<EpochStartEvent>): EpochStartEvent {
    const message = { ...baseEpochStartEvent } as EpochStartEvent;
    if (object.identifier !== undefined && object.identifier !== null) {
      message.identifier = object.identifier;
    } else {
      message.identifier = "";
    }
    if (object.epoch !== undefined && object.epoch !== null) {
      message.epoch = object.epoch;
    } else {
      message.epoch = "";
    }
    return message;
  },
};

const baseEpochEndEvent: object = { identifier: "", epoch: "" };

export const EpochEndEvent = {
  encode(message: EpochEndEvent, writer: Writer = Writer.create()): Writer {
    if (message.identifier !== "") {
      writer.uint32(10).string(message.identifier);
    }
    if (message.epoch !== "") {
      writer.uint32(18).string(message.epoch);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EpochEndEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseEpochEndEvent } as EpochEndEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.identifier = reader.string();
          break;
        case 2:
          message.epoch = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): EpochEndEvent {
    const message = { ...baseEpochEndEvent } as EpochEndEvent;
    if (object.identifier !== undefined && object.identifier !== null) {
      message.identifier = String(object.identifier);
    } else {
      message.identifier = "";
    }
    if (object.epoch !== undefined && object.epoch !== null) {
      message.epoch = String(object.epoch);
    } else {
      message.epoch = "";
    }
    return message;
  },

  toJSON(message: EpochEndEvent): unknown {
    const obj: any = {};
    message.identifier !== undefined && (obj.identifier = message.identifier);
    message.epoch !== undefined && (obj.epoch = message.epoch);
    return obj;
  },

  fromPartial(object: DeepPartial<EpochEndEvent>): EpochEndEvent {
    const message = { ...baseEpochEndEvent } as EpochEndEvent;
    if (object.identifier !== undefined && object.identifier !== null) {
      message.identifier = object.identifier;
    } else {
      message.identifier = "";
    }
    if (object.epoch !== undefined && object.epoch !== null) {
      message.epoch = object.epoch;
    } else {
      message.epoch = "";
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
