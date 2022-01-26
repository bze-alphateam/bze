/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Scavenge } from "../scavenge/scavenge";
import { PageRequest, PageResponse, } from "../cosmos/base/query/v1beta1/pagination";
import { Commit } from "../scavenge/commit";
export const protobufPackage = "bze.scavenge";
const baseQueryGetScavengeRequest = { index: "" };
export const QueryGetScavengeRequest = {
    encode(message, writer = Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetScavengeRequest,
        };
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
    fromJSON(object) {
        const message = {
            ...baseQueryGetScavengeRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetScavengeRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        return message;
    },
};
const baseQueryGetScavengeResponse = {};
export const QueryGetScavengeResponse = {
    encode(message, writer = Writer.create()) {
        if (message.scavenge !== undefined) {
            Scavenge.encode(message.scavenge, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetScavengeResponse,
        };
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
    fromJSON(object) {
        const message = {
            ...baseQueryGetScavengeResponse,
        };
        if (object.scavenge !== undefined && object.scavenge !== null) {
            message.scavenge = Scavenge.fromJSON(object.scavenge);
        }
        else {
            message.scavenge = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.scavenge !== undefined &&
            (obj.scavenge = message.scavenge
                ? Scavenge.toJSON(message.scavenge)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetScavengeResponse,
        };
        if (object.scavenge !== undefined && object.scavenge !== null) {
            message.scavenge = Scavenge.fromPartial(object.scavenge);
        }
        else {
            message.scavenge = undefined;
        }
        return message;
    },
};
const baseQueryAllScavengeRequest = {};
export const QueryAllScavengeRequest = {
    encode(message, writer = Writer.create()) {
        if (message.pagination !== undefined) {
            PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllScavengeRequest,
        };
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
    fromJSON(object) {
        const message = {
            ...baseQueryAllScavengeRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllScavengeRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryAllScavengeResponse = {};
export const QueryAllScavengeResponse = {
    encode(message, writer = Writer.create()) {
        for (const v of message.scavenge) {
            Scavenge.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllScavengeResponse,
        };
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
    fromJSON(object) {
        const message = {
            ...baseQueryAllScavengeResponse,
        };
        message.scavenge = [];
        if (object.scavenge !== undefined && object.scavenge !== null) {
            for (const e of object.scavenge) {
                message.scavenge.push(Scavenge.fromJSON(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.scavenge) {
            obj.scavenge = message.scavenge.map((e) => e ? Scavenge.toJSON(e) : undefined);
        }
        else {
            obj.scavenge = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllScavengeResponse,
        };
        message.scavenge = [];
        if (object.scavenge !== undefined && object.scavenge !== null) {
            for (const e of object.scavenge) {
                message.scavenge.push(Scavenge.fromPartial(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryGetCommitRequest = { index: "" };
export const QueryGetCommitRequest = {
    encode(message, writer = Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseQueryGetCommitRequest };
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
    fromJSON(object) {
        const message = { ...baseQueryGetCommitRequest };
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseQueryGetCommitRequest };
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        return message;
    },
};
const baseQueryGetCommitResponse = {};
export const QueryGetCommitResponse = {
    encode(message, writer = Writer.create()) {
        if (message.commit !== undefined) {
            Commit.encode(message.commit, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseQueryGetCommitResponse };
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
    fromJSON(object) {
        const message = { ...baseQueryGetCommitResponse };
        if (object.commit !== undefined && object.commit !== null) {
            message.commit = Commit.fromJSON(object.commit);
        }
        else {
            message.commit = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.commit !== undefined &&
            (obj.commit = message.commit ? Commit.toJSON(message.commit) : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseQueryGetCommitResponse };
        if (object.commit !== undefined && object.commit !== null) {
            message.commit = Commit.fromPartial(object.commit);
        }
        else {
            message.commit = undefined;
        }
        return message;
    },
};
const baseQueryAllCommitRequest = {};
export const QueryAllCommitRequest = {
    encode(message, writer = Writer.create()) {
        if (message.pagination !== undefined) {
            PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseQueryAllCommitRequest };
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
    fromJSON(object) {
        const message = { ...baseQueryAllCommitRequest };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseQueryAllCommitRequest };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryAllCommitResponse = {};
export const QueryAllCommitResponse = {
    encode(message, writer = Writer.create()) {
        for (const v of message.commit) {
            Commit.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseQueryAllCommitResponse };
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
    fromJSON(object) {
        const message = { ...baseQueryAllCommitResponse };
        message.commit = [];
        if (object.commit !== undefined && object.commit !== null) {
            for (const e of object.commit) {
                message.commit.push(Commit.fromJSON(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.commit) {
            obj.commit = message.commit.map((e) => e ? Commit.toJSON(e) : undefined);
        }
        else {
            obj.commit = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseQueryAllCommitResponse };
        message.commit = [];
        if (object.commit !== undefined && object.commit !== null) {
            for (const e of object.commit) {
                message.commit.push(Commit.fromPartial(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
export class QueryClientImpl {
    constructor(rpc) {
        this.rpc = rpc;
    }
    Scavenge(request) {
        const data = QueryGetScavengeRequest.encode(request).finish();
        const promise = this.rpc.request("bze.scavenge.Query", "Scavenge", data);
        return promise.then((data) => QueryGetScavengeResponse.decode(new Reader(data)));
    }
    ScavengeAll(request) {
        const data = QueryAllScavengeRequest.encode(request).finish();
        const promise = this.rpc.request("bze.scavenge.Query", "ScavengeAll", data);
        return promise.then((data) => QueryAllScavengeResponse.decode(new Reader(data)));
    }
    Commit(request) {
        const data = QueryGetCommitRequest.encode(request).finish();
        const promise = this.rpc.request("bze.scavenge.Query", "Commit", data);
        return promise.then((data) => QueryGetCommitResponse.decode(new Reader(data)));
    }
    CommitAll(request) {
        const data = QueryAllCommitRequest.encode(request).finish();
        const promise = this.rpc.request("bze.scavenge.Query", "CommitAll", data);
        return promise.then((data) => QueryAllCommitResponse.decode(new Reader(data)));
    }
}
