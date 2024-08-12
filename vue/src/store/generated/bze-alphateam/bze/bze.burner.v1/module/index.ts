// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.

import { StdFee } from "@cosmjs/launchpad";
import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry, OfflineSigner, EncodeObject, DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgStartRaffle } from "./types/burner/tx";
import { MsgFundBurner } from "./types/burner/tx";
import { MsgJoinRaffle } from "./types/burner/tx";


const types = [
  ["/bze.burner.v1.MsgStartRaffle", MsgStartRaffle],
  ["/bze.burner.v1.MsgFundBurner", MsgFundBurner],
  ["/bze.burner.v1.MsgJoinRaffle", MsgJoinRaffle],
  
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
    msgStartRaffle: (data: MsgStartRaffle): EncodeObject => ({ typeUrl: "/bze.burner.v1.MsgStartRaffle", value: MsgStartRaffle.fromPartial( data ) }),
    msgFundBurner: (data: MsgFundBurner): EncodeObject => ({ typeUrl: "/bze.burner.v1.MsgFundBurner", value: MsgFundBurner.fromPartial( data ) }),
    msgJoinRaffle: (data: MsgJoinRaffle): EncodeObject => ({ typeUrl: "/bze.burner.v1.MsgJoinRaffle", value: MsgJoinRaffle.fromPartial( data ) }),
    
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
