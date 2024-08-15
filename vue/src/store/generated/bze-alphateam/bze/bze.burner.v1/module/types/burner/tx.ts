/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";

export const protobufPackage = "bze.burner.v1";

export interface MsgFundBurner {
  creator: string;
  amount: string;
}

export interface MsgFundBurnerResponse {}

export interface MsgStartRaffle {
  creator: string;
  pot: string;
  duration: string;
  chances: string;
  ratio: string;
  ticket_price: string;
  denom: string;
}

export interface MsgStartRaffleResponse {}

export interface MsgJoinRaffle {
  creator: string;
  denom: string;
}

export interface MsgJoinRaffleResponse {}

const baseMsgFundBurner: object = { creator: "", amount: "" };

export const MsgFundBurner = {
  encode(message: MsgFundBurner, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.amount !== "") {
      writer.uint32(18).string(message.amount);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgFundBurner {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgFundBurner } as MsgFundBurner;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgFundBurner {
    const message = { ...baseMsgFundBurner } as MsgFundBurner;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    return message;
  },

  toJSON(message: MsgFundBurner): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.amount !== undefined && (obj.amount = message.amount);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgFundBurner>): MsgFundBurner {
    const message = { ...baseMsgFundBurner } as MsgFundBurner;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    return message;
  },
};

const baseMsgFundBurnerResponse: object = {};

export const MsgFundBurnerResponse = {
  encode(_: MsgFundBurnerResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgFundBurnerResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgFundBurnerResponse } as MsgFundBurnerResponse;
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

  fromJSON(_: any): MsgFundBurnerResponse {
    const message = { ...baseMsgFundBurnerResponse } as MsgFundBurnerResponse;
    return message;
  },

  toJSON(_: MsgFundBurnerResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgFundBurnerResponse>): MsgFundBurnerResponse {
    const message = { ...baseMsgFundBurnerResponse } as MsgFundBurnerResponse;
    return message;
  },
};

const baseMsgStartRaffle: object = {
  creator: "",
  pot: "",
  duration: "",
  chances: "",
  ratio: "",
  ticket_price: "",
  denom: "",
};

export const MsgStartRaffle = {
  encode(message: MsgStartRaffle, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.pot !== "") {
      writer.uint32(18).string(message.pot);
    }
    if (message.duration !== "") {
      writer.uint32(26).string(message.duration);
    }
    if (message.chances !== "") {
      writer.uint32(34).string(message.chances);
    }
    if (message.ratio !== "") {
      writer.uint32(42).string(message.ratio);
    }
    if (message.ticket_price !== "") {
      writer.uint32(50).string(message.ticket_price);
    }
    if (message.denom !== "") {
      writer.uint32(58).string(message.denom);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgStartRaffle {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgStartRaffle } as MsgStartRaffle;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.pot = reader.string();
          break;
        case 3:
          message.duration = reader.string();
          break;
        case 4:
          message.chances = reader.string();
          break;
        case 5:
          message.ratio = reader.string();
          break;
        case 6:
          message.ticket_price = reader.string();
          break;
        case 7:
          message.denom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgStartRaffle {
    const message = { ...baseMsgStartRaffle } as MsgStartRaffle;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.pot !== undefined && object.pot !== null) {
      message.pot = String(object.pot);
    } else {
      message.pot = "";
    }
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = String(object.duration);
    } else {
      message.duration = "";
    }
    if (object.chances !== undefined && object.chances !== null) {
      message.chances = String(object.chances);
    } else {
      message.chances = "";
    }
    if (object.ratio !== undefined && object.ratio !== null) {
      message.ratio = String(object.ratio);
    } else {
      message.ratio = "";
    }
    if (object.ticket_price !== undefined && object.ticket_price !== null) {
      message.ticket_price = String(object.ticket_price);
    } else {
      message.ticket_price = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    return message;
  },

  toJSON(message: MsgStartRaffle): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.pot !== undefined && (obj.pot = message.pot);
    message.duration !== undefined && (obj.duration = message.duration);
    message.chances !== undefined && (obj.chances = message.chances);
    message.ratio !== undefined && (obj.ratio = message.ratio);
    message.ticket_price !== undefined &&
      (obj.ticket_price = message.ticket_price);
    message.denom !== undefined && (obj.denom = message.denom);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgStartRaffle>): MsgStartRaffle {
    const message = { ...baseMsgStartRaffle } as MsgStartRaffle;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.pot !== undefined && object.pot !== null) {
      message.pot = object.pot;
    } else {
      message.pot = "";
    }
    if (object.duration !== undefined && object.duration !== null) {
      message.duration = object.duration;
    } else {
      message.duration = "";
    }
    if (object.chances !== undefined && object.chances !== null) {
      message.chances = object.chances;
    } else {
      message.chances = "";
    }
    if (object.ratio !== undefined && object.ratio !== null) {
      message.ratio = object.ratio;
    } else {
      message.ratio = "";
    }
    if (object.ticket_price !== undefined && object.ticket_price !== null) {
      message.ticket_price = object.ticket_price;
    } else {
      message.ticket_price = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    return message;
  },
};

const baseMsgStartRaffleResponse: object = {};

export const MsgStartRaffleResponse = {
  encode(_: MsgStartRaffleResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgStartRaffleResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgStartRaffleResponse } as MsgStartRaffleResponse;
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

  fromJSON(_: any): MsgStartRaffleResponse {
    const message = { ...baseMsgStartRaffleResponse } as MsgStartRaffleResponse;
    return message;
  },

  toJSON(_: MsgStartRaffleResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgStartRaffleResponse>): MsgStartRaffleResponse {
    const message = { ...baseMsgStartRaffleResponse } as MsgStartRaffleResponse;
    return message;
  },
};

const baseMsgJoinRaffle: object = { creator: "", denom: "" };

export const MsgJoinRaffle = {
  encode(message: MsgJoinRaffle, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.denom !== "") {
      writer.uint32(18).string(message.denom);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgJoinRaffle {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgJoinRaffle } as MsgJoinRaffle;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.denom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgJoinRaffle {
    const message = { ...baseMsgJoinRaffle } as MsgJoinRaffle;
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
    return message;
  },

  toJSON(message: MsgJoinRaffle): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.denom !== undefined && (obj.denom = message.denom);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgJoinRaffle>): MsgJoinRaffle {
    const message = { ...baseMsgJoinRaffle } as MsgJoinRaffle;
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
    return message;
  },
};

const baseMsgJoinRaffleResponse: object = {};

export const MsgJoinRaffleResponse = {
  encode(_: MsgJoinRaffleResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgJoinRaffleResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgJoinRaffleResponse } as MsgJoinRaffleResponse;
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

  fromJSON(_: any): MsgJoinRaffleResponse {
    const message = { ...baseMsgJoinRaffleResponse } as MsgJoinRaffleResponse;
    return message;
  },

  toJSON(_: MsgJoinRaffleResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgJoinRaffleResponse>): MsgJoinRaffleResponse {
    const message = { ...baseMsgJoinRaffleResponse } as MsgJoinRaffleResponse;
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  FundBurner(request: MsgFundBurner): Promise<MsgFundBurnerResponse>;
  StartRaffle(request: MsgStartRaffle): Promise<MsgStartRaffleResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  JoinRaffle(request: MsgJoinRaffle): Promise<MsgJoinRaffleResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  FundBurner(request: MsgFundBurner): Promise<MsgFundBurnerResponse> {
    const data = MsgFundBurner.encode(request).finish();
    const promise = this.rpc.request("bze.burner.v1.Msg", "FundBurner", data);
    return promise.then((data) =>
      MsgFundBurnerResponse.decode(new Reader(data))
    );
  }

  StartRaffle(request: MsgStartRaffle): Promise<MsgStartRaffleResponse> {
    const data = MsgStartRaffle.encode(request).finish();
    const promise = this.rpc.request("bze.burner.v1.Msg", "StartRaffle", data);
    return promise.then((data) =>
      MsgStartRaffleResponse.decode(new Reader(data))
    );
  }

  JoinRaffle(request: MsgJoinRaffle): Promise<MsgJoinRaffleResponse> {
    const data = MsgJoinRaffle.encode(request).finish();
    const promise = this.rpc.request("bze.burner.v1.Msg", "JoinRaffle", data);
    return promise.then((data) =>
      MsgJoinRaffleResponse.decode(new Reader(data))
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
