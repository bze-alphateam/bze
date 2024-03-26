/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Params } from "../cointrunk/params";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import { AcceptedDomain } from "../cointrunk/accepted_domain";
import { Publisher } from "../cointrunk/publisher";
import { Article } from "../cointrunk/article";
import { AnonArticlesCounter } from "../cointrunk/anon_articles_counter";

export const protobufPackage = "bze.cointrunk.v1";

/** QueryParamsRequest is request type for the Query/Params RPC method. */
export interface QueryParamsRequest {}

/** QueryParamsResponse is response type for the Query/Params RPC method. */
export interface QueryParamsResponse {
  /** params holds all the parameters of this module. */
  params: Params | undefined;
}

export interface QueryAcceptedDomainRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAcceptedDomainResponse {
  acceptedDomain: AcceptedDomain[];
  pagination: PageResponse | undefined;
}

export interface QueryPublisherRequest {
  pagination: PageRequest | undefined;
}

export interface QueryPublisherResponse {
  publisher: Publisher[];
  pagination: PageResponse | undefined;
}

export interface QueryPublisherByIndexRequest {
  index: string;
}

export interface QueryPublisherByIndexResponse {
  publisher: Publisher | undefined;
}

export interface QueryAllArticlesRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllArticlesResponse {
  article: Article[];
  pagination: PageResponse | undefined;
}

export interface QueryAllAnonArticlesCountersRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllAnonArticlesCountersResponse {
  AnonArticlesCounters: AnonArticlesCounter[];
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

const baseQueryAcceptedDomainRequest: object = {};

export const QueryAcceptedDomainRequest = {
  encode(
    message: QueryAcceptedDomainRequest,
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
  ): QueryAcceptedDomainRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAcceptedDomainRequest,
    } as QueryAcceptedDomainRequest;
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

  fromJSON(object: any): QueryAcceptedDomainRequest {
    const message = {
      ...baseQueryAcceptedDomainRequest,
    } as QueryAcceptedDomainRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAcceptedDomainRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAcceptedDomainRequest>
  ): QueryAcceptedDomainRequest {
    const message = {
      ...baseQueryAcceptedDomainRequest,
    } as QueryAcceptedDomainRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAcceptedDomainResponse: object = {};

export const QueryAcceptedDomainResponse = {
  encode(
    message: QueryAcceptedDomainResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.acceptedDomain) {
      AcceptedDomain.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryAcceptedDomainResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAcceptedDomainResponse,
    } as QueryAcceptedDomainResponse;
    message.acceptedDomain = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.acceptedDomain.push(
            AcceptedDomain.decode(reader, reader.uint32())
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

  fromJSON(object: any): QueryAcceptedDomainResponse {
    const message = {
      ...baseQueryAcceptedDomainResponse,
    } as QueryAcceptedDomainResponse;
    message.acceptedDomain = [];
    if (object.acceptedDomain !== undefined && object.acceptedDomain !== null) {
      for (const e of object.acceptedDomain) {
        message.acceptedDomain.push(AcceptedDomain.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAcceptedDomainResponse): unknown {
    const obj: any = {};
    if (message.acceptedDomain) {
      obj.acceptedDomain = message.acceptedDomain.map((e) =>
        e ? AcceptedDomain.toJSON(e) : undefined
      );
    } else {
      obj.acceptedDomain = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAcceptedDomainResponse>
  ): QueryAcceptedDomainResponse {
    const message = {
      ...baseQueryAcceptedDomainResponse,
    } as QueryAcceptedDomainResponse;
    message.acceptedDomain = [];
    if (object.acceptedDomain !== undefined && object.acceptedDomain !== null) {
      for (const e of object.acceptedDomain) {
        message.acceptedDomain.push(AcceptedDomain.fromPartial(e));
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

const baseQueryPublisherRequest: object = {};

export const QueryPublisherRequest = {
  encode(
    message: QueryPublisherRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryPublisherRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryPublisherRequest } as QueryPublisherRequest;
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

  fromJSON(object: any): QueryPublisherRequest {
    const message = { ...baseQueryPublisherRequest } as QueryPublisherRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryPublisherRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryPublisherRequest>
  ): QueryPublisherRequest {
    const message = { ...baseQueryPublisherRequest } as QueryPublisherRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryPublisherResponse: object = {};

export const QueryPublisherResponse = {
  encode(
    message: QueryPublisherResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.publisher) {
      Publisher.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryPublisherResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryPublisherResponse } as QueryPublisherResponse;
    message.publisher = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.publisher.push(Publisher.decode(reader, reader.uint32()));
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

  fromJSON(object: any): QueryPublisherResponse {
    const message = { ...baseQueryPublisherResponse } as QueryPublisherResponse;
    message.publisher = [];
    if (object.publisher !== undefined && object.publisher !== null) {
      for (const e of object.publisher) {
        message.publisher.push(Publisher.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryPublisherResponse): unknown {
    const obj: any = {};
    if (message.publisher) {
      obj.publisher = message.publisher.map((e) =>
        e ? Publisher.toJSON(e) : undefined
      );
    } else {
      obj.publisher = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryPublisherResponse>
  ): QueryPublisherResponse {
    const message = { ...baseQueryPublisherResponse } as QueryPublisherResponse;
    message.publisher = [];
    if (object.publisher !== undefined && object.publisher !== null) {
      for (const e of object.publisher) {
        message.publisher.push(Publisher.fromPartial(e));
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

const baseQueryPublisherByIndexRequest: object = { index: "" };

export const QueryPublisherByIndexRequest = {
  encode(
    message: QueryPublisherByIndexRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryPublisherByIndexRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryPublisherByIndexRequest,
    } as QueryPublisherByIndexRequest;
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

  fromJSON(object: any): QueryPublisherByIndexRequest {
    const message = {
      ...baseQueryPublisherByIndexRequest,
    } as QueryPublisherByIndexRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    return message;
  },

  toJSON(message: QueryPublisherByIndexRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryPublisherByIndexRequest>
  ): QueryPublisherByIndexRequest {
    const message = {
      ...baseQueryPublisherByIndexRequest,
    } as QueryPublisherByIndexRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    return message;
  },
};

const baseQueryPublisherByIndexResponse: object = {};

export const QueryPublisherByIndexResponse = {
  encode(
    message: QueryPublisherByIndexResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.publisher !== undefined) {
      Publisher.encode(message.publisher, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryPublisherByIndexResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryPublisherByIndexResponse,
    } as QueryPublisherByIndexResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.publisher = Publisher.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryPublisherByIndexResponse {
    const message = {
      ...baseQueryPublisherByIndexResponse,
    } as QueryPublisherByIndexResponse;
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = Publisher.fromJSON(object.publisher);
    } else {
      message.publisher = undefined;
    }
    return message;
  },

  toJSON(message: QueryPublisherByIndexResponse): unknown {
    const obj: any = {};
    message.publisher !== undefined &&
      (obj.publisher = message.publisher
        ? Publisher.toJSON(message.publisher)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryPublisherByIndexResponse>
  ): QueryPublisherByIndexResponse {
    const message = {
      ...baseQueryPublisherByIndexResponse,
    } as QueryPublisherByIndexResponse;
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = Publisher.fromPartial(object.publisher);
    } else {
      message.publisher = undefined;
    }
    return message;
  },
};

const baseQueryAllArticlesRequest: object = {};

export const QueryAllArticlesRequest = {
  encode(
    message: QueryAllArticlesRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryAllArticlesRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllArticlesRequest,
    } as QueryAllArticlesRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
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

  fromJSON(object: any): QueryAllArticlesRequest {
    const message = {
      ...baseQueryAllArticlesRequest,
    } as QueryAllArticlesRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllArticlesRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllArticlesRequest>
  ): QueryAllArticlesRequest {
    const message = {
      ...baseQueryAllArticlesRequest,
    } as QueryAllArticlesRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllArticlesResponse: object = {};

export const QueryAllArticlesResponse = {
  encode(
    message: QueryAllArticlesResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.article) {
      Article.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryAllArticlesResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllArticlesResponse,
    } as QueryAllArticlesResponse;
    message.article = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.article.push(Article.decode(reader, reader.uint32()));
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

  fromJSON(object: any): QueryAllArticlesResponse {
    const message = {
      ...baseQueryAllArticlesResponse,
    } as QueryAllArticlesResponse;
    message.article = [];
    if (object.article !== undefined && object.article !== null) {
      for (const e of object.article) {
        message.article.push(Article.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllArticlesResponse): unknown {
    const obj: any = {};
    if (message.article) {
      obj.article = message.article.map((e) =>
        e ? Article.toJSON(e) : undefined
      );
    } else {
      obj.article = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllArticlesResponse>
  ): QueryAllArticlesResponse {
    const message = {
      ...baseQueryAllArticlesResponse,
    } as QueryAllArticlesResponse;
    message.article = [];
    if (object.article !== undefined && object.article !== null) {
      for (const e of object.article) {
        message.article.push(Article.fromPartial(e));
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

const baseQueryAllAnonArticlesCountersRequest: object = {};

export const QueryAllAnonArticlesCountersRequest = {
  encode(
    message: QueryAllAnonArticlesCountersRequest,
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
  ): QueryAllAnonArticlesCountersRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllAnonArticlesCountersRequest,
    } as QueryAllAnonArticlesCountersRequest;
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

  fromJSON(object: any): QueryAllAnonArticlesCountersRequest {
    const message = {
      ...baseQueryAllAnonArticlesCountersRequest,
    } as QueryAllAnonArticlesCountersRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllAnonArticlesCountersRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllAnonArticlesCountersRequest>
  ): QueryAllAnonArticlesCountersRequest {
    const message = {
      ...baseQueryAllAnonArticlesCountersRequest,
    } as QueryAllAnonArticlesCountersRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllAnonArticlesCountersResponse: object = {};

export const QueryAllAnonArticlesCountersResponse = {
  encode(
    message: QueryAllAnonArticlesCountersResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.AnonArticlesCounters) {
      AnonArticlesCounter.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryAllAnonArticlesCountersResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllAnonArticlesCountersResponse,
    } as QueryAllAnonArticlesCountersResponse;
    message.AnonArticlesCounters = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.AnonArticlesCounters.push(
            AnonArticlesCounter.decode(reader, reader.uint32())
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

  fromJSON(object: any): QueryAllAnonArticlesCountersResponse {
    const message = {
      ...baseQueryAllAnonArticlesCountersResponse,
    } as QueryAllAnonArticlesCountersResponse;
    message.AnonArticlesCounters = [];
    if (
      object.AnonArticlesCounters !== undefined &&
      object.AnonArticlesCounters !== null
    ) {
      for (const e of object.AnonArticlesCounters) {
        message.AnonArticlesCounters.push(AnonArticlesCounter.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllAnonArticlesCountersResponse): unknown {
    const obj: any = {};
    if (message.AnonArticlesCounters) {
      obj.AnonArticlesCounters = message.AnonArticlesCounters.map((e) =>
        e ? AnonArticlesCounter.toJSON(e) : undefined
      );
    } else {
      obj.AnonArticlesCounters = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllAnonArticlesCountersResponse>
  ): QueryAllAnonArticlesCountersResponse {
    const message = {
      ...baseQueryAllAnonArticlesCountersResponse,
    } as QueryAllAnonArticlesCountersResponse;
    message.AnonArticlesCounters = [];
    if (
      object.AnonArticlesCounters !== undefined &&
      object.AnonArticlesCounters !== null
    ) {
      for (const e of object.AnonArticlesCounters) {
        message.AnonArticlesCounters.push(AnonArticlesCounter.fromPartial(e));
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
  /** Queries a list of AcceptedDomain items. */
  AcceptedDomain(
    request: QueryAcceptedDomainRequest
  ): Promise<QueryAcceptedDomainResponse>;
  /** Queries a list of Publisher items. */
  Publisher(request: QueryPublisherRequest): Promise<QueryPublisherResponse>;
  /** Queries publisher by index/address. */
  PublisherByIndex(
    request: QueryPublisherByIndexRequest
  ): Promise<QueryPublisherByIndexResponse>;
  /** Queries a list of Article items. */
  AllArticles(
    request: QueryAllArticlesRequest
  ): Promise<QueryAllArticlesResponse>;
  /** Queries a list of AllAnonArticlesCounters items. */
  AllAnonArticlesCounters(
    request: QueryAllAnonArticlesCountersRequest
  ): Promise<QueryAllAnonArticlesCountersResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  Params(request: QueryParamsRequest): Promise<QueryParamsResponse> {
    const data = QueryParamsRequest.encode(request).finish();
    const promise = this.rpc.request("bze.cointrunk.v1.Query", "Params", data);
    return promise.then((data) => QueryParamsResponse.decode(new Reader(data)));
  }

  AcceptedDomain(
    request: QueryAcceptedDomainRequest
  ): Promise<QueryAcceptedDomainResponse> {
    const data = QueryAcceptedDomainRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.cointrunk.v1.Query",
      "AcceptedDomain",
      data
    );
    return promise.then((data) =>
      QueryAcceptedDomainResponse.decode(new Reader(data))
    );
  }

  Publisher(request: QueryPublisherRequest): Promise<QueryPublisherResponse> {
    const data = QueryPublisherRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.cointrunk.v1.Query",
      "Publisher",
      data
    );
    return promise.then((data) =>
      QueryPublisherResponse.decode(new Reader(data))
    );
  }

  PublisherByIndex(
    request: QueryPublisherByIndexRequest
  ): Promise<QueryPublisherByIndexResponse> {
    const data = QueryPublisherByIndexRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.cointrunk.v1.Query",
      "PublisherByIndex",
      data
    );
    return promise.then((data) =>
      QueryPublisherByIndexResponse.decode(new Reader(data))
    );
  }

  AllArticles(
    request: QueryAllArticlesRequest
  ): Promise<QueryAllArticlesResponse> {
    const data = QueryAllArticlesRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.cointrunk.v1.Query",
      "AllArticles",
      data
    );
    return promise.then((data) =>
      QueryAllArticlesResponse.decode(new Reader(data))
    );
  }

  AllAnonArticlesCounters(
    request: QueryAllAnonArticlesCountersRequest
  ): Promise<QueryAllAnonArticlesCountersResponse> {
    const data = QueryAllAnonArticlesCountersRequest.encode(request).finish();
    const promise = this.rpc.request(
      "bze.cointrunk.v1.Query",
      "AllAnonArticlesCounters",
      data
    );
    return promise.then((data) =>
      QueryAllAnonArticlesCountersResponse.decode(new Reader(data))
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
