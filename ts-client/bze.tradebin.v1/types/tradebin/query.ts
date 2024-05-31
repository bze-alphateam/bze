/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Params } from "../tradebin/params";
import { Market } from "../tradebin/market";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import {
  OrderReference,
  AggregatedOrder,
  HistoryOrder,
  Order,
} from "../tradebin/order";

export const protobufPackage = "bze.tradebin.v1";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params: Params | undefined;
}

export interface QueryGetMarketRequest {
  base: string;
  quote: string;
}

export interface QueryGetMarketResponse {
  market: Market | undefined;
}

export interface QueryAllMarketRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllMarketResponse {
  market: Market[];
  pagination: PageResponse | undefined;
}

export interface QueryAssetMarketsRequest {
  asset: string;
}

export interface QueryAssetMarketsResponse {
  base: Market[];
  quote: Market[];
}

export interface QueryUserMarketOrdersRequest {
  address: string;
  market: string;
  pagination: PageRequest | undefined;
}

export interface QueryUserMarketOrdersResponse {
  list: OrderReference[];
  pagination: PageResponse | undefined;
}

export interface QueryMarketAggregatedOrdersRequest {
  market: string;
  orderType: string;
  pagination: PageRequest | undefined;
}

export interface QueryMarketAggregatedOrdersResponse {
  list: AggregatedOrder[];
  pagination: PageResponse | undefined;
}

export interface QueryMarketHistoryRequest {
  market: string;
  pagination: PageRequest | undefined;
}

export interface QueryMarketHistoryResponse {
  list: HistoryOrder[];
  pagination: PageResponse | undefined;
}

export interface QueryMarketOrderRequest {
  market: string;
  orderType: string;
  orderId: string;
}

export interface QueryMarketOrderResponse {
  order: Order | undefined;
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

const baseQueryGetMarketRequest: object = { base: "", quote: "" };

export const QueryGetMarketRequest = {
  encode(
    message: QueryGetMarketRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.base !== "") {
      writer.uint32(10).string(message.base);
    }
    if (message.quote !== "") {
      writer.uint32(18).string(message.quote);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGetMarketRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryGetMarketRequest } as QueryGetMarketRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.base = reader.string();
          break;
        case 2:
          message.quote = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetMarketRequest {
    const message = { ...baseQueryGetMarketRequest } as QueryGetMarketRequest;
    if (object.base !== undefined && object.base !== null) {
      message.base = String(object.base);
    } else {
      message.base = "";
    }
    if (object.quote !== undefined && object.quote !== null) {
      message.quote = String(object.quote);
    } else {
      message.quote = "";
    }
    return message;
  },

  toJSON(message: QueryGetMarketRequest): unknown {
    const obj: any = {};
    message.base !== undefined && (obj.base = message.base);
    message.quote !== undefined && (obj.quote = message.quote);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetMarketRequest>
  ): QueryGetMarketRequest {
    const message = { ...baseQueryGetMarketRequest } as QueryGetMarketRequest;
    if (object.base !== undefined && object.base !== null) {
      message.base = object.base;
    } else {
      message.base = "";
    }
    if (object.quote !== undefined && object.quote !== null) {
      message.quote = object.quote;
    } else {
      message.quote = "";
    }
    return message;
  },
};

const baseQueryGetMarketResponse: object = {};

export const QueryGetMarketResponse = {
  encode(
    message: QueryGetMarketResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.market !== undefined) {
      Market.encode(message.market, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGetMarketResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryGetMarketResponse } as QueryGetMarketResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.market = Market.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetMarketResponse {
    const message = { ...baseQueryGetMarketResponse } as QueryGetMarketResponse;
    if (object.market !== undefined && object.market !== null) {
      message.market = Market.fromJSON(object.market);
    } else {
      message.market = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetMarketResponse): unknown {
    const obj: any = {};
    message.market !== undefined &&
      (obj.market = message.market ? Market.toJSON(message.market) : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetMarketResponse>
  ): QueryGetMarketResponse {
    const message = { ...baseQueryGetMarketResponse } as QueryGetMarketResponse;
    if (object.market !== undefined && object.market !== null) {
      message.market = Market.fromPartial(object.market);
    } else {
      message.market = undefined;
    }
    return message;
  },
};

const baseQueryAllMarketRequest: object = {};

export const QueryAllMarketRequest = {
  encode(
    message: QueryAllMarketRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryAllMarketRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryAllMarketRequest } as QueryAllMarketRequest;
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

  fromJSON(object: any): QueryAllMarketRequest {
    const message = { ...baseQueryAllMarketRequest } as QueryAllMarketRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllMarketRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllMarketRequest>
  ): QueryAllMarketRequest {
    const message = { ...baseQueryAllMarketRequest } as QueryAllMarketRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllMarketResponse: object = {};

export const QueryAllMarketResponse = {
  encode(
    message: QueryAllMarketResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.market) {
      Market.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryAllMarketResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryAllMarketResponse } as QueryAllMarketResponse;
    message.market = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.market.push(Market.decode(reader, reader.uint32()));
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

  fromJSON(object: any): QueryAllMarketResponse {
    const message = { ...baseQueryAllMarketResponse } as QueryAllMarketResponse;
    message.market = [];
    if (object.market !== undefined && object.market !== null) {
      for (const e of object.market) {
        message.market.push(Market.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllMarketResponse): unknown {
    const obj: any = {};
    if (message.market) {
      obj.market = message.market.map((e) =>
        e ? Market.toJSON(e) : undefined
      );
    } else {
      obj.market = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllMarketResponse>
  ): QueryAllMarketResponse {
    const message = { ...baseQueryAllMarketResponse } as QueryAllMarketResponse;
    message.market = [];
    if (object.market !== undefined && object.market !== null) {
      for (const e of object.market) {
        message.market.push(Market.fromPartial(e));
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

const baseQueryAssetMarketsRequest: object = { asset: "" };

export const QueryAssetMarketsRequest = {
  encode(
    message: QueryAssetMarketsRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.asset !== "") {
      writer.uint32(10).string(message.asset);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAssetMarketsRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAssetMarketsRequest,
    } as QueryAssetMarketsRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.asset = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAssetMarketsRequest {
    const message = {
      ...baseQueryAssetMarketsRequest,
    } as QueryAssetMarketsRequest;
    if (object.asset !== undefined && object.asset !== null) {
      message.asset = String(object.asset);
    } else {
      message.asset = "";
    }
    return message;
  },

  toJSON(message: QueryAssetMarketsRequest): unknown {
    const obj: any = {};
    message.asset !== undefined && (obj.asset = message.asset);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAssetMarketsRequest>
  ): QueryAssetMarketsRequest {
    const message = {
      ...baseQueryAssetMarketsRequest,
    } as QueryAssetMarketsRequest;
    if (object.asset !== undefined && object.asset !== null) {
      message.asset = object.asset;
    } else {
      message.asset = "";
    }
    return message;
  },
};

const baseQueryAssetMarketsResponse: object = {};

export const QueryAssetMarketsResponse = {
  encode(
    message: QueryAssetMarketsResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.base) {
      Market.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.quote) {
      Market.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAssetMarketsResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAssetMarketsResponse,
    } as QueryAssetMarketsResponse;
    message.base = [];
    message.quote = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.base.push(Market.decode(reader, reader.uint32()));
          break;
        case 2:
          message.quote.push(Market.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAssetMarketsResponse {
    const message = {
      ...baseQueryAssetMarketsResponse,
    } as QueryAssetMarketsResponse;
    message.base = [];
    message.quote = [];
    if (object.base !== undefined && object.base !== null) {
      for (const e of object.base) {
        message.base.push(Market.fromJSON(e));
      }
    }
    if (object.quote !== undefined && object.quote !== null) {
      for (const e of object.quote) {
        message.quote.push(Market.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: QueryAssetMarketsResponse): unknown {
    const obj: any = {};
    if (message.base) {
      obj.base = message.base.map((e) => (e ? Market.toJSON(e) : undefined));
    } else {
      obj.base = [];
    }
    if (message.quote) {
      obj.quote = message.quote.map((e) => (e ? Market.toJSON(e) : undefined));
    } else {
      obj.quote = [];
    }
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAssetMarketsResponse>
  ): QueryAssetMarketsResponse {
    const message = {
      ...baseQueryAssetMarketsResponse,
    } as QueryAssetMarketsResponse;
    message.base = [];
    message.quote = [];
    if (object.base !== undefined && object.base !== null) {
      for (const e of object.base) {
        message.base.push(Market.fromPartial(e));
      }
    }
    if (object.quote !== undefined && object.quote !== null) {
      for (const e of object.quote) {
        message.quote.push(Market.fromPartial(e));
      }
    }
    return message;
  },
};

const baseQueryUserMarketOrdersRequest: object = { address: "", market: "" };

export const QueryUserMarketOrdersRequest = {
  encode(
    message: QueryUserMarketOrdersRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.market !== "") {
      writer.uint32(18).string(message.market);
    }
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryUserMarketOrdersRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryUserMarketOrdersRequest,
    } as QueryUserMarketOrdersRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.address = reader.string();
          break;
        case 2:
          message.market = reader.string();
          break;
        case 3:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryUserMarketOrdersRequest {
    const message = {
      ...baseQueryUserMarketOrdersRequest,
    } as QueryUserMarketOrdersRequest;
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    if (object.market !== undefined && object.market !== null) {
      message.market = String(object.market);
    } else {
      message.market = "";
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryUserMarketOrdersRequest): unknown {
    const obj: any = {};
    message.address !== undefined && (obj.address = message.address);
    message.market !== undefined && (obj.market = message.market);
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryUserMarketOrdersRequest>
  ): QueryUserMarketOrdersRequest {
    const message = {
      ...baseQueryUserMarketOrdersRequest,
    } as QueryUserMarketOrdersRequest;
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    if (object.market !== undefined && object.market !== null) {
      message.market = object.market;
    } else {
      message.market = "";
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryUserMarketOrdersResponse: object = {};

export const QueryUserMarketOrdersResponse = {
  encode(
    message: QueryUserMarketOrdersResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.list) {
      OrderReference.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryUserMarketOrdersResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryUserMarketOrdersResponse,
    } as QueryUserMarketOrdersResponse;
    message.list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.list.push(OrderReference.decode(reader, reader.uint32()));
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

  fromJSON(object: any): QueryUserMarketOrdersResponse {
    const message = {
      ...baseQueryUserMarketOrdersResponse,
    } as QueryUserMarketOrdersResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(OrderReference.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryUserMarketOrdersResponse): unknown {
    const obj: any = {};
    if (message.list) {
      obj.list = message.list.map((e) =>
        e ? OrderReference.toJSON(e) : undefined
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
    object: DeepPartial<QueryUserMarketOrdersResponse>
  ): QueryUserMarketOrdersResponse {
    const message = {
      ...baseQueryUserMarketOrdersResponse,
    } as QueryUserMarketOrdersResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(OrderReference.fromPartial(e));
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

const baseQueryMarketAggregatedOrdersRequest: object = {
  market: "",
  orderType: "",
};

export const QueryMarketAggregatedOrdersRequest = {
  encode(
    message: QueryMarketAggregatedOrdersRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.market !== "") {
      writer.uint32(10).string(message.market);
    }
    if (message.orderType !== "") {
      writer.uint32(18).string(message.orderType);
    }
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryMarketAggregatedOrdersRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryMarketAggregatedOrdersRequest,
    } as QueryMarketAggregatedOrdersRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.market = reader.string();
          break;
        case 2:
          message.orderType = reader.string();
          break;
        case 3:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryMarketAggregatedOrdersRequest {
    const message = {
      ...baseQueryMarketAggregatedOrdersRequest,
    } as QueryMarketAggregatedOrdersRequest;
    if (object.market !== undefined && object.market !== null) {
      message.market = String(object.market);
    } else {
      message.market = "";
    }
    if (object.orderType !== undefined && object.orderType !== null) {
      message.orderType = String(object.orderType);
    } else {
      message.orderType = "";
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryMarketAggregatedOrdersRequest): unknown {
    const obj: any = {};
    message.market !== undefined && (obj.market = message.market);
    message.orderType !== undefined && (obj.orderType = message.orderType);
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryMarketAggregatedOrdersRequest>
  ): QueryMarketAggregatedOrdersRequest {
    const message = {
      ...baseQueryMarketAggregatedOrdersRequest,
    } as QueryMarketAggregatedOrdersRequest;
    if (object.market !== undefined && object.market !== null) {
      message.market = object.market;
    } else {
      message.market = "";
    }
    if (object.orderType !== undefined && object.orderType !== null) {
      message.orderType = object.orderType;
    } else {
      message.orderType = "";
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryMarketAggregatedOrdersResponse: object = {};

export const QueryMarketAggregatedOrdersResponse = {
  encode(
    message: QueryMarketAggregatedOrdersResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.list) {
      AggregatedOrder.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryMarketAggregatedOrdersResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryMarketAggregatedOrdersResponse,
    } as QueryMarketAggregatedOrdersResponse;
    message.list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.list.push(AggregatedOrder.decode(reader, reader.uint32()));
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

  fromJSON(object: any): QueryMarketAggregatedOrdersResponse {
    const message = {
      ...baseQueryMarketAggregatedOrdersResponse,
    } as QueryMarketAggregatedOrdersResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(AggregatedOrder.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryMarketAggregatedOrdersResponse): unknown {
    const obj: any = {};
    if (message.list) {
      obj.list = message.list.map((e) =>
        e ? AggregatedOrder.toJSON(e) : undefined
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
    object: DeepPartial<QueryMarketAggregatedOrdersResponse>
  ): QueryMarketAggregatedOrdersResponse {
    const message = {
      ...baseQueryMarketAggregatedOrdersResponse,
    } as QueryMarketAggregatedOrdersResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(AggregatedOrder.fromPartial(e));
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

const baseQueryMarketHistoryRequest: object = { market: "" };

export const QueryMarketHistoryRequest = {
  encode(
    message: QueryMarketHistoryRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.market !== "") {
      writer.uint32(10).string(message.market);
    }
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryMarketHistoryRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryMarketHistoryRequest,
    } as QueryMarketHistoryRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.market = reader.string();
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

  fromJSON(object: any): QueryMarketHistoryRequest {
    const message = {
      ...baseQueryMarketHistoryRequest,
    } as QueryMarketHistoryRequest;
    if (object.market !== undefined && object.market !== null) {
      message.market = String(object.market);
    } else {
      message.market = "";
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryMarketHistoryRequest): unknown {
    const obj: any = {};
    message.market !== undefined && (obj.market = message.market);
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryMarketHistoryRequest>
  ): QueryMarketHistoryRequest {
    const message = {
      ...baseQueryMarketHistoryRequest,
    } as QueryMarketHistoryRequest;
    if (object.market !== undefined && object.market !== null) {
      message.market = object.market;
    } else {
      message.market = "";
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryMarketHistoryResponse: object = {};

export const QueryMarketHistoryResponse = {
  encode(
    message: QueryMarketHistoryResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.list) {
      HistoryOrder.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryMarketHistoryResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryMarketHistoryResponse,
    } as QueryMarketHistoryResponse;
    message.list = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.list.push(HistoryOrder.decode(reader, reader.uint32()));
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

  fromJSON(object: any): QueryMarketHistoryResponse {
    const message = {
      ...baseQueryMarketHistoryResponse,
    } as QueryMarketHistoryResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(HistoryOrder.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryMarketHistoryResponse): unknown {
    const obj: any = {};
    if (message.list) {
      obj.list = message.list.map((e) =>
        e ? HistoryOrder.toJSON(e) : undefined
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
    object: DeepPartial<QueryMarketHistoryResponse>
  ): QueryMarketHistoryResponse {
    const message = {
      ...baseQueryMarketHistoryResponse,
    } as QueryMarketHistoryResponse;
    message.list = [];
    if (object.list !== undefined && object.list !== null) {
      for (const e of object.list) {
        message.list.push(HistoryOrder.fromPartial(e));
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

const baseQueryMarketOrderRequest: object = {
  market: "",
  orderType: "",
  orderId: "",
};

export const QueryMarketOrderRequest = {
  encode(
    message: QueryMarketOrderRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.market !== "") {
      writer.uint32(10).string(message.market);
    }
    if (message.orderType !== "") {
      writer.uint32(18).string(message.orderType);
    }
    if (message.orderId !== "") {
      writer.uint32(26).string(message.orderId);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryMarketOrderRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryMarketOrderRequest,
    } as QueryMarketOrderRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.market = reader.string();
          break;
        case 2:
          message.orderType = reader.string();
          break;
        case 3:
          message.orderId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryMarketOrderRequest {
    const message = {
      ...baseQueryMarketOrderRequest,
    } as QueryMarketOrderRequest;
    if (object.market !== undefined && object.market !== null) {
      message.market = String(object.market);
    } else {
      message.market = "";
    }
    if (object.orderType !== undefined && object.orderType !== null) {
      message.orderType = String(object.orderType);
    } else {
      message.orderType = "";
    }
    if (object.orderId !== undefined && object.orderId !== null) {
      message.orderId = String(object.orderId);
    } else {
      message.orderId = "";
    }
    return message;
  },

  toJSON(message: QueryMarketOrderRequest): unknown {
    const obj: any = {};
    message.market !== undefined && (obj.market = message.market);
    message.orderType !== undefined && (obj.orderType = message.orderType);
    message.orderId !== undefined && (obj.orderId = message.orderId);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryMarketOrderRequest>
  ): QueryMarketOrderRequest {
    const message = {
      ...baseQueryMarketOrderRequest,
    } as QueryMarketOrderRequest;
    if (object.market !== undefined && object.market !== null) {
      message.market = object.market;
    } else {
      message.market = "";
    }
    if (object.orderType !== undefined && object.orderType !== null) {
      message.orderType = object.orderType;
    } else {
      message.orderType = "";
    }
    if (object.orderId !== undefined && object.orderId !== null) {
      message.orderId = object.orderId;
    } else {
      message.orderId = "";
    }
    return message;
  },
};

const baseQueryMarketOrderResponse: object = {};

export const QueryMarketOrderResponse = {
  encode(
    message: QueryMarketOrderResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.order !== undefined) {
      Order.encode(message.order, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryMarketOrderResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryMarketOrderResponse,
    } as QueryMarketOrderResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.order = Order.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryMarketOrderResponse {
    const message = {
      ...baseQueryMarketOrderResponse,
    } as QueryMarketOrderResponse;
    if (object.order !== undefined && object.order !== null) {
      message.order = Order.fromJSON(object.order);
    } else {
      message.order = undefined;
    }
    return message;
  },

  toJSON(message: QueryMarketOrderResponse): unknown {
    const obj: any = {};
    message.order !== undefined &&
      (obj.order = message.order ? Order.toJSON(message.order) : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryMarketOrderResponse>
  ): QueryMarketOrderResponse {
    const message = {
      ...baseQueryMarketOrderResponse,
    } as QueryMarketOrderResponse;
    if (object.order !== undefined && object.order !== null) {
      message.order = Order.fromPartial(object.order);
    } else {
      message.order = undefined;
    }
    return message;
  },
};

/** Query defines the gRPC querier service. */
export interface Query {
  /** Parameters queries the parameters of the module. */
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse>;
  /** Queries a Market by index. */
  Market(request: QueryGetMarketRequest): Promise<QueryGetMarketResponse>;
  /** Queries a list of Market items. */
  MarketAll(request: QueryAllMarketRequest): Promise<QueryAllMarketResponse>;
  /** Queries a list of AssetMarkets items. */
  AssetMarkets(
    request: QueryAssetMarketsRequest
  ): Promise<QueryAssetMarketsResponse>;
  /** Queries a list of UserMarketOrders items. */
  UserMarketOrders(
    request: QueryUserMarketOrdersRequest
  ): Promise<QueryUserMarketOrdersResponse>;
  /** Queries a list of MarketAggregatedOrders items. */
  MarketAggregatedOrders(
    request: QueryMarketAggregatedOrdersRequest
  ): Promise<QueryMarketAggregatedOrdersResponse>;
  /** Queries a list of MarketHistory items. */
  MarketHistory(
    request: QueryMarketHistoryRequest
  ): Promise<QueryMarketHistoryResponse>;
  /** Queries a list of MarketOrder items. */
  MarketOrder(
    request: QueryMarketOrderRequest
  ): Promise<QueryMarketOrderResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request("bze.tradebin.v1.Query", "Params", data);
    return promise.then((data) => QueryParamsResponse.decode(new Reader(data)));
  }

  Market(request: QueryGetMarketRequest): Promise<QueryGetMarketResponse> {
    const data = QueryGetMarketRequest.encode(request).finish();
    const promise = this.rpc.request("bze.tradebin.v1.Query", "Market", data);
    return promise.then((data) =>
      QueryGetMarketResponse.decode(new Reader(data))
    );
  }

  MarketAll(request: QueryAllMarketRequest): Promise<QueryAllMarketResponse> {
    const data = QueryAllMarketRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.tradebin.v1.Query",
      "MarketAll",
      data
    );
    return promise.then((data) =>
      QueryAllMarketResponse.decode(new Reader(data))
    );
  }

  AssetMarkets(
    request: QueryAssetMarketsRequest
  ): Promise<QueryAssetMarketsResponse> {
    const data = QueryAssetMarketsRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.tradebin.v1.Query",
      "AssetMarkets",
      data
    );
    return promise.then((data) =>
      QueryAssetMarketsResponse.decode(new Reader(data))
    );
  }

  UserMarketOrders(
    request: QueryUserMarketOrdersRequest
  ): Promise<QueryUserMarketOrdersResponse> {
    const data = QueryUserMarketOrdersRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.tradebin.v1.Query",
      "UserMarketOrders",
      data
    );
    return promise.then((data) =>
      QueryUserMarketOrdersResponse.decode(new Reader(data))
    );
  }

  MarketAggregatedOrders(
    request: QueryMarketAggregatedOrdersRequest
  ): Promise<QueryMarketAggregatedOrdersResponse> {
    const data = QueryMarketAggregatedOrdersRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.tradebin.v1.Query",
      "MarketAggregatedOrders",
      data
    );
    return promise.then((data) =>
      QueryMarketAggregatedOrdersResponse.decode(new Reader(data))
    );
  }

  MarketHistory(
    request: QueryMarketHistoryRequest
  ): Promise<QueryMarketHistoryResponse> {
    const data = QueryMarketHistoryRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.tradebin.v1.Query",
      "MarketHistory",
      data
    );
    return promise.then((data) =>
      QueryMarketHistoryResponse.decode(new Reader(data))
    );
  }

  MarketOrder(
    request: QueryMarketOrderRequest
  ): Promise<QueryMarketOrderResponse> {
    const data = QueryMarketOrderRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.tradebin.v1.Query",
      "MarketOrder",
      data
    );
    return promise.then((data) =>
      QueryMarketOrderResponse.decode(new Reader(data))
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
