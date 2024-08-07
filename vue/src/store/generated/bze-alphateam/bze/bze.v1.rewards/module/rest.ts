/* eslint-disable */
/* tslint:disable */
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export interface ProtobufAny {
  "@type"?: string;
}

export interface RewardsMarketIdTradingRewardId {
  reward_id?: string;
  market_id?: string;
}

export interface RewardsMsgClaimStakingRewardsResponse {
  amount?: string;
}

export interface RewardsMsgCreateStakingRewardResponse {
  reward_id?: string;
}

export interface RewardsMsgCreateTradingRewardResponse {
  reward_id?: string;
}

export type RewardsMsgDistributeStakingRewardsResponse = object;

export type RewardsMsgExitStakingResponse = object;

export type RewardsMsgJoinStakingResponse = object;

export type RewardsMsgUpdateStakingRewardResponse = object;

export interface RewardsPendingUnlockParticipant {
  index?: string;
  address?: string;
  amount?: string;
  denom?: string;
}

export interface RewardsQueryAllPendingUnlockParticipantResponse {
  list?: RewardsPendingUnlockParticipant[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface RewardsQueryAllStakingRewardParticipantResponse {
  list?: V1RewardsStakingRewardParticipant[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface RewardsQueryAllStakingRewardResponse {
  list?: V1RewardsStakingReward[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface RewardsQueryAllTradingRewardResponse {
  list?: V1RewardsTradingReward[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface RewardsQueryGetMarketIdTradingRewardIdHandlerResponse {
  market_id_reward_id?: RewardsMarketIdTradingRewardId;
}

export interface RewardsQueryGetStakingRewardParticipantResponse {
  list?: V1RewardsStakingRewardParticipant[];

  /**
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;
}

export interface RewardsQueryGetStakingRewardResponse {
  staking_reward?: V1RewardsStakingReward;
}

export interface RewardsQueryGetTradingRewardLeaderboardResponse {
  leaderboard?: RewardsTradingRewardLeaderboard;
}

export interface RewardsQueryGetTradingRewardResponse {
  trading_reward?: V1RewardsTradingReward;
}

/**
 * QueryParamsResponse is response type for the Query/Params RPC method.
 */
export interface RewardsQueryParamsResponse {
  /** params holds all the parameters of this module. */
  params?: V1RewardsParams;
}

export interface RewardsTradingRewardLeaderboard {
  reward_id?: string;
  list?: RewardsTradingRewardLeaderboardEntry[];
}

export interface RewardsTradingRewardLeaderboardEntry {
  amount?: string;
  address?: string;

  /** @format int64 */
  created_at?: string;
}

export interface RpcStatus {
  /** @format int32 */
  code?: number;
  message?: string;
  details?: ProtobufAny[];
}

/**
* message SomeRequest {
         Foo some_parameter = 1;
         PageRequest pagination = 2;
 }
*/
export interface V1Beta1PageRequest {
  /**
   * key is a value returned in PageResponse.next_key to begin
   * querying the next page most efficiently. Only one of offset or key
   * should be set.
   * @format byte
   */
  key?: string;

  /**
   * offset is a numeric offset that can be used when key is unavailable.
   * It is less efficient than using key. Only one of offset or key should
   * be set.
   * @format uint64
   */
  offset?: string;

  /**
   * limit is the total number of results to be returned in the result page.
   * If left empty it will default to a value to be set by each app.
   * @format uint64
   */
  limit?: string;

  /**
   * count_total is set to true  to indicate that the result set should include
   * a count of the total number of items available for pagination in UIs.
   * count_total is only respected when offset is used. It is ignored when key
   * is set.
   */
  count_total?: boolean;

  /**
   * reverse is set to true if results are to be returned in the descending order.
   *
   * Since: cosmos-sdk 0.43
   */
  reverse?: boolean;
}

/**
* PageResponse is to be embedded in gRPC response messages where the
corresponding request message has used PageRequest.

 message SomeResponse {
         repeated Bar results = 1;
         PageResponse page = 2;
 }
*/
export interface V1Beta1PageResponse {
  /** @format byte */
  next_key?: string;

  /** @format uint64 */
  total?: string;
}

/**
 * Params defines the parameters for the module.
 */
export interface V1RewardsParams {
  createStakingRewardFee?: string;
  createTradingRewardFee?: string;
}

export interface V1RewardsStakingReward {
  reward_id?: string;
  prize_amount?: string;
  prize_denom?: string;
  staking_denom?: string;

  /** @format int64 */
  duration?: number;

  /** @format int64 */
  payouts?: number;

  /** @format uint64 */
  min_stake?: string;

  /** @format int64 */
  lock?: number;
  staked_amount?: string;
  distributed_stake?: string;
}

export interface V1RewardsStakingRewardParticipant {
  address?: string;
  reward_id?: string;
  amount?: string;
  joined_at?: string;
}

export interface V1RewardsTradingReward {
  reward_id?: string;
  prize_amount?: string;
  prize_denom?: string;

  /** @format int64 */
  duration?: number;
  market_id?: string;

  /** @format int64 */
  slots?: number;

  /** @format int64 */
  expire_at?: number;
}

export type QueryParamsType = Record<string | number, any>;
export type ResponseFormat = keyof Omit<Body, "body" | "bodyUsed">;

export interface FullRequestParams extends Omit<RequestInit, "body"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: keyof Omit<Body, "body" | "bodyUsed">;
  /** request body */
  body?: unknown;
  /** base url */
  baseUrl?: string;
  /** request cancellation token */
  cancelToken?: CancelToken;
}

export type RequestParams = Omit<FullRequestParams, "body" | "method" | "query" | "path">;

export interface ApiConfig<SecurityDataType = unknown> {
  baseUrl?: string;
  baseApiParams?: Omit<RequestParams, "baseUrl" | "cancelToken" | "signal">;
  securityWorker?: (securityData: SecurityDataType) => RequestParams | void;
}

export interface HttpResponse<D extends unknown, E extends unknown = unknown> extends Response {
  data: D;
  error: E;
}

type CancelToken = Symbol | string | number;

export enum ContentType {
  Json = "application/json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
}

export class HttpClient<SecurityDataType = unknown> {
  public baseUrl: string = "";
  private securityData: SecurityDataType = null as any;
  private securityWorker: null | ApiConfig<SecurityDataType>["securityWorker"] = null;
  private abortControllers = new Map<CancelToken, AbortController>();

  private baseApiParams: RequestParams = {
    credentials: "same-origin",
    headers: {},
    redirect: "follow",
    referrerPolicy: "no-referrer",
  };

  constructor(apiConfig: ApiConfig<SecurityDataType> = {}) {
    Object.assign(this, apiConfig);
  }

  public setSecurityData = (data: SecurityDataType) => {
    this.securityData = data;
  };

  private addQueryParam(query: QueryParamsType, key: string) {
    const value = query[key];

    return (
      encodeURIComponent(key) +
      "=" +
      encodeURIComponent(Array.isArray(value) ? value.join(",") : typeof value === "number" ? value : `${value}`)
    );
  }

  protected toQueryString(rawQuery?: QueryParamsType): string {
    const query = rawQuery || {};
    const keys = Object.keys(query).filter((key) => "undefined" !== typeof query[key]);
    return keys
      .map((key) =>
        typeof query[key] === "object" && !Array.isArray(query[key])
          ? this.toQueryString(query[key] as QueryParamsType)
          : this.addQueryParam(query, key),
      )
      .join("&");
  }

  protected addQueryParams(rawQuery?: QueryParamsType): string {
    const queryString = this.toQueryString(rawQuery);
    return queryString ? `?${queryString}` : "";
  }

  private contentFormatters: Record<ContentType, (input: any) => any> = {
    [ContentType.Json]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string") ? JSON.stringify(input) : input,
    [ContentType.FormData]: (input: any) =>
      Object.keys(input || {}).reduce((data, key) => {
        data.append(key, input[key]);
        return data;
      }, new FormData()),
    [ContentType.UrlEncoded]: (input: any) => this.toQueryString(input),
  };

  private mergeRequestParams(params1: RequestParams, params2?: RequestParams): RequestParams {
    return {
      ...this.baseApiParams,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...(this.baseApiParams.headers || {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  private createAbortSignal = (cancelToken: CancelToken): AbortSignal | undefined => {
    if (this.abortControllers.has(cancelToken)) {
      const abortController = this.abortControllers.get(cancelToken);
      if (abortController) {
        return abortController.signal;
      }
      return void 0;
    }

    const abortController = new AbortController();
    this.abortControllers.set(cancelToken, abortController);
    return abortController.signal;
  };

  public abortRequest = (cancelToken: CancelToken) => {
    const abortController = this.abortControllers.get(cancelToken);

    if (abortController) {
      abortController.abort();
      this.abortControllers.delete(cancelToken);
    }
  };

  public request = <T = any, E = any>({
    body,
    secure,
    path,
    type,
    query,
    format = "json",
    baseUrl,
    cancelToken,
    ...params
  }: FullRequestParams): Promise<HttpResponse<T, E>> => {
    const secureParams = (secure && this.securityWorker && this.securityWorker(this.securityData)) || {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const queryString = query && this.toQueryString(query);
    const payloadFormatter = this.contentFormatters[type || ContentType.Json];

    return fetch(`${baseUrl || this.baseUrl || ""}${path}${queryString ? `?${queryString}` : ""}`, {
      ...requestParams,
      headers: {
        ...(type && type !== ContentType.FormData ? { "Content-Type": type } : {}),
        ...(requestParams.headers || {}),
      },
      signal: cancelToken ? this.createAbortSignal(cancelToken) : void 0,
      body: typeof body === "undefined" || body === null ? null : payloadFormatter(body),
    }).then(async (response) => {
      const r = response as HttpResponse<T, E>;
      r.data = (null as unknown) as T;
      r.error = (null as unknown) as E;

      const data = await response[format]()
        .then((data) => {
          if (r.ok) {
            r.data = data;
          } else {
            r.error = data;
          }
          return r;
        })
        .catch((e) => {
          r.error = e;
          return r;
        });

      if (cancelToken) {
        this.abortControllers.delete(cancelToken);
      }

      if (!response.ok) throw data;
      return data;
    });
  };
}

/**
 * @title rewards/events.proto
 * @version version not set
 */
export class Api<SecurityDataType extends unknown> extends HttpClient<SecurityDataType> {
  /**
   * No description
   *
   * @tags Query
   * @name QueryAllPendingUnlockParticipant
   * @summary Queries a list of AllPendingUnlockParticipant items.
   * @request GET:/bze/rewards/v1/all_pending_unlock_participant
   */
  queryAllPendingUnlockParticipant = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<RewardsQueryAllPendingUnlockParticipantResponse, RpcStatus>({
      path: `/bze/rewards/v1/all_pending_unlock_participant`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryGetMarketIdTradingRewardIdHandler
   * @summary Queries a list of GetMarketIdTradingRewardIdHandler items.
   * @request GET:/bze/rewards/v1/market_id_trading_reward_id
   */
  queryGetMarketIdTradingRewardIdHandler = (query?: { market_id?: string }, params: RequestParams = {}) =>
    this.request<RewardsQueryGetMarketIdTradingRewardIdHandlerResponse, RpcStatus>({
      path: `/bze/rewards/v1/market_id_trading_reward_id`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryParams
   * @summary Parameters queries the parameters of the module.
   * @request GET:/bze/rewards/v1/params
   */
  queryParams = (params: RequestParams = {}) =>
    this.request<RewardsQueryParamsResponse, RpcStatus>({
      path: `/bze/rewards/v1/params`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryStakingRewardAll
   * @summary Queries a list of StakingReward items.
   * @request GET:/bze/rewards/v1/staking_reward
   */
  queryStakingRewardAll = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<RewardsQueryAllStakingRewardResponse, RpcStatus>({
      path: `/bze/rewards/v1/staking_reward`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryStakingReward
   * @summary Queries a StakingReward by index.
   * @request GET:/bze/rewards/v1/staking_reward/{reward_id}
   */
  queryStakingReward = (reward_id: string, params: RequestParams = {}) =>
    this.request<RewardsQueryGetStakingRewardResponse, RpcStatus>({
      path: `/bze/rewards/v1/staking_reward/${reward_id}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryStakingRewardParticipant
   * @summary Queries a StakingRewardParticipant by index.
   * @request GET:/bze/rewards/v1/staking_reward_participant/{address}
   */
  queryStakingRewardParticipant = (
    address: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<RewardsQueryGetStakingRewardParticipantResponse, RpcStatus>({
      path: `/bze/rewards/v1/staking_reward_participant/${address}`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryStakingRewardParticipantAll
   * @summary Queries a list of StakingRewardParticipant items.
   * @request GET:/bze/rewards/v1/staking_reward_participants
   */
  queryStakingRewardParticipantAll = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<RewardsQueryAllStakingRewardParticipantResponse, RpcStatus>({
      path: `/bze/rewards/v1/staking_reward_participants`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryTradingReward
   * @summary Queries a TradingReward by index.
   * @request GET:/bze/rewards/v1/trading_reward/{reward_id}
   */
  queryTradingReward = (reward_id: string, params: RequestParams = {}) =>
    this.request<RewardsQueryGetTradingRewardResponse, RpcStatus>({
      path: `/bze/rewards/v1/trading_reward/${reward_id}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryTradingRewardAll
   * @summary Queries a list of TradingReward items.
   * @request GET:/bze/rewards/v1/trading_reward/{state}
   */
  queryTradingRewardAll = (
    state: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<RewardsQueryAllTradingRewardResponse, RpcStatus>({
      path: `/bze/rewards/v1/trading_reward/${state}`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryGetTradingRewardLeaderboardHandler
   * @summary Queries a list of GetTradingRewardLeaderboard items.
   * @request GET:/bze/rewards/v1/trading_reward_leaderboard/{reward_id}
   */
  queryGetTradingRewardLeaderboardHandler = (reward_id: string, params: RequestParams = {}) =>
    this.request<RewardsQueryGetTradingRewardLeaderboardResponse, RpcStatus>({
      path: `/bze/rewards/v1/trading_reward_leaderboard/${reward_id}`,
      method: "GET",
      format: "json",
      ...params,
    });
}
