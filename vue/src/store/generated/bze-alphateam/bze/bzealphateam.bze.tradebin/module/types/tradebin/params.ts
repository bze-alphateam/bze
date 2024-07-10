/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bzealphateam.bze.tradebin";

/** Params defines the parameters for the module. */
export interface Params {
  createMarketFee: string;
  marketMakerFee: string;
  marketTakerFee: string;
  makerFeeDestination: string;
  takerFeeDestination: string;
}

const baseParams: object = {
  createMarketFee: "",
  marketMakerFee: "",
  marketTakerFee: "",
  makerFeeDestination: "",
  takerFeeDestination: "",
};

export const Params = {
  encode(message: Params, writer: Writer = Writer.create()): Writer {
    if (message.createMarketFee !== "") {
      writer.uint32(10).string(message.createMarketFee);
    }
    if (message.marketMakerFee !== "") {
      writer.uint32(18).string(message.marketMakerFee);
    }
    if (message.marketTakerFee !== "") {
      writer.uint32(26).string(message.marketTakerFee);
    }
    if (message.makerFeeDestination !== "") {
      writer.uint32(34).string(message.makerFeeDestination);
    }
    if (message.takerFeeDestination !== "") {
      writer.uint32(42).string(message.takerFeeDestination);
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
          message.createMarketFee = reader.string();
          break;
        case 2:
          message.marketMakerFee = reader.string();
          break;
        case 3:
          message.marketTakerFee = reader.string();
          break;
        case 4:
          message.makerFeeDestination = reader.string();
          break;
        case 5:
          message.takerFeeDestination = reader.string();
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
    if (
      object.createMarketFee !== undefined &&
      object.createMarketFee !== null
    ) {
      message.createMarketFee = String(object.createMarketFee);
    } else {
      message.createMarketFee = "";
    }
    if (object.marketMakerFee !== undefined && object.marketMakerFee !== null) {
      message.marketMakerFee = String(object.marketMakerFee);
    } else {
      message.marketMakerFee = "";
    }
    if (object.marketTakerFee !== undefined && object.marketTakerFee !== null) {
      message.marketTakerFee = String(object.marketTakerFee);
    } else {
      message.marketTakerFee = "";
    }
    if (
      object.makerFeeDestination !== undefined &&
      object.makerFeeDestination !== null
    ) {
      message.makerFeeDestination = String(object.makerFeeDestination);
    } else {
      message.makerFeeDestination = "";
    }
    if (
      object.takerFeeDestination !== undefined &&
      object.takerFeeDestination !== null
    ) {
      message.takerFeeDestination = String(object.takerFeeDestination);
    } else {
      message.takerFeeDestination = "";
    }
    return message;
  },

  toJSON(message: Params): unknown {
    const obj: any = {};
    message.createMarketFee !== undefined &&
      (obj.createMarketFee = message.createMarketFee);
    message.marketMakerFee !== undefined &&
      (obj.marketMakerFee = message.marketMakerFee);
    message.marketTakerFee !== undefined &&
      (obj.marketTakerFee = message.marketTakerFee);
    message.makerFeeDestination !== undefined &&
      (obj.makerFeeDestination = message.makerFeeDestination);
    message.takerFeeDestination !== undefined &&
      (obj.takerFeeDestination = message.takerFeeDestination);
    return obj;
  },

  fromPartial(object: DeepPartial<Params>): Params {
    const message = { ...baseParams } as Params;
    if (
      object.createMarketFee !== undefined &&
      object.createMarketFee !== null
    ) {
      message.createMarketFee = object.createMarketFee;
    } else {
      message.createMarketFee = "";
    }
    if (object.marketMakerFee !== undefined && object.marketMakerFee !== null) {
      message.marketMakerFee = object.marketMakerFee;
    } else {
      message.marketMakerFee = "";
    }
    if (object.marketTakerFee !== undefined && object.marketTakerFee !== null) {
      message.marketTakerFee = object.marketTakerFee;
    } else {
      message.marketTakerFee = "";
    }
    if (
      object.makerFeeDestination !== undefined &&
      object.makerFeeDestination !== null
    ) {
      message.makerFeeDestination = object.makerFeeDestination;
    } else {
      message.makerFeeDestination = "";
    }
    if (
      object.takerFeeDestination !== undefined &&
      object.takerFeeDestination !== null
    ) {
      message.takerFeeDestination = object.takerFeeDestination;
    } else {
      message.takerFeeDestination = "";
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
