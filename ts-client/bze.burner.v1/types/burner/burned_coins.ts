/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.burner.v1";

export interface BurnedCoins {
  burned: string;
  height: string;
}

const baseBurnedCoins: object = { burned: "", height: "" };

export const BurnedCoins = {
  encode(message: BurnedCoins, writer: Writer = Writer.create()): Writer {
    if (message.burned !== "") {
      writer.uint32(10).string(message.burned);
    }
    if (message.height !== "") {
      writer.uint32(18).string(message.height);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): BurnedCoins {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseBurnedCoins } as BurnedCoins;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.burned = reader.string();
          break;
        case 2:
          message.height = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): BurnedCoins {
    const message = { ...baseBurnedCoins } as BurnedCoins;
    if (object.burned !== undefined && object.burned !== null) {
      message.burned = String(object.burned);
    } else {
      message.burned = "";
    }
    if (object.height !== undefined && object.height !== null) {
      message.height = String(object.height);
    } else {
      message.height = "";
    }
    return message;
  },

  toJSON(message: BurnedCoins): unknown {
    const obj: any = {};
    message.burned !== undefined && (obj.burned = message.burned);
    message.height !== undefined && (obj.height = message.height);
    return obj;
  },

  fromPartial(object: DeepPartial<BurnedCoins>): BurnedCoins {
    const message = { ...baseBurnedCoins } as BurnedCoins;
    if (object.burned !== undefined && object.burned !== null) {
      message.burned = object.burned;
    } else {
      message.burned = "";
    }
    if (object.height !== undefined && object.height !== null) {
      message.height = object.height;
    } else {
      message.height = "";
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
