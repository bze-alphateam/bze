/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bzealphateam.bze.rewards";

/** Params defines the parameters for the module. */
export interface Params {
  createStakingRewardFee: string;
  createTradingRewardFee: string;
}

const baseParams: object = {
  createStakingRewardFee: "",
  createTradingRewardFee: "",
};

export const Params = {
  encode(message: Params, writer: Writer = Writer.create()): Writer {
    if (message.createStakingRewardFee !== "") {
      writer.uint32(10).string(message.createStakingRewardFee);
    }
    if (message.createTradingRewardFee !== "") {
      writer.uint32(18).string(message.createTradingRewardFee);
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
          message.createStakingRewardFee = reader.string();
          break;
        case 2:
          message.createTradingRewardFee = reader.string();
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
      object.createStakingRewardFee !== undefined &&
      object.createStakingRewardFee !== null
    ) {
      message.createStakingRewardFee = String(object.createStakingRewardFee);
    } else {
      message.createStakingRewardFee = "";
    }
    if (
      object.createTradingRewardFee !== undefined &&
      object.createTradingRewardFee !== null
    ) {
      message.createTradingRewardFee = String(object.createTradingRewardFee);
    } else {
      message.createTradingRewardFee = "";
    }
    return message;
  },

  toJSON(message: Params): unknown {
    const obj: any = {};
    message.createStakingRewardFee !== undefined &&
      (obj.createStakingRewardFee = message.createStakingRewardFee);
    message.createTradingRewardFee !== undefined &&
      (obj.createTradingRewardFee = message.createTradingRewardFee);
    return obj;
  },

  fromPartial(object: DeepPartial<Params>): Params {
    const message = { ...baseParams } as Params;
    if (
      object.createStakingRewardFee !== undefined &&
      object.createStakingRewardFee !== null
    ) {
      message.createStakingRewardFee = object.createStakingRewardFee;
    } else {
      message.createStakingRewardFee = "";
    }
    if (
      object.createTradingRewardFee !== undefined &&
      object.createTradingRewardFee !== null
    ) {
      message.createTradingRewardFee = object.createTradingRewardFee;
    } else {
      message.createTradingRewardFee = "";
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
