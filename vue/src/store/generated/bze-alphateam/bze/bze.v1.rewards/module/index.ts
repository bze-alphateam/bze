// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.

import { StdFee } from "@cosmjs/launchpad";
import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry, OfflineSigner, EncodeObject, DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgExitStaking } from "./types/rewards/tx";
import { MsgJoinStaking } from "./types/rewards/tx";
import { MsgClaimStakingRewards } from "./types/rewards/tx";
import { MsgCreateTradingReward } from "./types/rewards/tx";
import { MsgUpdateStakingReward } from "./types/rewards/tx";
import { MsgCreateStakingReward } from "./types/rewards/tx";


const types = [
  ["/bze.v1.rewards.MsgExitStaking", MsgExitStaking],
  ["/bze.v1.rewards.MsgJoinStaking", MsgJoinStaking],
  ["/bze.v1.rewards.MsgClaimStakingRewards", MsgClaimStakingRewards],
  ["/bze.v1.rewards.MsgCreateTradingReward", MsgCreateTradingReward],
  ["/bze.v1.rewards.MsgUpdateStakingReward", MsgUpdateStakingReward],
  ["/bze.v1.rewards.MsgCreateStakingReward", MsgCreateStakingReward],
  
];
export const MissingWalletError = new Error("wallet is required");

export const registry = new Registry(<any>types);

const defaultFee = {
  amount: [],
  gas: "200000",
};

interface TxClientOptions {
  addr: string
}

interface SignAndBroadcastOptions {
  fee: StdFee,
  memo?: string
}

const txClient = async (wallet: OfflineSigner, { addr: addr }: TxClientOptions = { addr: "http://localhost:26657" }) => {
  if (!wallet) throw MissingWalletError;
  let client;
  if (addr) {
    client = await SigningStargateClient.connectWithSigner(addr, wallet, { registry });
  }else{
    client = await SigningStargateClient.offline( wallet, { registry });
  }
  const { address } = (await wallet.getAccounts())[0];

  return {
    signAndBroadcast: (msgs: EncodeObject[], { fee, memo }: SignAndBroadcastOptions = {fee: defaultFee, memo: ""}) => client.signAndBroadcast(address, msgs, fee,memo),
    msgExitStaking: (data: MsgExitStaking): EncodeObject => ({ typeUrl: "/bze.v1.rewards.MsgExitStaking", value: MsgExitStaking.fromPartial( data ) }),
    msgJoinStaking: (data: MsgJoinStaking): EncodeObject => ({ typeUrl: "/bze.v1.rewards.MsgJoinStaking", value: MsgJoinStaking.fromPartial( data ) }),
    msgClaimStakingRewards: (data: MsgClaimStakingRewards): EncodeObject => ({ typeUrl: "/bze.v1.rewards.MsgClaimStakingRewards", value: MsgClaimStakingRewards.fromPartial( data ) }),
    msgCreateTradingReward: (data: MsgCreateTradingReward): EncodeObject => ({ typeUrl: "/bze.v1.rewards.MsgCreateTradingReward", value: MsgCreateTradingReward.fromPartial( data ) }),
    msgUpdateStakingReward: (data: MsgUpdateStakingReward): EncodeObject => ({ typeUrl: "/bze.v1.rewards.MsgUpdateStakingReward", value: MsgUpdateStakingReward.fromPartial( data ) }),
    msgCreateStakingReward: (data: MsgCreateStakingReward): EncodeObject => ({ typeUrl: "/bze.v1.rewards.MsgCreateStakingReward", value: MsgCreateStakingReward.fromPartial( data ) }),
    
  };
};

interface QueryClientOptions {
  addr: string
}

const queryClient = async ({ addr: addr }: QueryClientOptions = { addr: "http://localhost:1317" }) => {
  return new Api({ baseUrl: addr });
};

export {
  txClient,
  queryClient,
};
