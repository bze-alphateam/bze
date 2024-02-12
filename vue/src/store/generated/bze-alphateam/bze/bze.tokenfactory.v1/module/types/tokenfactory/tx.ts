/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Metadata } from "../cosmos/bank/v1beta1/bank";

export const protobufPackage = "bze.tokenfactory.v1";

export interface MsgCreateDenom {
  creator: string;
  subdenom: string;
}

export interface MsgCreateDenomResponse {
  new_denom: string;
}

export interface MsgMint {
  creator: string;
  coins: string;
}

export interface MsgMintResponse {}

export interface MsgBurn {
  creator: string;
  coins: string;
}

export interface MsgBurnResponse {}

export interface MsgChangeAdmin {
  creator: string;
  denom: string;
  newAdmin: string;
}

export interface MsgChangeAdminResponse {}

export interface MsgSetDenomMetadata {
  creator: string;
  metadata: Metadata | undefined;
}

export interface MsgSetDenomMetadataResponse {}

const baseMsgCreateDenom: object = { creator: "", subdenom: "" };

export const MsgCreateDenom = {
  encode(message: MsgCreateDenom, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.subdenom !== "") {
      writer.uint32(18).string(message.subdenom);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgCreateDenom {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgCreateDenom } as MsgCreateDenom;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.subdenom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgCreateDenom {
    const message = { ...baseMsgCreateDenom } as MsgCreateDenom;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.subdenom !== undefined && object.subdenom !== null) {
      message.subdenom = String(object.subdenom);
    } else {
      message.subdenom = "";
    }
    return message;
  },

  toJSON(message: MsgCreateDenom): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.subdenom !== undefined && (obj.subdenom = message.subdenom);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgCreateDenom>): MsgCreateDenom {
    const message = { ...baseMsgCreateDenom } as MsgCreateDenom;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.subdenom !== undefined && object.subdenom !== null) {
      message.subdenom = object.subdenom;
    } else {
      message.subdenom = "";
    }
    return message;
  },
};

const baseMsgCreateDenomResponse: object = { new_denom: "" };

export const MsgCreateDenomResponse = {
  encode(
    message: MsgCreateDenomResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.new_denom !== "") {
      writer.uint32(10).string(message.new_denom);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgCreateDenomResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgCreateDenomResponse } as MsgCreateDenomResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.new_denom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgCreateDenomResponse {
    const message = { ...baseMsgCreateDenomResponse } as MsgCreateDenomResponse;
    if (object.new_denom !== undefined && object.new_denom !== null) {
      message.new_denom = String(object.new_denom);
    } else {
      message.new_denom = "";
    }
    return message;
  },

  toJSON(message: MsgCreateDenomResponse): unknown {
    const obj: any = {};
    message.new_denom !== undefined && (obj.new_denom = message.new_denom);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgCreateDenomResponse>
  ): MsgCreateDenomResponse {
    const message = { ...baseMsgCreateDenomResponse } as MsgCreateDenomResponse;
    if (object.new_denom !== undefined && object.new_denom !== null) {
      message.new_denom = object.new_denom;
    } else {
      message.new_denom = "";
    }
    return message;
  },
};

const baseMsgMint: object = { creator: "", coins: "" };

export const MsgMint = {
  encode(message: MsgMint, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.coins !== "") {
      writer.uint32(18).string(message.coins);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgMint {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgMint } as MsgMint;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.coins = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgMint {
    const message = { ...baseMsgMint } as MsgMint;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.coins !== undefined && object.coins !== null) {
      message.coins = String(object.coins);
    } else {
      message.coins = "";
    }
    return message;
  },

  toJSON(message: MsgMint): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.coins !== undefined && (obj.coins = message.coins);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgMint>): MsgMint {
    const message = { ...baseMsgMint } as MsgMint;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.coins !== undefined && object.coins !== null) {
      message.coins = object.coins;
    } else {
      message.coins = "";
    }
    return message;
  },
};

const baseMsgMintResponse: object = {};

export const MsgMintResponse = {
  encode(_: MsgMintResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgMintResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgMintResponse } as MsgMintResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): MsgMintResponse {
    const message = { ...baseMsgMintResponse } as MsgMintResponse;
    return message;
  },

  toJSON(_: MsgMintResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgMintResponse>): MsgMintResponse {
    const message = { ...baseMsgMintResponse } as MsgMintResponse;
    return message;
  },
};

const baseMsgBurn: object = { creator: "", coins: "" };

export const MsgBurn = {
  encode(message: MsgBurn, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.coins !== "") {
      writer.uint32(18).string(message.coins);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgBurn {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgBurn } as MsgBurn;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.coins = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgBurn {
    const message = { ...baseMsgBurn } as MsgBurn;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.coins !== undefined && object.coins !== null) {
      message.coins = String(object.coins);
    } else {
      message.coins = "";
    }
    return message;
  },

  toJSON(message: MsgBurn): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.coins !== undefined && (obj.coins = message.coins);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgBurn>): MsgBurn {
    const message = { ...baseMsgBurn } as MsgBurn;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.coins !== undefined && object.coins !== null) {
      message.coins = object.coins;
    } else {
      message.coins = "";
    }
    return message;
  },
};

const baseMsgBurnResponse: object = {};

export const MsgBurnResponse = {
  encode(_: MsgBurnResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgBurnResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgBurnResponse } as MsgBurnResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): MsgBurnResponse {
    const message = { ...baseMsgBurnResponse } as MsgBurnResponse;
    return message;
  },

  toJSON(_: MsgBurnResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgBurnResponse>): MsgBurnResponse {
    const message = { ...baseMsgBurnResponse } as MsgBurnResponse;
    return message;
  },
};

const baseMsgChangeAdmin: object = { creator: "", denom: "", newAdmin: "" };

export const MsgChangeAdmin = {
  encode(message: MsgChangeAdmin, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.denom !== "") {
      writer.uint32(18).string(message.denom);
    }
    if (message.newAdmin !== "") {
      writer.uint32(26).string(message.newAdmin);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgChangeAdmin {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgChangeAdmin } as MsgChangeAdmin;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.denom = reader.string();
          break;
        case 3:
          message.newAdmin = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgChangeAdmin {
    const message = { ...baseMsgChangeAdmin } as MsgChangeAdmin;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    if (object.newAdmin !== undefined && object.newAdmin !== null) {
      message.newAdmin = String(object.newAdmin);
    } else {
      message.newAdmin = "";
    }
    return message;
  },

  toJSON(message: MsgChangeAdmin): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.denom !== undefined && (obj.denom = message.denom);
    message.newAdmin !== undefined && (obj.newAdmin = message.newAdmin);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgChangeAdmin>): MsgChangeAdmin {
    const message = { ...baseMsgChangeAdmin } as MsgChangeAdmin;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    if (object.newAdmin !== undefined && object.newAdmin !== null) {
      message.newAdmin = object.newAdmin;
    } else {
      message.newAdmin = "";
    }
    return message;
  },
};

const baseMsgChangeAdminResponse: object = {};

export const MsgChangeAdminResponse = {
  encode(_: MsgChangeAdminResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgChangeAdminResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgChangeAdminResponse } as MsgChangeAdminResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): MsgChangeAdminResponse {
    const message = { ...baseMsgChangeAdminResponse } as MsgChangeAdminResponse;
    return message;
  },

  toJSON(_: MsgChangeAdminResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgChangeAdminResponse>): MsgChangeAdminResponse {
    const message = { ...baseMsgChangeAdminResponse } as MsgChangeAdminResponse;
    return message;
  },
};

const baseMsgSetDenomMetadata: object = { creator: "" };

export const MsgSetDenomMetadata = {
  encode(
    message: MsgSetDenomMetadata,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.metadata !== undefined) {
      Metadata.encode(message.metadata, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgSetDenomMetadata {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgSetDenomMetadata } as MsgSetDenomMetadata;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.metadata = Metadata.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgSetDenomMetadata {
    const message = { ...baseMsgSetDenomMetadata } as MsgSetDenomMetadata;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.metadata !== undefined && object.metadata !== null) {
      message.metadata = Metadata.fromJSON(object.metadata);
    } else {
      message.metadata = undefined;
    }
    return message;
  },

  toJSON(message: MsgSetDenomMetadata): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.metadata !== undefined &&
      (obj.metadata = message.metadata
        ? Metadata.toJSON(message.metadata)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgSetDenomMetadata>): MsgSetDenomMetadata {
    const message = { ...baseMsgSetDenomMetadata } as MsgSetDenomMetadata;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.metadata !== undefined && object.metadata !== null) {
      message.metadata = Metadata.fromPartial(object.metadata);
    } else {
      message.metadata = undefined;
    }
    return message;
  },
};

const baseMsgSetDenomMetadataResponse: object = {};

export const MsgSetDenomMetadataResponse = {
  encode(
    _: MsgSetDenomMetadataResponse,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgSetDenomMetadataResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgSetDenomMetadataResponse,
    } as MsgSetDenomMetadataResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): MsgSetDenomMetadataResponse {
    const message = {
      ...baseMsgSetDenomMetadataResponse,
    } as MsgSetDenomMetadataResponse;
    return message;
  },

  toJSON(_: MsgSetDenomMetadataResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<MsgSetDenomMetadataResponse>
  ): MsgSetDenomMetadataResponse {
    const message = {
      ...baseMsgSetDenomMetadataResponse,
    } as MsgSetDenomMetadataResponse;
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  CreateDenom(request: MsgCreateDenom): Promise<MsgCreateDenomResponse>;
  Mint(request: MsgMint): Promise<MsgMintResponse>;
  Burn(request: MsgBurn): Promise<MsgBurnResponse>;
  ChangeAdmin(request: MsgChangeAdmin): Promise<MsgChangeAdminResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  SetDenomMetadata(
    request: MsgSetDenomMetadata
  ): Promise<MsgSetDenomMetadataResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  CreateDenom(request: MsgCreateDenom): Promise<MsgCreateDenomResponse> {
    const data = MsgCreateDenom.encode(request).finish();
    const promise = this.rpc.request(
      "bze.tokenfactory.v1.Msg",
      "CreateDenom",
      data
    );
    return promise.then((data) =>
      MsgCreateDenomResponse.decode(new Reader(data))
    );
  }

  Mint(request: MsgMint): Promise<MsgMintResponse> {
    const data = MsgMint.encode(request).finish();
    const promise = this.rpc.request("bze.tokenfactory.v1.Msg", "Mint", data);
    return promise.then((data) => MsgMintResponse.decode(new Reader(data)));
  }

  Burn(request: MsgBurn): Promise<MsgBurnResponse> {
    const data = MsgBurn.encode(request).finish();
    const promise = this.rpc.request("bze.tokenfactory.v1.Msg", "Burn", data);
    return promise.then((data) => MsgBurnResponse.decode(new Reader(data)));
  }

  ChangeAdmin(request: MsgChangeAdmin): Promise<MsgChangeAdminResponse> {
    const data = MsgChangeAdmin.encode(request).finish();
    const promise = this.rpc.request(
      "bze.tokenfactory.v1.Msg",
      "ChangeAdmin",
      data
    );
    return promise.then((data) =>
      MsgChangeAdminResponse.decode(new Reader(data))
    );
  }

  SetDenomMetadata(
    request: MsgSetDenomMetadata
  ): Promise<MsgSetDenomMetadataResponse> {
    const data = MsgSetDenomMetadata.encode(request).finish();
    const promise = this.rpc.request(
      "bze.tokenfactory.v1.Msg",
      "SetDenomMetadata",
      data
    );
    return promise.then((data) =>
      MsgSetDenomMetadataResponse.decode(new Reader(data))
    );
  }
}

interface Rpc {
  request(
    service: string,
    method: string,
    data: Uint8Array
  ): Promise<Uint8Array>;
}

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
