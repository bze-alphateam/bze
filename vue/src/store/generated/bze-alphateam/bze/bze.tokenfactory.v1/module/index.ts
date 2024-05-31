// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.

import { StdFee } from "@cosmjs/launchpad";
import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry, OfflineSigner, EncodeObject, DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgCreateDenom } from "./types/tokenfactory/tx";
import { MsgMint } from "./types/tokenfactory/tx";
import { MsgBurn } from "./types/tokenfactory/tx";
import { MsgSetDenomMetadata } from "./types/tokenfactory/tx";
import { MsgChangeAdmin } from "./types/tokenfactory/tx";


const types = [
  ["/bze.tokenfactory.v1.MsgCreateDenom", MsgCreateDenom],
  ["/bze.tokenfactory.v1.MsgMint", MsgMint],
  ["/bze.tokenfactory.v1.MsgBurn", MsgBurn],
  ["/bze.tokenfactory.v1.MsgSetDenomMetadata", MsgSetDenomMetadata],
  ["/bze.tokenfactory.v1.MsgChangeAdmin", MsgChangeAdmin],
  
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
    msgCreateDenom: (data: MsgCreateDenom): EncodeObject => ({ typeUrl: "/bze.tokenfactory.v1.MsgCreateDenom", value: MsgCreateDenom.fromPartial( data ) }),
    msgMint: (data: MsgMint): EncodeObject => ({ typeUrl: "/bze.tokenfactory.v1.MsgMint", value: MsgMint.fromPartial( data ) }),
    msgBurn: (data: MsgBurn): EncodeObject => ({ typeUrl: "/bze.tokenfactory.v1.MsgBurn", value: MsgBurn.fromPartial( data ) }),
    msgSetDenomMetadata: (data: MsgSetDenomMetadata): EncodeObject => ({ typeUrl: "/bze.tokenfactory.v1.MsgSetDenomMetadata", value: MsgSetDenomMetadata.fromPartial( data ) }),
    msgChangeAdmin: (data: MsgChangeAdmin): EncodeObject => ({ typeUrl: "/bze.tokenfactory.v1.MsgChangeAdmin", value: MsgChangeAdmin.fromPartial( data ) }),
    
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
