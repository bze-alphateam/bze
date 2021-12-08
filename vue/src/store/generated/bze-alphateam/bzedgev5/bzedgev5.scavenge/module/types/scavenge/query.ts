/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Scavenge } from "../scavenge/scavenge";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import { Commit } from "../scavenge/commit";

export const protobufPackage = "bzedgev5.scavenge";

export interface QueryGetScavengeRequest {
  index: string;
}

export interface QueryGetScavengeResponse {
  scavenge: Scavenge | undefined;
}

export interface QueryAllScavengeRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllScavengeResponse {
  scavenge: Scavenge[];
  pagination: PageResponse | undefined;
}

export interface QueryGetCommitRequest {
  index: string;
}

export interface QueryGetCommitResponse {
  commit: Commit | undefined;
}

export interface QueryAllCommitRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllCommitResponse {
  commit: Commit[];
  pagination: PageResponse | undefined;
}

const baseQueryGetScavengeRequest: object = { index: "" };

export const QueryGetScavengeRequest = {
  encode(
    message: QueryGetScavengeRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGetScavengeRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetScavengeRequest,
    } as QueryGetScavengeRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetScavengeRequest {
    const message = {
      ...baseQueryGetScavengeRequest,
    } as QueryGetScavengeRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    return message;
  },

  toJSON(message: QueryGetScavengeRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetScavengeRequest>
  ): QueryGetScavengeRequest {
    const message = {
      ...baseQueryGetScavengeRequest,
    } as QueryGetScavengeRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    return message;
  },
};

const baseQueryGetScavengeResponse: object = {};

export const QueryGetScavengeResponse = {
  encode(
    message: QueryGetScavengeResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.scavenge !== undefined) {
      Scavenge.encode(message.scavenge, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetScavengeResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetScavengeResponse,
    } as QueryGetScavengeResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.scavenge = Scavenge.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetScavengeResponse {
    const message = {
      ...baseQueryGetScavengeResponse,
    } as QueryGetScavengeResponse;
    if (object.scavenge !== undefined && object.scavenge !== null) {
      message.scavenge = Scavenge.fromJSON(object.scavenge);
    } else {
      message.scavenge = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetScavengeResponse): unknown {
    const obj: any = {};
    message.scavenge !== undefined &&
      (obj.scavenge = message.scavenge
        ? Scavenge.toJSON(message.scavenge)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetScavengeResponse>
  ): QueryGetScavengeResponse {
    const message = {
      ...baseQueryGetScavengeResponse,
    } as QueryGetScavengeResponse;
    if (object.scavenge !== undefined && object.scavenge !== null) {
      message.scavenge = Scavenge.fromPartial(object.scavenge);
    } else {
      message.scavenge = undefined;
    }
    return message;
  },
};

const baseQueryAllScavengeRequest: object = {};

export const QueryAllScavengeRequest = {
  encode(
    message: QueryAllScavengeRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryAllScavengeRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllScavengeRequest,
    } as QueryAllScavengeRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllScavengeRequest {
    const message = {
      ...baseQueryAllScavengeRequest,
    } as QueryAllScavengeRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllScavengeRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllScavengeRequest>
  ): QueryAllScavengeRequest {
    const message = {
      ...baseQueryAllScavengeRequest,
    } as QueryAllScavengeRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllScavengeResponse: object = {};

export const QueryAllScavengeResponse = {
  encode(
    message: QueryAllScavengeResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.scavenge) {
      Scavenge.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllScavengeResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllScavengeResponse,
    } as QueryAllScavengeResponse;
    message.scavenge = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.scavenge.push(Scavenge.decode(reader, reader.uint32()));
          break;
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllScavengeResponse {
    const message = {
      ...baseQueryAllScavengeResponse,
    } as QueryAllScavengeResponse;
    message.scavenge = [];
    if (object.scavenge !== undefined && object.scavenge !== null) {
      for (const e of object.scavenge) {
        message.scavenge.push(Scavenge.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllScavengeResponse): unknown {
    const obj: any = {};
    if (message.scavenge) {
      obj.scavenge = message.scavenge.map((e) =>
        e ? Scavenge.toJSON(e) : undefined
      );
    } else {
      obj.scavenge = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllScavengeResponse>
  ): QueryAllScavengeResponse {
    const message = {
      ...baseQueryAllScavengeResponse,
    } as QueryAllScavengeResponse;
    message.scavenge = [];
    if (object.scavenge !== undefined && object.scavenge !== null) {
      for (const e of object.scavenge) {
        message.scavenge.push(Scavenge.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryGetCommitRequest: object = { index: "" };

export const QueryGetCommitRequest = {
  encode(
    message: QueryGetCommitRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGetCommitRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryGetCommitRequest } as QueryGetCommitRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetCommitRequest {
    const message = { ...baseQueryGetCommitRequest } as QueryGetCommitRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    return message;
  },

  toJSON(message: QueryGetCommitRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetCommitRequest>
  ): QueryGetCommitRequest {
    const message = { ...baseQueryGetCommitRequest } as QueryGetCommitRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    return message;
  },
};

const baseQueryGetCommitResponse: object = {};

export const QueryGetCommitResponse = {
  encode(
    message: QueryGetCommitResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.commit !== undefined) {
      Commit.encode(message.commit, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGetCommitResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryGetCommitResponse } as QueryGetCommitResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.commit = Commit.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetCommitResponse {
    const message = { ...baseQueryGetCommitResponse } as QueryGetCommitResponse;
    if (object.commit !== undefined && object.commit !== null) {
      message.commit = Commit.fromJSON(object.commit);
    } else {
      message.commit = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetCommitResponse): unknown {
    const obj: any = {};
    message.commit !== undefined &&
      (obj.commit = message.commit ? Commit.toJSON(message.commit) : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetCommitResponse>
  ): QueryGetCommitResponse {
    const message = { ...baseQueryGetCommitResponse } as QueryGetCommitResponse;
    if (object.commit !== undefined && object.commit !== null) {
      message.commit = Commit.fromPartial(object.commit);
    } else {
      message.commit = undefined;
    }
    return message;
  },
};

const baseQueryAllCommitRequest: object = {};

export const QueryAllCommitRequest = {
  encode(
    message: QueryAllCommitRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryAllCommitRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryAllCommitRequest } as QueryAllCommitRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllCommitRequest {
    const message = { ...baseQueryAllCommitRequest } as QueryAllCommitRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllCommitRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllCommitRequest>
  ): QueryAllCommitRequest {
    const message = { ...baseQueryAllCommitRequest } as QueryAllCommitRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllCommitResponse: object = {};

export const QueryAllCommitResponse = {
  encode(
    message: QueryAllCommitResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.commit) {
      Commit.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryAllCommitResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryAllCommitResponse } as QueryAllCommitResponse;
    message.commit = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.commit.push(Commit.decode(reader, reader.uint32()));
          break;
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllCommitResponse {
    const message = { ...baseQueryAllCommitResponse } as QueryAllCommitResponse;
    message.commit = [];
    if (object.commit !== undefined && object.commit !== null) {
      for (const e of object.commit) {
        message.commit.push(Commit.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllCommitResponse): unknown {
    const obj: any = {};
    if (message.commit) {
      obj.commit = message.commit.map((e) =>
        e ? Commit.toJSON(e) : undefined
      );
    } else {
      obj.commit = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllCommitResponse>
  ): QueryAllCommitResponse {
    const message = { ...baseQueryAllCommitResponse } as QueryAllCommitResponse;
    message.commit = [];
    if (object.commit !== undefined && object.commit !== null) {
      for (const e of object.commit) {
        message.commit.push(Commit.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

/** Query defines the gRPC querier service. */
export interface Query {
  /** Queries a scavenge by index. */
  Scavenge(request: QueryGetScavengeRequest): Promise<QueryGetScavengeResponse>;
  /** Queries a list of scavenge items. */
  ScavengeAll(
    request: QueryAllScavengeRequest
  ): Promise<QueryAllScavengeResponse>;
  /** Queries a commit by index. */
  Commit(request: QueryGetCommitRequest): Promise<QueryGetCommitResponse>;
  /** Queries a list of commit items. */
  CommitAll(request: QueryAllCommitRequest): Promise<QueryAllCommitResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  Scavenge(
    request: QueryGetScavengeRequest
  ): Promise<QueryGetScavengeResponse> {
    const data = QueryGetScavengeRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bzedgev5.scavenge.Query",
      "Scavenge",
      data
    );
    return promise.then((data) =>
      QueryGetScavengeResponse.decode(new Reader(data))
    );
  }

  ScavengeAll(
    request: QueryAllScavengeRequest
  ): Promise<QueryAllScavengeResponse> {
    const data = QueryAllScavengeRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bzedgev5.scavenge.Query",
      "ScavengeAll",
      data
    );
    return promise.then((data) =>
      QueryAllScavengeResponse.decode(new Reader(data))
    );
  }

  Commit(request: QueryGetCommitRequest): Promise<QueryGetCommitResponse> {
    const data = QueryGetCommitRequest.encode(request).finish();
    const promise = this.rpc.request("bzedgev5.scavenge.Query", "Commit", data);
    return promise.then((data) =>
      QueryGetCommitResponse.decode(new Reader(data))
    );
  }

  CommitAll(request: QueryAllCommitRequest): Promise<QueryAllCommitResponse> {
    const data = QueryAllCommitRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bzedgev5.scavenge.Query",
      "CommitAll",
      data
    );
    return promise.then((data) =>
      QueryAllCommitResponse.decode(new Reader(data))
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
