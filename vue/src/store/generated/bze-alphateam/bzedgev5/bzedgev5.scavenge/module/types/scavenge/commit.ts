/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "bzedgev5.scavenge";

export interface Commit {
  index: string;
  solutionHash: string;
  solutionScavengerHash: string;
}

const baseCommit: object = {
  index: "",
  solutionHash: "",
  solutionScavengerHash: "",
};

export const Commit = {
  encode(message: Commit, writer: Writer = Writer.create()): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.solutionHash !== "") {
      writer.uint32(18).string(message.solutionHash);
    }
    if (message.solutionScavengerHash !== "") {
      writer.uint32(26).string(message.solutionScavengerHash);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): Commit {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseCommit } as Commit;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        case 2:
          message.solutionHash = reader.string();
          break;
        case 3:
          message.solutionScavengerHash = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Commit {
    const message = { ...baseCommit } as Commit;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    if (object.solutionHash !== undefined && object.solutionHash !== null) {
      message.solutionHash = String(object.solutionHash);
    } else {
      message.solutionHash = "";
    }
    if (
      object.solutionScavengerHash !== undefined &&
      object.solutionScavengerHash !== null
    ) {
      message.solutionScavengerHash = String(object.solutionScavengerHash);
    } else {
      message.solutionScavengerHash = "";
    }
    return message;
  },

  toJSON(message: Commit): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.solutionHash !== undefined &&
      (obj.solutionHash = message.solutionHash);
    message.solutionScavengerHash !== undefined &&
      (obj.solutionScavengerHash = message.solutionScavengerHash);
    return obj;
  },

  fromPartial(object: DeepPartial<Commit>): Commit {
    const message = { ...baseCommit } as Commit;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    if (object.solutionHash !== undefined && object.solutionHash !== null) {
      message.solutionHash = object.solutionHash;
    } else {
      message.solutionHash = "";
    }
    if (
      object.solutionScavengerHash !== undefined &&
      object.solutionScavengerHash !== null
    ) {
      message.solutionScavengerHash = object.solutionScavengerHash;
    } else {
      message.solutionScavengerHash = "";
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
