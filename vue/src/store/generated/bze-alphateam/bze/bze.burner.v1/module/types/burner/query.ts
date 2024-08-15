/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Params } from "../burner/params";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import { BurnedCoins } from "../burner/burned_coins";
import { Raffle } from "../burner/raffle";

export const protobufPackage = "bze.burner.v1";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params: Params | undefined;
}

export interface QueryAllBurnedCoinsRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllBurnedCoinsResponse {
  burnedCoins: BurnedCoins[];
  pagination: PageResponse | undefined;
}

export interface QueryRafflesRequest {
  pagination: PageRequest | undefined;
}

export interface QueryRafflesResponse {
  list: Raffle[];
  pagination: PageResponse | undefined;
}

export interface QueryRaffleWinnersRequest {
  pagination: PageRequest | undefined;
}

export interface QueryRaffleWinnersResponse {
  pagination: PageResponse | undefined;
}

const baseQueryParamsRequest: object = {};

export const QueryParamsRequest = {
  encode(_: QueryParamsRequest, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryParamsRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryParamsRequest } as QueryParamsRequest;
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

  fromJSON(_: any): QueryParamsRequest {
    const message = { ...baseQueryParamsRequest } as QueryParamsRequest;
    return message;
  },

  toJSON(_: QueryParamsRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<QueryParamsRequest>): QueryParamsRequest {
    const message = { ...baseQueryParamsRequest } as QueryParamsRequest;
    return message;
  },
};

const baseQueryParamsResponse: object = {};

export const QueryParamsResponse = {
  encode(
    message: QueryParamsResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryParamsResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryParamsResponse } as QueryParamsResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryParamsResponse {
    const message = { ...baseQueryParamsResponse } as QueryParamsResponse;
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    return message;
  },

  toJSON(message: QueryParamsResponse): unknown {
    const obj: any = {};
    message.params !== undefined &&
      (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<QueryParamsResponse>): QueryParamsResponse {
    const message = { ...baseQueryParamsResponse } as QueryParamsResponse;
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    return message;
  },
};

const baseQueryAllBurnedCoinsRequest: object = {};

export const QueryAllBurnedCoinsRequest = {
  encode(
    message: QueryAllBurnedCoinsRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllBurnedCoinsRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllBurnedCoinsRequest,
    } as QueryAllBurnedCoinsRequest;
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

  fromJSON(object: any): QueryAllBurnedCoinsRequest {
    const message = {
      ...baseQueryAllBurnedCoinsRequest,
    } as QueryAllBurnedCoinsRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllBurnedCoinsRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllBurnedCoinsRequest>
  ): QueryAllBurnedCoinsRequest {
    const message = {
      ...baseQueryAllBurnedCoinsRequest,
    } as QueryAllBurnedCoinsRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllBurnedCoinsResponse: object = {};

export const QueryAllBurnedCoinsResponse = {
  encode(
    message: QueryAllBurnedCoinsResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.burnedCoins) {
      BurnedCoins.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryAllBurnedCoinsResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllBurnedCoinsResponse,
    } as QueryAllBurnedCoinsResponse;
    message.burnedCoins = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.burnedCoins.push(BurnedCoins.decode(reader, reader.uint32()));
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

  fromJSON(object: any): QueryAllBurnedCoinsResponse {
    const message = {
      ...baseQueryAllBurnedCoinsResponse,
    } as QueryAllBurnedCoinsResponse;
    message.burnedCoins = [];
    if (object.burnedCoins !== undefined && object.burnedCoins !== null) {
      for (const e of object.burnedCoins) {
        message.burnedCoins.push(BurnedCoins.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllBurnedCoinsResponse): unknown {
    const obj: any = {};
    if (message.burnedCoins) {
      obj.burnedCoins = message.burnedCoins.map((e) =>
        e ? BurnedCoins.toJSON(e) : undefined
      );
    } else {
      obj.burnedCoins = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllBurnedCoinsResponse>
  ): QueryAllBurnedCoinsResponse {
    const message = {
      ...baseQueryAllBurnedCoinsResponse,
    } as QueryAllBurnedCoinsResponse;
    message.burnedCoins = [];
    if (object.burnedCoins !== undefined && object.burnedCoins !== null) {
      for (const e of object.burnedCoins) {
        message.burnedCoins.push(BurnedCoins.fromPartial(e));
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

const baseQueryRafflesRequest: object = {};

export const QueryRafflesRequest = {
  encode(
    message: QueryRafflesRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryRafflesRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryRafflesRequest } as QueryRafflesRequest;
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

  fromJSON(object: any): QueryRafflesRequest {
    const message = { ...baseQueryRafflesRequest } as QueryRafflesRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryRafflesRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<QueryRafflesRequest>): QueryRafflesRequest {
    const message = { ...baseQueryRafflesRequest } as QueryRafflesRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryRafflesResponse: object = {};

export const QueryRafflesResponse = {
  encode(
    message: QueryRafflesResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.list) {
      Raffle.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryRafflesResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryRafflesResponse } as QueryRafflesResponse;
    message.list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.list.push(Raffle.decode(reader, reader.uint32()));
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

  fromJSON(object: any): QueryRafflesResponse {
    const message = { ...baseQueryRafflesResponse } as QueryRafflesResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(Raffle.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryRafflesResponse): unknown {
    const obj: any = {};
    if (message.list) {
      obj.list = message.list.map((e) => (e ? Raffle.toJSON(e) : undefined));
    } else {
      obj.list = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<QueryRafflesResponse>): QueryRafflesResponse {
    const message = { ...baseQueryRafflesResponse } as QueryRafflesResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(Raffle.fromPartial(e));
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

const baseQueryRaffleWinnersRequest: object = {};

export const QueryRaffleWinnersRequest = {
  encode(
    message: QueryRaffleWinnersRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryRaffleWinnersRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryRaffleWinnersRequest,
    } as QueryRaffleWinnersRequest;
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

  fromJSON(object: any): QueryRaffleWinnersRequest {
    const message = {
      ...baseQueryRaffleWinnersRequest,
    } as QueryRaffleWinnersRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryRaffleWinnersRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryRaffleWinnersRequest>
  ): QueryRaffleWinnersRequest {
    const message = {
      ...baseQueryRaffleWinnersRequest,
    } as QueryRaffleWinnersRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryRaffleWinnersResponse: object = {};

export const QueryRaffleWinnersResponse = {
  encode(
    message: QueryRaffleWinnersResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryRaffleWinnersResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryRaffleWinnersResponse,
    } as QueryRaffleWinnersResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryRaffleWinnersResponse {
    const message = {
      ...baseQueryRaffleWinnersResponse,
    } as QueryRaffleWinnersResponse;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryRaffleWinnersResponse): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryRaffleWinnersResponse>
  ): QueryRaffleWinnersResponse {
    const message = {
      ...baseQueryRaffleWinnersResponse,
    } as QueryRaffleWinnersResponse;
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
  /** Parameters queries the parameters of the module. */
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
  /** Queries a list of Raffles items. */
  Raffles(request: QueryRafflesRequest): Promise<QueryRafflesResponse>;
  /** Queries a list of RaffleWinners items. */
  RaffleWinners(
    request: QueryRaffleWinnersRequest
  ): Promise<QueryRaffleWinnersResponse>;
  AllBurnedCoins(
    request: QueryAllBurnedCoinsRequest
  ): Promise<QueryAllBurnedCoinsResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request("bze.burner.v1.Query", "Params", data);
    return promise.then((data) => QueryParamsResponse.decode(new Reader(data)));
  }

  Raffles(request: QueryRafflesRequest): Promise<QueryRafflesResponse> {
    const data = QueryRafflesRequest.encode(request).finish();
    const promise = this.rpc.request("bze.burner.v1.Query", "Raffles", data);
    return promise.then((data) =>
      QueryRafflesResponse.decode(new Reader(data))
    );
  }

  RaffleWinners(
    request: QueryRaffleWinnersRequest
  ): Promise<QueryRaffleWinnersResponse> {
    const data = QueryRaffleWinnersRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.burner.v1.Query",
      "RaffleWinners",
      data
    );
    return promise.then((data) =>
      QueryRaffleWinnersResponse.decode(new Reader(data))
    );
  }

  AllBurnedCoins(
    request: QueryAllBurnedCoinsRequest
  ): Promise<QueryAllBurnedCoinsResponse> {
    const data = QueryAllBurnedCoinsRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.burner.v1.Query",
      "AllBurnedCoins",
      data
    );
    return promise.then((data) =>
      QueryAllBurnedCoinsResponse.decode(new Reader(data))
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
