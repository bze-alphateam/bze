/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Publisher } from "../cointrunk/publisher";
import { AcceptedDomain } from "../cointrunk/accepted_domain";

export const protobufPackage = "bze.cointrunk.v1";

export interface ArticleAddedEvent {
  publisher: string;
  article_id: number;
  paid: boolean;
}

export interface PublisherAddedEvent {
  publisher: Publisher | undefined;
}

export interface PublisherUpdatedEvent {
  publisher: Publisher | undefined;
}

export interface AcceptedDomainAddedEvent {
  accepted_domain: AcceptedDomain | undefined;
}

export interface AcceptedDomainUpdatedEvent {
  accepted_domain: AcceptedDomain | undefined;
}

const baseArticleAddedEvent: object = {
  publisher: "",
  article_id: 0,
  paid: false,
};

export const ArticleAddedEvent = {
  encode(message: ArticleAddedEvent, writer: Writer = Writer.create()): Writer {
    if (message.publisher !== "") {
      writer.uint32(10).string(message.publisher);
    }
    if (message.article_id !== 0) {
      writer.uint32(16).uint64(message.article_id);
    }
    if (message.paid === true) {
      writer.uint32(24).bool(message.paid);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): ArticleAddedEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseArticleAddedEvent } as ArticleAddedEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.publisher = reader.string();
          break;
        case 2:
          message.article_id = longToNumber(reader.uint64() as Long);
          break;
        case 3:
          message.paid = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ArticleAddedEvent {
    const message = { ...baseArticleAddedEvent } as ArticleAddedEvent;
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = String(object.publisher);
    } else {
      message.publisher = "";
    }
    if (object.article_id !== undefined && object.article_id !== null) {
      message.article_id = Number(object.article_id);
    } else {
      message.article_id = 0;
    }
    if (object.paid !== undefined && object.paid !== null) {
      message.paid = Boolean(object.paid);
    } else {
      message.paid = false;
    }
    return message;
  },

  toJSON(message: ArticleAddedEvent): unknown {
    const obj: any = {};
    message.publisher !== undefined && (obj.publisher = message.publisher);
    message.article_id !== undefined && (obj.article_id = message.article_id);
    message.paid !== undefined && (obj.paid = message.paid);
    return obj;
  },

  fromPartial(object: DeepPartial<ArticleAddedEvent>): ArticleAddedEvent {
    const message = { ...baseArticleAddedEvent } as ArticleAddedEvent;
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = object.publisher;
    } else {
      message.publisher = "";
    }
    if (object.article_id !== undefined && object.article_id !== null) {
      message.article_id = object.article_id;
    } else {
      message.article_id = 0;
    }
    if (object.paid !== undefined && object.paid !== null) {
      message.paid = object.paid;
    } else {
      message.paid = false;
    }
    return message;
  },
};

const basePublisherAddedEvent: object = {};

export const PublisherAddedEvent = {
  encode(
    message: PublisherAddedEvent,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.publisher !== undefined) {
      Publisher.encode(message.publisher, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): PublisherAddedEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePublisherAddedEvent } as PublisherAddedEvent;
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

  fromJSON(object: any): PublisherAddedEvent {
    const message = { ...basePublisherAddedEvent } as PublisherAddedEvent;
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = Publisher.fromJSON(object.publisher);
    } else {
      message.publisher = undefined;
    }
    return message;
  },

  toJSON(message: PublisherAddedEvent): unknown {
    const obj: any = {};
    message.publisher !== undefined &&
      (obj.publisher = message.publisher
        ? Publisher.toJSON(message.publisher)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<PublisherAddedEvent>): PublisherAddedEvent {
    const message = { ...basePublisherAddedEvent } as PublisherAddedEvent;
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = Publisher.fromPartial(object.publisher);
    } else {
      message.publisher = undefined;
    }
    return message;
  },
};

const basePublisherUpdatedEvent: object = {};

export const PublisherUpdatedEvent = {
  encode(
    message: PublisherUpdatedEvent,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.publisher !== undefined) {
      Publisher.encode(message.publisher, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): PublisherUpdatedEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePublisherUpdatedEvent } as PublisherUpdatedEvent;
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

  fromJSON(object: any): PublisherUpdatedEvent {
    const message = { ...basePublisherUpdatedEvent } as PublisherUpdatedEvent;
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = Publisher.fromJSON(object.publisher);
    } else {
      message.publisher = undefined;
    }
    return message;
  },

  toJSON(message: PublisherUpdatedEvent): unknown {
    const obj: any = {};
    message.publisher !== undefined &&
      (obj.publisher = message.publisher
        ? Publisher.toJSON(message.publisher)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<PublisherUpdatedEvent>
  ): PublisherUpdatedEvent {
    const message = { ...basePublisherUpdatedEvent } as PublisherUpdatedEvent;
    if (object.publisher !== undefined && object.publisher !== null) {
      message.publisher = Publisher.fromPartial(object.publisher);
    } else {
      message.publisher = undefined;
    }
    return message;
  },
};

const baseAcceptedDomainAddedEvent: object = {};

export const AcceptedDomainAddedEvent = {
  encode(
    message: AcceptedDomainAddedEvent,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.accepted_domain !== undefined) {
      AcceptedDomain.encode(
        message.accepted_domain,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): AcceptedDomainAddedEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseAcceptedDomainAddedEvent,
    } as AcceptedDomainAddedEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.accepted_domain = AcceptedDomain.decode(
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

  fromJSON(object: any): AcceptedDomainAddedEvent {
    const message = {
      ...baseAcceptedDomainAddedEvent,
    } as AcceptedDomainAddedEvent;
    if (
      object.accepted_domain !== undefined &&
      object.accepted_domain !== null
    ) {
      message.accepted_domain = AcceptedDomain.fromJSON(object.accepted_domain);
    } else {
      message.accepted_domain = undefined;
    }
    return message;
  },

  toJSON(message: AcceptedDomainAddedEvent): unknown {
    const obj: any = {};
    message.accepted_domain !== undefined &&
      (obj.accepted_domain = message.accepted_domain
        ? AcceptedDomain.toJSON(message.accepted_domain)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<AcceptedDomainAddedEvent>
  ): AcceptedDomainAddedEvent {
    const message = {
      ...baseAcceptedDomainAddedEvent,
    } as AcceptedDomainAddedEvent;
    if (
      object.accepted_domain !== undefined &&
      object.accepted_domain !== null
    ) {
      message.accepted_domain = AcceptedDomain.fromPartial(
        object.accepted_domain
      );
    } else {
      message.accepted_domain = undefined;
    }
    return message;
  },
};

const baseAcceptedDomainUpdatedEvent: object = {};

export const AcceptedDomainUpdatedEvent = {
  encode(
    message: AcceptedDomainUpdatedEvent,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.accepted_domain !== undefined) {
      AcceptedDomain.encode(
        message.accepted_domain,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): AcceptedDomainUpdatedEvent {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseAcceptedDomainUpdatedEvent,
    } as AcceptedDomainUpdatedEvent;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.accepted_domain = AcceptedDomain.decode(
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

  fromJSON(object: any): AcceptedDomainUpdatedEvent {
    const message = {
      ...baseAcceptedDomainUpdatedEvent,
    } as AcceptedDomainUpdatedEvent;
    if (
      object.accepted_domain !== undefined &&
      object.accepted_domain !== null
    ) {
      message.accepted_domain = AcceptedDomain.fromJSON(object.accepted_domain);
    } else {
      message.accepted_domain = undefined;
    }
    return message;
  },

  toJSON(message: AcceptedDomainUpdatedEvent): unknown {
    const obj: any = {};
    message.accepted_domain !== undefined &&
      (obj.accepted_domain = message.accepted_domain
        ? AcceptedDomain.toJSON(message.accepted_domain)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<AcceptedDomainUpdatedEvent>
  ): AcceptedDomainUpdatedEvent {
    const message = {
      ...baseAcceptedDomainUpdatedEvent,
    } as AcceptedDomainUpdatedEvent;
    if (
      object.accepted_domain !== undefined &&
      object.accepted_domain !== null
    ) {
      message.accepted_domain = AcceptedDomain.fromPartial(
        object.accepted_domain
      );
    } else {
      message.accepted_domain = undefined;
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
