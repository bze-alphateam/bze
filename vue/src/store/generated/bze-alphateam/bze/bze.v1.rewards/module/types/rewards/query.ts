/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Params } from "../rewards/params";
import { StakingReward } from "../rewards/staking_reward";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import {
  TradingReward,
  TradingRewardLeaderboard,
} from "../rewards/trading_reward";
import { StakingRewardParticipant } from "../rewards/staking_reward_participant";

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
  reward_id: string;
}

export interface QueryGetTradingRewardResponse {
  trading_reward: TradingReward | undefined;
}

export interface QueryAllTradingRewardRequest {
  state: string;
  pagination: PageRequest | undefined;
}

export interface QueryAllTradingRewardResponse {
  list: TradingReward[];
  pagination: PageResponse | undefined;
}

export interface QueryGetStakingRewardParticipantRequest {
  address: string;
  pagination: PageRequest | undefined;
}

export interface QueryGetStakingRewardParticipantResponse {
  list: StakingRewardParticipant[];
  pagination: PageResponse | undefined;
}

export interface QueryAllStakingRewardParticipantRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllStakingRewardParticipantResponse {
  list: StakingRewardParticipant[];
  pagination: PageResponse | undefined;
}

export interface QueryGetTradingRewardLeaderboardRequest {
  reward_id: string;
}

export interface QueryGetTradingRewardLeaderboardResponse {
  leaderboard: TradingRewardLeaderboard | undefined;
}

export interface QueryGetMarketIdTradingRewardIdHandlerRequest {
  marketId: string;
}

export interface QueryGetMarketIdTradingRewardIdHandlerResponse {}

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

const baseQueryGetTradingRewardRequest: object = { reward_id: "" };

export const QueryGetTradingRewardRequest = {
  encode(
    message: QueryGetTradingRewardRequest,
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
          message.reward_id = reader.string();
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
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = String(object.reward_id);
    } else {
      message.reward_id = "";
    }
    return message;
  },

  toJSON(message: QueryGetTradingRewardRequest): unknown {
    const obj: any = {};
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetTradingRewardRequest>
  ): QueryGetTradingRewardRequest {
    const message = {
      ...baseQueryGetTradingRewardRequest,
    } as QueryGetTradingRewardRequest;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = object.reward_id;
    } else {
      message.reward_id = "";
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
    if (message.trading_reward !== undefined) {
      TradingReward.encode(
        message.trading_reward,
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
          message.trading_reward = TradingReward.decode(
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

  fromJSON(object: any): QueryGetTradingRewardResponse {
    const message = {
      ...baseQueryGetTradingRewardResponse,
    } as QueryGetTradingRewardResponse;
    if (object.trading_reward !== undefined && object.trading_reward !== null) {
      message.trading_reward = TradingReward.fromJSON(object.trading_reward);
    } else {
      message.trading_reward = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetTradingRewardResponse): unknown {
    const obj: any = {};
    message.trading_reward !== undefined &&
      (obj.trading_reward = message.trading_reward
        ? TradingReward.toJSON(message.trading_reward)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetTradingRewardResponse>
  ): QueryGetTradingRewardResponse {
    const message = {
      ...baseQueryGetTradingRewardResponse,
    } as QueryGetTradingRewardResponse;
    if (object.trading_reward !== undefined && object.trading_reward !== null) {
      message.trading_reward = TradingReward.fromPartial(object.trading_reward);
    } else {
      message.trading_reward = undefined;
    }
    return message;
  },
};

const baseQueryAllTradingRewardRequest: object = { state: "" };

export const QueryAllTradingRewardRequest = {
  encode(
    message: QueryAllTradingRewardRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.state !== "") {
      writer.uint32(10).string(message.state);
    }
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(18).fork()).ldelim();
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
          message.state = reader.string();
          break;
        case 2:
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
    if (object.state !== undefined && object.state !== null) {
      message.state = String(object.state);
    } else {
      message.state = "";
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllTradingRewardRequest): unknown {
    const obj: any = {};
    message.state !== undefined && (obj.state = message.state);
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
    if (object.state !== undefined && object.state !== null) {
      message.state = object.state;
    } else {
      message.state = "";
    }
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
    for (const v of message.list) {
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
    message.list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.list.push(TradingReward.decode(reader, reader.uint32()));
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
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(TradingReward.fromJSON(e));
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
    if (message.list) {
      obj.list = message.list.map((e) =>
        e ? TradingReward.toJSON(e) : undefined
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
    object: DeepPartial<QueryAllTradingRewardResponse>
  ): QueryAllTradingRewardResponse {
    const message = {
      ...baseQueryAllTradingRewardResponse,
    } as QueryAllTradingRewardResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(TradingReward.fromPartial(e));
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

const baseQueryGetStakingRewardParticipantRequest: object = { address: "" };

export const QueryGetStakingRewardParticipantRequest = {
  encode(
    message: QueryGetStakingRewardParticipantRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetStakingRewardParticipantRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetStakingRewardParticipantRequest,
    } as QueryGetStakingRewardParticipantRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.address = reader.string();
          break;
        case 2:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetStakingRewardParticipantRequest {
    const message = {
      ...baseQueryGetStakingRewardParticipantRequest,
    } as QueryGetStakingRewardParticipantRequest;
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetStakingRewardParticipantRequest): unknown {
    const obj: any = {};
    message.address !== undefined && (obj.address = message.address);
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetStakingRewardParticipantRequest>
  ): QueryGetStakingRewardParticipantRequest {
    const message = {
      ...baseQueryGetStakingRewardParticipantRequest,
    } as QueryGetStakingRewardParticipantRequest;
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryGetStakingRewardParticipantResponse: object = {};

export const QueryGetStakingRewardParticipantResponse = {
  encode(
    message: QueryGetStakingRewardParticipantResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.list) {
      StakingRewardParticipant.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryGetStakingRewardParticipantResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetStakingRewardParticipantResponse,
    } as QueryGetStakingRewardParticipantResponse;
    message.list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.list.push(
            StakingRewardParticipant.decode(reader, reader.uint32())
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

  fromJSON(object: any): QueryGetStakingRewardParticipantResponse {
    const message = {
      ...baseQueryGetStakingRewardParticipantResponse,
    } as QueryGetStakingRewardParticipantResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(StakingRewardParticipant.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetStakingRewardParticipantResponse): unknown {
    const obj: any = {};
    if (message.list) {
      obj.list = message.list.map((e) =>
        e ? StakingRewardParticipant.toJSON(e) : undefined
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
    object: DeepPartial<QueryGetStakingRewardParticipantResponse>
  ): QueryGetStakingRewardParticipantResponse {
    const message = {
      ...baseQueryGetStakingRewardParticipantResponse,
    } as QueryGetStakingRewardParticipantResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(StakingRewardParticipant.fromPartial(e));
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

const baseQueryAllStakingRewardParticipantRequest: object = {};

export const QueryAllStakingRewardParticipantRequest = {
  encode(
    message: QueryAllStakingRewardParticipantRequest,
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
  ): QueryAllStakingRewardParticipantRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllStakingRewardParticipantRequest,
    } as QueryAllStakingRewardParticipantRequest;
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

  fromJSON(object: any): QueryAllStakingRewardParticipantRequest {
    const message = {
      ...baseQueryAllStakingRewardParticipantRequest,
    } as QueryAllStakingRewardParticipantRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllStakingRewardParticipantRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllStakingRewardParticipantRequest>
  ): QueryAllStakingRewardParticipantRequest {
    const message = {
      ...baseQueryAllStakingRewardParticipantRequest,
    } as QueryAllStakingRewardParticipantRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllStakingRewardParticipantResponse: object = {};

export const QueryAllStakingRewardParticipantResponse = {
  encode(
    message: QueryAllStakingRewardParticipantResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.list) {
      StakingRewardParticipant.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryAllStakingRewardParticipantResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllStakingRewardParticipantResponse,
    } as QueryAllStakingRewardParticipantResponse;
    message.list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.list.push(
            StakingRewardParticipant.decode(reader, reader.uint32())
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

  fromJSON(object: any): QueryAllStakingRewardParticipantResponse {
    const message = {
      ...baseQueryAllStakingRewardParticipantResponse,
    } as QueryAllStakingRewardParticipantResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(StakingRewardParticipant.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllStakingRewardParticipantResponse): unknown {
    const obj: any = {};
    if (message.list) {
      obj.list = message.list.map((e) =>
        e ? StakingRewardParticipant.toJSON(e) : undefined
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
    object: DeepPartial<QueryAllStakingRewardParticipantResponse>
  ): QueryAllStakingRewardParticipantResponse {
    const message = {
      ...baseQueryAllStakingRewardParticipantResponse,
    } as QueryAllStakingRewardParticipantResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(StakingRewardParticipant.fromPartial(e));
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

const baseQueryGetTradingRewardLeaderboardRequest: object = { reward_id: "" };

export const QueryGetTradingRewardLeaderboardRequest = {
  encode(
    message: QueryGetTradingRewardLeaderboardRequest,
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
  ): QueryGetTradingRewardLeaderboardRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetTradingRewardLeaderboardRequest,
    } as QueryGetTradingRewardLeaderboardRequest;
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

  fromJSON(object: any): QueryGetTradingRewardLeaderboardRequest {
    const message = {
      ...baseQueryGetTradingRewardLeaderboardRequest,
    } as QueryGetTradingRewardLeaderboardRequest;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = String(object.reward_id);
    } else {
      message.reward_id = "";
    }
    return message;
  },

  toJSON(message: QueryGetTradingRewardLeaderboardRequest): unknown {
    const obj: any = {};
    message.reward_id !== undefined && (obj.reward_id = message.reward_id);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetTradingRewardLeaderboardRequest>
  ): QueryGetTradingRewardLeaderboardRequest {
    const message = {
      ...baseQueryGetTradingRewardLeaderboardRequest,
    } as QueryGetTradingRewardLeaderboardRequest;
    if (object.reward_id !== undefined && object.reward_id !== null) {
      message.reward_id = object.reward_id;
    } else {
      message.reward_id = "";
    }
    return message;
  },
};

const baseQueryGetTradingRewardLeaderboardResponse: object = {};

export const QueryGetTradingRewardLeaderboardResponse = {
  encode(
    message: QueryGetTradingRewardLeaderboardResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.leaderboard !== undefined) {
      TradingRewardLeaderboard.encode(
        message.leaderboard,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetTradingRewardLeaderboardResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetTradingRewardLeaderboardResponse,
    } as QueryGetTradingRewardLeaderboardResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.leaderboard = TradingRewardLeaderboard.decode(
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

  fromJSON(object: any): QueryGetTradingRewardLeaderboardResponse {
    const message = {
      ...baseQueryGetTradingRewardLeaderboardResponse,
    } as QueryGetTradingRewardLeaderboardResponse;
    if (object.leaderboard !== undefined && object.leaderboard !== null) {
      message.leaderboard = TradingRewardLeaderboard.fromJSON(
        object.leaderboard
      );
    } else {
      message.leaderboard = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetTradingRewardLeaderboardResponse): unknown {
    const obj: any = {};
    message.leaderboard !== undefined &&
      (obj.leaderboard = message.leaderboard
        ? TradingRewardLeaderboard.toJSON(message.leaderboard)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetTradingRewardLeaderboardResponse>
  ): QueryGetTradingRewardLeaderboardResponse {
    const message = {
      ...baseQueryGetTradingRewardLeaderboardResponse,
    } as QueryGetTradingRewardLeaderboardResponse;
    if (object.leaderboard !== undefined && object.leaderboard !== null) {
      message.leaderboard = TradingRewardLeaderboard.fromPartial(
        object.leaderboard
      );
    } else {
      message.leaderboard = undefined;
    }
    return message;
  },
};

const baseQueryGetMarketIdTradingRewardIdHandlerRequest: object = {
  marketId: "",
};

export const QueryGetMarketIdTradingRewardIdHandlerRequest = {
  encode(
    message: QueryGetMarketIdTradingRewardIdHandlerRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.marketId !== "") {
      writer.uint32(10).string(message.marketId);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetMarketIdTradingRewardIdHandlerRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetMarketIdTradingRewardIdHandlerRequest,
    } as QueryGetMarketIdTradingRewardIdHandlerRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.marketId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetMarketIdTradingRewardIdHandlerRequest {
    const message = {
      ...baseQueryGetMarketIdTradingRewardIdHandlerRequest,
    } as QueryGetMarketIdTradingRewardIdHandlerRequest;
    if (object.marketId !== undefined && object.marketId !== null) {
      message.marketId = String(object.marketId);
    } else {
      message.marketId = "";
    }
    return message;
  },

  toJSON(message: QueryGetMarketIdTradingRewardIdHandlerRequest): unknown {
    const obj: any = {};
    message.marketId !== undefined && (obj.marketId = message.marketId);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetMarketIdTradingRewardIdHandlerRequest>
  ): QueryGetMarketIdTradingRewardIdHandlerRequest {
    const message = {
      ...baseQueryGetMarketIdTradingRewardIdHandlerRequest,
    } as QueryGetMarketIdTradingRewardIdHandlerRequest;
    if (object.marketId !== undefined && object.marketId !== null) {
      message.marketId = object.marketId;
    } else {
      message.marketId = "";
    }
    return message;
  },
};

const baseQueryGetMarketIdTradingRewardIdHandlerResponse: object = {};

export const QueryGetMarketIdTradingRewardIdHandlerResponse = {
  encode(
    _: QueryGetMarketIdTradingRewardIdHandlerResponse,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetMarketIdTradingRewardIdHandlerResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetMarketIdTradingRewardIdHandlerResponse,
    } as QueryGetMarketIdTradingRewardIdHandlerResponse;
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

  fromJSON(_: any): QueryGetMarketIdTradingRewardIdHandlerResponse {
    const message = {
      ...baseQueryGetMarketIdTradingRewardIdHandlerResponse,
    } as QueryGetMarketIdTradingRewardIdHandlerResponse;
    return message;
  },

  toJSON(_: QueryGetMarketIdTradingRewardIdHandlerResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<QueryGetMarketIdTradingRewardIdHandlerResponse>
  ): QueryGetMarketIdTradingRewardIdHandlerResponse {
    const message = {
      ...baseQueryGetMarketIdTradingRewardIdHandlerResponse,
    } as QueryGetMarketIdTradingRewardIdHandlerResponse;
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
  /** Queries a StakingRewardParticipant by index. */
  StakingRewardParticipant(
    request: QueryGetStakingRewardParticipantRequest
  ): Promise<QueryGetStakingRewardParticipantResponse>;
  /** Queries a list of StakingRewardParticipant items. */
  StakingRewardParticipantAll(
    request: QueryAllStakingRewardParticipantRequest
  ): Promise<QueryAllStakingRewardParticipantResponse>;
  /** Queries a list of GetTradingRewardLeaderboard items. */
  GetTradingRewardLeaderboardHandler(
    request: QueryGetTradingRewardLeaderboardRequest
  ): Promise<QueryGetTradingRewardLeaderboardResponse>;
  /** Queries a list of GetMarketIdTradingRewardIdHandler items. */
  GetMarketIdTradingRewardIdHandler(
    request: QueryGetMarketIdTradingRewardIdHandlerRequest
  ): Promise<QueryGetMarketIdTradingRewardIdHandlerResponse>;
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

  StakingRewardParticipant(
    request: QueryGetStakingRewardParticipantRequest
  ): Promise<QueryGetStakingRewardParticipantResponse> {
    const data = QueryGetStakingRewardParticipantRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "bze.v1.rewards.Query",
      "StakingRewardParticipant",
      data
    );
    return promise.then((data) =>
      QueryGetStakingRewardParticipantResponse.decode(new Reader(data))
    );
  }

  StakingRewardParticipantAll(
    request: QueryAllStakingRewardParticipantRequest
  ): Promise<QueryAllStakingRewardParticipantResponse> {
    const data = QueryAllStakingRewardParticipantRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "bze.v1.rewards.Query",
      "StakingRewardParticipantAll",
      data
    );
    return promise.then((data) =>
      QueryAllStakingRewardParticipantResponse.decode(new Reader(data))
    );
  }

  GetTradingRewardLeaderboardHandler(
    request: QueryGetTradingRewardLeaderboardRequest
  ): Promise<QueryGetTradingRewardLeaderboardResponse> {
    const data = QueryGetTradingRewardLeaderboardRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "bze.v1.rewards.Query",
      "GetTradingRewardLeaderboardHandler",
      data
    );
    return promise.then((data) =>
      QueryGetTradingRewardLeaderboardResponse.decode(new Reader(data))
    );
  }

  GetMarketIdTradingRewardIdHandler(
    request: QueryGetMarketIdTradingRewardIdHandlerRequest
  ): Promise<QueryGetMarketIdTradingRewardIdHandlerResponse> {
    const data = QueryGetMarketIdTradingRewardIdHandlerRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "bze.v1.rewards.Query",
      "GetMarketIdTradingRewardIdHandler",
      data
    );
    return promise.then((data) =>
      QueryGetMarketIdTradingRewardIdHandlerResponse.decode(new Reader(data))
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
