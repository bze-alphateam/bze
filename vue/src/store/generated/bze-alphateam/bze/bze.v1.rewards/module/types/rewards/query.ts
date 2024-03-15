/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Params } from "../rewards/params";
import { StakingReward } from "../rewards/staking_reward";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import { TradingReward } from "../rewards/trading_reward";

export const protobufPackage = "bze.v1.rewards";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params: Params | undefined;
}

export interface QueryGetStakingRewardRequest {
  reward_id: string;
}

export interface QueryGetStakingRewardResponse {
  staking_reward: StakingReward | undefined;
}

export interface QueryAllStakingRewardRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllStakingRewardResponse {
  list: StakingReward[];
  pagination: PageResponse | undefined;
}

export interface QueryGetTradingRewardRequest {
  rewardId: string;
}

export interface QueryGetTradingRewardResponse {
  tradingReward: TradingReward | undefined;
}

export interface QueryAllTradingRewardRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllTradingRewardResponse {
  tradingReward: TradingReward[];
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

const baseQueryGetStakingRewardRequest: object = { reward_id: "" };

export const QueryGetStakingRewardRequest = {
  encode(
    message: QueryGetStakingRewardRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.reward_id !== "") {
      writer.uint32(10).string(message.reward_id);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetStakingRewardRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetStakingRewardRequest,
    } as QueryGetStakingRewardRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reward_id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetStakingRewardRequest {
    const message = {
      ...baseQueryGetStakingRewardRequest,
    } as QueryGetStakingRewardRequest;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = String(object.reward_id);
    } else {
      message.reward_id = "";
    }
    return message;
  },

  toJSON(message: QueryGetStakingRewardRequest): unknown {
    const obj: any = {};
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetStakingRewardRequest>
  ): QueryGetStakingRewardRequest {
    const message = {
      ...baseQueryGetStakingRewardRequest,
    } as QueryGetStakingRewardRequest;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = object.reward_id;
    } else {
      message.reward_id = "";
    }
    return message;
  },
};

const baseQueryGetStakingRewardResponse: object = {};

export const QueryGetStakingRewardResponse = {
  encode(
    message: QueryGetStakingRewardResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.staking_reward !== undefined) {
      StakingReward.encode(
        message.staking_reward,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetStakingRewardResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetStakingRewardResponse,
    } as QueryGetStakingRewardResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.staking_reward = StakingReward.decode(
            reader,
            reader.uint32()
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetStakingRewardResponse {
    const message = {
      ...baseQueryGetStakingRewardResponse,
    } as QueryGetStakingRewardResponse;
    if (object.staking_reward !== undefined && object.staking_reward !== null) {
      message.staking_reward = StakingReward.fromJSON(object.staking_reward);
    } else {
      message.staking_reward = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetStakingRewardResponse): unknown {
    const obj: any = {};
    message.staking_reward !== undefined &&
      (obj.staking_reward = message.staking_reward
        ? StakingReward.toJSON(message.staking_reward)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetStakingRewardResponse>
  ): QueryGetStakingRewardResponse {
    const message = {
      ...baseQueryGetStakingRewardResponse,
    } as QueryGetStakingRewardResponse;
    if (object.staking_reward !== undefined && object.staking_reward !== null) {
      message.staking_reward = StakingReward.fromPartial(object.staking_reward);
    } else {
      message.staking_reward = undefined;
    }
    return message;
  },
};

const baseQueryAllStakingRewardRequest: object = {};

export const QueryAllStakingRewardRequest = {
  encode(
    message: QueryAllStakingRewardRequest,
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
  ): QueryAllStakingRewardRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllStakingRewardRequest,
    } as QueryAllStakingRewardRequest;
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

  fromJSON(object: any): QueryAllStakingRewardRequest {
    const message = {
      ...baseQueryAllStakingRewardRequest,
    } as QueryAllStakingRewardRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllStakingRewardRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllStakingRewardRequest>
  ): QueryAllStakingRewardRequest {
    const message = {
      ...baseQueryAllStakingRewardRequest,
    } as QueryAllStakingRewardRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllStakingRewardResponse: object = {};

export const QueryAllStakingRewardResponse = {
  encode(
    message: QueryAllStakingRewardResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.list) {
      StakingReward.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryAllStakingRewardResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllStakingRewardResponse,
    } as QueryAllStakingRewardResponse;
    message.list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.list.push(StakingReward.decode(reader, reader.uint32()));
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

  fromJSON(object: any): QueryAllStakingRewardResponse {
    const message = {
      ...baseQueryAllStakingRewardResponse,
    } as QueryAllStakingRewardResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(StakingReward.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllStakingRewardResponse): unknown {
    const obj: any = {};
    if (message.list) {
      obj.list = message.list.map((e) =>
        e ? StakingReward.toJSON(e) : undefined
      );
    } else {
      obj.list = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllStakingRewardResponse>
  ): QueryAllStakingRewardResponse {
    const message = {
      ...baseQueryAllStakingRewardResponse,
    } as QueryAllStakingRewardResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(StakingReward.fromPartial(e));
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

const baseQueryGetTradingRewardRequest: object = { rewardId: "" };

export const QueryGetTradingRewardRequest = {
  encode(
    message: QueryGetTradingRewardRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.rewardId !== "") {
      writer.uint32(10).string(message.rewardId);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetTradingRewardRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetTradingRewardRequest,
    } as QueryGetTradingRewardRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.rewardId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetTradingRewardRequest {
    const message = {
      ...baseQueryGetTradingRewardRequest,
    } as QueryGetTradingRewardRequest;
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = String(object.rewardId);
    } else {
      message.rewardId = "";
    }
    return message;
  },

  toJSON(message: QueryGetTradingRewardRequest): unknown {
    const obj: any = {};
    message.rewardId !== undefined && (obj.rewardId = message.rewardId);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetTradingRewardRequest>
  ): QueryGetTradingRewardRequest {
    const message = {
      ...baseQueryGetTradingRewardRequest,
    } as QueryGetTradingRewardRequest;
    if (object.rewardId !== undefined && object.rewardId !== null) {
      message.rewardId = object.rewardId;
    } else {
      message.rewardId = "";
    }
    return message;
  },
};

const baseQueryGetTradingRewardResponse: object = {};

export const QueryGetTradingRewardResponse = {
  encode(
    message: QueryGetTradingRewardResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.tradingReward !== undefined) {
      TradingReward.encode(
        message.tradingReward,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetTradingRewardResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetTradingRewardResponse,
    } as QueryGetTradingRewardResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tradingReward = TradingReward.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetTradingRewardResponse {
    const message = {
      ...baseQueryGetTradingRewardResponse,
    } as QueryGetTradingRewardResponse;
    if (object.tradingReward !== undefined && object.tradingReward !== null) {
      message.tradingReward = TradingReward.fromJSON(object.tradingReward);
    } else {
      message.tradingReward = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetTradingRewardResponse): unknown {
    const obj: any = {};
    message.tradingReward !== undefined &&
      (obj.tradingReward = message.tradingReward
        ? TradingReward.toJSON(message.tradingReward)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetTradingRewardResponse>
  ): QueryGetTradingRewardResponse {
    const message = {
      ...baseQueryGetTradingRewardResponse,
    } as QueryGetTradingRewardResponse;
    if (object.tradingReward !== undefined && object.tradingReward !== null) {
      message.tradingReward = TradingReward.fromPartial(object.tradingReward);
    } else {
      message.tradingReward = undefined;
    }
    return message;
  },
};

const baseQueryAllTradingRewardRequest: object = {};

export const QueryAllTradingRewardRequest = {
  encode(
    message: QueryAllTradingRewardRequest,
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
  ): QueryAllTradingRewardRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllTradingRewardRequest,
    } as QueryAllTradingRewardRequest;
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

  fromJSON(object: any): QueryAllTradingRewardRequest {
    const message = {
      ...baseQueryAllTradingRewardRequest,
    } as QueryAllTradingRewardRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllTradingRewardRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllTradingRewardRequest>
  ): QueryAllTradingRewardRequest {
    const message = {
      ...baseQueryAllTradingRewardRequest,
    } as QueryAllTradingRewardRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllTradingRewardResponse: object = {};

export const QueryAllTradingRewardResponse = {
  encode(
    message: QueryAllTradingRewardResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.tradingReward) {
      TradingReward.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryAllTradingRewardResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllTradingRewardResponse,
    } as QueryAllTradingRewardResponse;
    message.tradingReward = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tradingReward.push(
            TradingReward.decode(reader, reader.uint32())
          );
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

  fromJSON(object: any): QueryAllTradingRewardResponse {
    const message = {
      ...baseQueryAllTradingRewardResponse,
    } as QueryAllTradingRewardResponse;
    message.tradingReward = [];
    if (object.tradingReward !== undefined && object.tradingReward !== null) {
      for (const e of object.tradingReward) {
        message.tradingReward.push(TradingReward.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllTradingRewardResponse): unknown {
    const obj: any = {};
    if (message.tradingReward) {
      obj.tradingReward = message.tradingReward.map((e) =>
        e ? TradingReward.toJSON(e) : undefined
      );
    } else {
      obj.tradingReward = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllTradingRewardResponse>
  ): QueryAllTradingRewardResponse {
    const message = {
      ...baseQueryAllTradingRewardResponse,
    } as QueryAllTradingRewardResponse;
    message.tradingReward = [];
    if (object.tradingReward !== undefined && object.tradingReward !== null) {
      for (const e of object.tradingReward) {
        message.tradingReward.push(TradingReward.fromPartial(e));
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
  /** Parameters queries the parameters of the module. */
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
  /** Queries a StakingReward by index. */
  StakingReward(
    request: QueryGetStakingRewardRequest
  ): Promise<QueryGetStakingRewardResponse>;
  /** Queries a list of StakingReward items. */
  StakingRewardAll(
    request: QueryAllStakingRewardRequest
  ): Promise<QueryAllStakingRewardResponse>;
  /** Queries a TradingReward by index. */
  TradingReward(
    request: QueryGetTradingRewardRequest
  ): Promise<QueryGetTradingRewardResponse>;
  /** Queries a list of TradingReward items. */
  TradingRewardAll(
    request: QueryAllTradingRewardRequest
  ): Promise<QueryAllTradingRewardResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request("bze.v1.rewards.Query", "Params", data);
    return promise.then((data) => QueryParamsResponse.decode(new Reader(data)));
  }

  StakingReward(
    request: QueryGetStakingRewardRequest
  ): Promise<QueryGetStakingRewardResponse> {
    const data = QueryGetStakingRewardRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.v1.rewards.Query",
      "StakingReward",
      data
    );
    return promise.then((data) =>
      QueryGetStakingRewardResponse.decode(new Reader(data))
    );
  }

  StakingRewardAll(
    request: QueryAllStakingRewardRequest
  ): Promise<QueryAllStakingRewardResponse> {
    const data = QueryAllStakingRewardRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.v1.rewards.Query",
      "StakingRewardAll",
      data
    );
    return promise.then((data) =>
      QueryAllStakingRewardResponse.decode(new Reader(data))
    );
  }

  TradingReward(
    request: QueryGetTradingRewardRequest
  ): Promise<QueryGetTradingRewardResponse> {
    const data = QueryGetTradingRewardRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.v1.rewards.Query",
      "TradingReward",
      data
    );
    return promise.then((data) =>
      QueryGetTradingRewardResponse.decode(new Reader(data))
    );
  }

  TradingRewardAll(
    request: QueryAllTradingRewardRequest
  ): Promise<QueryAllTradingRewardResponse> {
    const data = QueryAllTradingRewardRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.v1.rewards.Query",
      "TradingRewardAll",
      data
    );
    return promise.then((data) =>
      QueryAllTradingRewardResponse.decode(new Reader(data))
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
