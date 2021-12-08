import { Reader, Writer } from "protobufjs/minimal";
import { Scavenge } from "../scavenge/scavenge";
import { PageRequest, PageResponse } from "../cosmos/base/query/v1beta1/pagination";
import { Commit } from "../scavenge/commit";
export declare const protobufPackage = "bzedgev5.scavenge";
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
export declare const QueryGetScavengeRequest: {
    encode(message: QueryGetScavengeRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetScavengeRequest;
    fromJSON(object: any): QueryGetScavengeRequest;
    toJSON(message: QueryGetScavengeRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetScavengeRequest>): QueryGetScavengeRequest;
};
export declare const QueryGetScavengeResponse: {
    encode(message: QueryGetScavengeResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetScavengeResponse;
    fromJSON(object: any): QueryGetScavengeResponse;
    toJSON(message: QueryGetScavengeResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetScavengeResponse>): QueryGetScavengeResponse;
};
export declare const QueryAllScavengeRequest: {
    encode(message: QueryAllScavengeRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllScavengeRequest;
    fromJSON(object: any): QueryAllScavengeRequest;
    toJSON(message: QueryAllScavengeRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllScavengeRequest>): QueryAllScavengeRequest;
};
export declare const QueryAllScavengeResponse: {
    encode(message: QueryAllScavengeResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllScavengeResponse;
    fromJSON(object: any): QueryAllScavengeResponse;
    toJSON(message: QueryAllScavengeResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllScavengeResponse>): QueryAllScavengeResponse;
};
export declare const QueryGetCommitRequest: {
    encode(message: QueryGetCommitRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetCommitRequest;
    fromJSON(object: any): QueryGetCommitRequest;
    toJSON(message: QueryGetCommitRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetCommitRequest>): QueryGetCommitRequest;
};
export declare const QueryGetCommitResponse: {
    encode(message: QueryGetCommitResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetCommitResponse;
    fromJSON(object: any): QueryGetCommitResponse;
    toJSON(message: QueryGetCommitResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetCommitResponse>): QueryGetCommitResponse;
};
export declare const QueryAllCommitRequest: {
    encode(message: QueryAllCommitRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllCommitRequest;
    fromJSON(object: any): QueryAllCommitRequest;
    toJSON(message: QueryAllCommitRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllCommitRequest>): QueryAllCommitRequest;
};
export declare const QueryAllCommitResponse: {
    encode(message: QueryAllCommitResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllCommitResponse;
    fromJSON(object: any): QueryAllCommitResponse;
    toJSON(message: QueryAllCommitResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllCommitResponse>): QueryAllCommitResponse;
};
/** Query defines the gRPC querier service. */
export interface Query {
    /** Queries a scavenge by index. */
    Scavenge(request: QueryGetScavengeRequest): Promise<QueryGetScavengeResponse>;
    /** Queries a list of scavenge items. */
    ScavengeAll(request: QueryAllScavengeRequest): Promise<QueryAllScavengeResponse>;
    /** Queries a commit by index. */
    Commit(request: QueryGetCommitRequest): Promise<QueryGetCommitResponse>;
    /** Queries a list of commit items. */
    CommitAll(request: QueryAllCommitRequest): Promise<QueryAllCommitResponse>;
}
export declare class QueryClientImpl implements Query {
    private readonly rpc;
    constructor(rpc: Rpc);
    Scavenge(request: QueryGetScavengeRequest): Promise<QueryGetScavengeResponse>;
    ScavengeAll(request: QueryAllScavengeRequest): Promise<QueryAllScavengeResponse>;
    Commit(request: QueryGetCommitRequest): Promise<QueryGetCommitResponse>;
    CommitAll(request: QueryAllCommitRequest): Promise<QueryAllCommitResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
