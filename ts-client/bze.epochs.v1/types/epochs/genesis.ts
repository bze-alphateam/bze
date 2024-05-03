/* eslint-disable */
import { Timestamp } from "../google/protobuf/timestamp";
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";

export const protobufPackage = "bze.epochs.v1";

/**
 * EpochInfo is a struct that describes the data going into
 * a timer defined by the x/epochs module.
 */
export interface EpochInfo {
  /** identifier is a unique reference to this particular timer. */
  identifier: string;
  /**
   * start_time is the time at which the timer first ever ticks.
   * If start_time is in the future, the epoch will not begin until the start
   * time.
   */
  startTime: Date | undefined;
  /**
   * duration is the time in between epoch ticks.
   * In order for intended behavior to be met, duration should
   * be greater than the chains expected block time.
   * Duration must be non-zero.
   */
  duration: Duration | undefined;
  /**
   * current_epoch is the current epoch number, or in other words,
   * how many times has the timer 'ticked'.
   * The first tick (current_epoch=1) is defined as
   * the first block whose blocktime is greater than the EpochInfo start_time.
   */
  currentEpoch: number;
  /**
   * current_epoch_start_time describes the start time of the current timer
   * interval. The interval is (current_epoch_start_time,
   * current_epoch_start_time + duration] When the timer ticks, this is set to
   * current_epoch_start_time = last_epoch_start_time + duration only one timer
   * tick for a given identifier can occur per block.
   *
   * NOTE! The current_epoch_start_time may diverge significantly from the
   * wall-clock time the epoch began at. Wall-clock time of epoch start may be
   * >> current_epoch_start_time. Suppose current_epoch_start_time = 10,
   * duration = 5. Suppose the chain goes offline at t=14, and comes back online
   * at t=30, and produces blocks at every successive time. (t=31, 32, etc.)
   * * The t=30 block will start the epoch for (10, 15]
   * * The t=31 block will start the epoch for (15, 20]
   * * The t=32 block will start the epoch for (20, 25]
   * * The t=33 block will start the epoch for (25, 30]
   * * The t=34 block will start the epoch for (30, 35]
   * * The **t=36** block will start the epoch for (35, 40]
   */
  currentEpochStartTime: Date | undefined;
  /**
   * epoch_counting_started is a boolean, that indicates whether this
   * epoch timer has began yet.
   */
  epochCountingStarted: boolean;
  /**
   * current_epoch_start_height is the block height at which the current epoch
   * started. (The block height at which the timer last ticked)
   */
  currentEpochStartHeight: number;
}

/** GenesisState defines the epochs module's genesis state. */
export interface GenesisState {
  epochs: EpochInfo[];
}

const baseEpochInfo: object = {
  identifier: "",
  currentEpoch: 0,
  epochCountingStarted: false,
  currentEpochStartHeight: 0,
};

export const EpochInfo = {
  encode(message: EpochInfo, writer: Writer = Writer.create()): Writer {
    if (message.identifier !== "") {
      writer.uint32(10).string(message.identifier);
    }
    if (message.startTime !== undefined) {
      Timestamp.encode(
        toTimestamp(message.startTime),
        writer.uint32(18).fork()
      ).ldelim();
    }
    if (message.duration !== undefined) {
      Duration.encode(message.duration, writer.uint32(26).fork()).ldelim();
    }
    if (message.currentEpoch !== 0) {
      writer.uint32(32).int64(message.currentEpoch);
    }
    if (message.currentEpochStartTime !== undefined) {
      Timestamp.encode(
        toTimestamp(message.currentEpochStartTime),
        writer.uint32(42).fork()
      ).ldelim();
    }
    if (message.epochCountingStarted === true) {
      writer.uint32(48).bool(message.epochCountingStarted);
    }
    if (message.currentEpochStartHeight !== 0) {
      writer.uint32(64).int64(message.currentEpochStartHeight);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EpochInfo {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseEpochInfo } as EpochInfo;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.identifier = reader.string();
          break;
        case 2:
          message.startTime = fromTimestamp(
            Timestamp.decode(reader, reader.uint32())
          );
          break;
        case 3:
          message.duration = Duration.decode(reader, reader.uint32());
          break;
        case 4:
          message.currentEpoch = longToNumber(reader.int64() as Long);
          break;
        case 5:
          message.currentEpochStartTime = fromTimestamp(
            Timestamp.decode(reader, reader.uint32())
          );
          break;
        case 6:
          message.epochCountingStarted = reader.bool();
          break;
        case 8:
          message.currentEpochStartHeight = longToNumber(
            reader.int64() as Long
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): EpochInfo {
    const message = { ...baseEpochInfo } as EpochInfo;
    if (object.identifier !== undefined && object.identifier !== null) {
      message.identifier = String(object.identifier);
    } else {
      message.identifier = "";
    }
    if (object.startTime !== undefined && object.startTime !== null) {
      message.startTime = fromJsonTimestamp(object.startTime);
    } else {
      message.startTime = undefined;
    }
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = Duration.fromJSON(object.duration);
    } else {
      message.duration = undefined;
    }
    if (object.currentEpoch !== undefined && object.currentEpoch !== null) {
      message.currentEpoch = Number(object.currentEpoch);
    } else {
      message.currentEpoch = 0;
    }
    if (
      object.currentEpochStartTime !== undefined &&
      object.currentEpochStartTime !== null
    ) {
      message.currentEpochStartTime = fromJsonTimestamp(
        object.currentEpochStartTime
      );
    } else {
      message.currentEpochStartTime = undefined;
    }
    if (
      object.epochCountingStarted !== undefined &&
      object.epochCountingStarted !== null
    ) {
      message.epochCountingStarted = Boolean(object.epochCountingStarted);
    } else {
      message.epochCountingStarted = false;
    }
    if (
      object.currentEpochStartHeight !== undefined &&
      object.currentEpochStartHeight !== null
    ) {
      message.currentEpochStartHeight = Number(object.currentEpochStartHeight);
    } else {
      message.currentEpochStartHeight = 0;
    }
    return message;
  },

  toJSON(message: EpochInfo): unknown {
    const obj: any = {};
    message.identifier !== undefined && (obj.identifier = message.identifier);
    message.startTime !== undefined &&
      (obj.startTime =
        message.startTime !== undefined
          ? message.startTime.toISOString()
          : null);
    message.duration !== undefined &&
      (obj.duration = message.duration
        ? Duration.toJSON(message.duration)
        : undefined);
    message.currentEpoch !== undefined &&
      (obj.currentEpoch = message.currentEpoch);
    message.currentEpochStartTime !== undefined &&
      (obj.currentEpochStartTime =
        message.currentEpochStartTime !== undefined
          ? message.currentEpochStartTime.toISOString()
          : null);
    message.epochCountingStarted !== undefined &&
      (obj.epochCountingStarted = message.epochCountingStarted);
    message.currentEpochStartHeight !== undefined &&
      (obj.currentEpochStartHeight = message.currentEpochStartHeight);
    return obj;
  },

  fromPartial(object: DeepPartial<EpochInfo>): EpochInfo {
    const message = { ...baseEpochInfo } as EpochInfo;
    if (object.identifier !== undefined && object.identifier !== null) {
      message.identifier = object.identifier;
    } else {
      message.identifier = "";
    }
    if (object.startTime !== undefined && object.startTime !== null) {
      message.startTime = object.startTime;
    } else {
      message.startTime = undefined;
    }
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = Duration.fromPartial(object.duration);
    } else {
      message.duration = undefined;
    }
    if (object.currentEpoch !== undefined && object.currentEpoch !== null) {
      message.currentEpoch = object.currentEpoch;
    } else {
      message.currentEpoch = 0;
    }
    if (
      object.currentEpochStartTime !== undefined &&
      object.currentEpochStartTime !== null
    ) {
      message.currentEpochStartTime = object.currentEpochStartTime;
    } else {
      message.currentEpochStartTime = undefined;
    }
    if (
      object.epochCountingStarted !== undefined &&
      object.epochCountingStarted !== null
    ) {
      message.epochCountingStarted = object.epochCountingStarted;
    } else {
      message.epochCountingStarted = false;
    }
    if (
      object.currentEpochStartHeight !== undefined &&
      object.currentEpochStartHeight !== null
    ) {
      message.currentEpochStartHeight = object.currentEpochStartHeight;
    } else {
      message.currentEpochStartHeight = 0;
    }
    return message;
  },
};

const baseGenesisState: object = {};

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    for (const v of message.epochs) {
      EpochInfo.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.epochs = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.epochs.push(EpochInfo.decode(reader, reader.uint32()));
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
    message.epochs = [];
    if (object.epochs !== undefined && object.epochs !== null) {
      for (const e of object.epochs) {
        message.epochs.push(EpochInfo.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    if (message.epochs) {
      obj.epochs = message.epochs.map((e) =>
        e ? EpochInfo.toJSON(e) : undefined
      );
    } else {
      obj.epochs = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.epochs = [];
    if (object.epochs !== undefined && object.epochs !== null) {
      for (const e of object.epochs) {
        message.epochs.push(EpochInfo.fromPartial(e));
      }
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

function toTimestamp(date: Date): Timestamp {
  const seconds = date.getTime() / 1_000;
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = t.seconds * 1_000;
  millis += t.nanos / 1_000_000;
  return new Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof Date) {
    return o;
  } else if (typeof o === "string") {
    return new Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

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
