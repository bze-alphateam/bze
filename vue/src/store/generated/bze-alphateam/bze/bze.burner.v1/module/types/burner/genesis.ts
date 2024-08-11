/* eslint-disable */
import { Params } from "../burner/params";
import { BurnedCoins } from "../burner/burned_coins";
import { Raffle } from "../burner/raffle";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bze.burner.v1";

/** GenesisState defines the burner module's genesis state. */
export interface GenesisState {
  params: Params | undefined;
  burned_coins_list: BurnedCoins[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  raffle_list: Raffle[];
}

const baseGenesisState: object = {};

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.burned_coins_list) {
      BurnedCoins.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.raffle_list) {
      Raffle.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.burned_coins_list = [];
    message.raffle_list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.burned_coins_list.push(
            BurnedCoins.decode(reader, reader.uint32())
          );
          break;
        case 3:
          message.raffle_list.push(Raffle.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.burned_coins_list = [];
    message.raffle_list = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    if (
      object.burned_coins_list !== undefined &&
      object.burned_coins_list !== null
    ) {
      for (const e of object.burned_coins_list) {
        message.burned_coins_list.push(BurnedCoins.fromJSON(e));
      }
    }
    if (object.raffle_list !== undefined && object.raffle_list !== null) {
      for (const e of object.raffle_list) {
        message.raffle_list.push(Raffle.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined &&
      (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    if (message.burned_coins_list) {
      obj.burned_coins_list = message.burned_coins_list.map((e) =>
        e ? BurnedCoins.toJSON(e) : undefined
      );
    } else {
      obj.burned_coins_list = [];
    }
    if (message.raffle_list) {
      obj.raffle_list = message.raffle_list.map((e) =>
        e ? Raffle.toJSON(e) : undefined
      );
    } else {
      obj.raffle_list = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.burned_coins_list = [];
    message.raffle_list = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    if (
      object.burned_coins_list !== undefined &&
      object.burned_coins_list !== null
    ) {
      for (const e of object.burned_coins_list) {
        message.burned_coins_list.push(BurnedCoins.fromPartial(e));
      }
    }
    if (object.raffle_list !== undefined && object.raffle_list !== null) {
      for (const e of object.raffle_list) {
        message.raffle_list.push(Raffle.fromPartial(e));
      }
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
