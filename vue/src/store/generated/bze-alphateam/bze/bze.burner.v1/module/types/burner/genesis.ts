/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Params } from "../burner/params";
import { BurnedCoins } from "../burner/burned_coins";
import { Raffle, RaffleWinner, RaffleParticipant } from "../burner/raffle";

export const protobufPackage = "bze.burner.v1";

/** GenesisState defines the burner module's genesis state. */
export interface GenesisState {
  params: Params | undefined;
  burned_coins_list: BurnedCoins[];
  raffle_list: Raffle[];
  raffle_winners_list: RaffleWinner[];
  raffle_participants_list: RaffleParticipant[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  raffle_participant_counter: number;
}

const baseGenesisState: object = { raffle_participant_counter: 0 };

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
    for (const v of message.raffle_winners_list) {
      RaffleWinner.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.raffle_participants_list) {
      RaffleParticipant.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    if (message.raffle_participant_counter !== 0) {
      writer.uint32(48).uint64(message.raffle_participant_counter);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.burned_coins_list = [];
    message.raffle_list = [];
    message.raffle_winners_list = [];
    message.raffle_participants_list = [];
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
        case 4:
          message.raffle_winners_list.push(
            RaffleWinner.decode(reader, reader.uint32())
          );
          break;
        case 5:
          message.raffle_participants_list.push(
            RaffleParticipant.decode(reader, reader.uint32())
          );
          break;
        case 6:
          message.raffle_participant_counter = longToNumber(
            reader.uint64() as Long
          );
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
    message.raffle_winners_list = [];
    message.raffle_participants_list = [];
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
    if (
      object.raffle_winners_list !== undefined &&
      object.raffle_winners_list !== null
    ) {
      for (const e of object.raffle_winners_list) {
        message.raffle_winners_list.push(RaffleWinner.fromJSON(e));
      }
    }
    if (
      object.raffle_participants_list !== undefined &&
      object.raffle_participants_list !== null
    ) {
      for (const e of object.raffle_participants_list) {
        message.raffle_participants_list.push(RaffleParticipant.fromJSON(e));
      }
    }
    if (
      object.raffle_participant_counter !== undefined &&
      object.raffle_participant_counter !== null
    ) {
      message.raffle_participant_counter = Number(
        object.raffle_participant_counter
      );
    } else {
      message.raffle_participant_counter = 0;
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
    if (message.raffle_winners_list) {
      obj.raffle_winners_list = message.raffle_winners_list.map((e) =>
        e ? RaffleWinner.toJSON(e) : undefined
      );
    } else {
      obj.raffle_winners_list = [];
    }
    if (message.raffle_participants_list) {
      obj.raffle_participants_list = message.raffle_participants_list.map((e) =>
        e ? RaffleParticipant.toJSON(e) : undefined
      );
    } else {
      obj.raffle_participants_list = [];
    }
    message.raffle_participant_counter !== undefined &&
      (obj.raffle_participant_counter = message.raffle_participant_counter);
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.burned_coins_list = [];
    message.raffle_list = [];
    message.raffle_winners_list = [];
    message.raffle_participants_list = [];
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
    if (
      object.raffle_winners_list !== undefined &&
      object.raffle_winners_list !== null
    ) {
      for (const e of object.raffle_winners_list) {
        message.raffle_winners_list.push(RaffleWinner.fromPartial(e));
      }
    }
    if (
      object.raffle_participants_list !== undefined &&
      object.raffle_participants_list !== null
    ) {
      for (const e of object.raffle_participants_list) {
        message.raffle_participants_list.push(RaffleParticipant.fromPartial(e));
      }
    }
    if (
      object.raffle_participant_counter !== undefined &&
      object.raffle_participant_counter !== null
    ) {
      message.raffle_participant_counter = object.raffle_participant_counter;
    } else {
      message.raffle_participant_counter = 0;
    }
    return message;
  },
};

declare var self: any | undefined;
declare var window: any | undefined;
var globalThis: any = (() => {
  if (typeof globalThis !== "undefined") return globalThis;
  if (typeof self !== "undefined") return self;
  if (typeof window !== "undefined") return window;
  if (typeof global !== "undefined") return global;
  throw "Unable to locate global object";
})();

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

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (util.Long !== Long) {
  util.Long = Long as any;
  configure();
}
