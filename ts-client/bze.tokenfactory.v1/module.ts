// Generated by Ignite ignite.com/cli

import { StdFee } from "@cosmjs/launchpad";
import { SigningStargateClient, DeliverTxResponse } from "@cosmjs/stargate";
import { EncodeObject, GeneratedType, OfflineSigner, Registry } from "@cosmjs/proto-signing";
import { msgTypes } from './registry';
import { IgniteClient } from "../client"
import { MissingWalletError } from "../helpers"
import { Api } from "./rest";
import { MsgChangeAdmin } from "./types/tokenfactory/tx";
import { MsgMint } from "./types/tokenfactory/tx";
import { MsgSetDenomMetadata } from "./types/tokenfactory/tx";
import { MsgBurn } from "./types/tokenfactory/tx";
import { MsgCreateDenom } from "./types/tokenfactory/tx";


export { MsgChangeAdmin, MsgMint, MsgSetDenomMetadata, MsgBurn, MsgCreateDenom };

type sendMsgChangeAdminParams = {
  value: MsgChangeAdmin,
  fee?: StdFee,
  memo?: string
};

type sendMsgMintParams = {
  value: MsgMint,
  fee?: StdFee,
  memo?: string
};

type sendMsgSetDenomMetadataParams = {
  value: MsgSetDenomMetadata,
  fee?: StdFee,
  memo?: string
};

type sendMsgBurnParams = {
  value: MsgBurn,
  fee?: StdFee,
  memo?: string
};

type sendMsgCreateDenomParams = {
  value: MsgCreateDenom,
  fee?: StdFee,
  memo?: string
};


type msgChangeAdminParams = {
  value: MsgChangeAdmin,
};

type msgMintParams = {
  value: MsgMint,
};

type msgSetDenomMetadataParams = {
  value: MsgSetDenomMetadata,
};

type msgBurnParams = {
  value: MsgBurn,
};

type msgCreateDenomParams = {
  value: MsgCreateDenom,
};


export const registry = new Registry(msgTypes);

const defaultFee = {
  amount: [],
  gas: "200000",
};

interface TxClientOptions {
  addr: string
	prefix: string
	signer?: OfflineSigner
}

export const txClient = ({ signer, prefix, addr }: TxClientOptions = { addr: "http://localhost:26657", prefix: "cosmos" }) => {

  return {
		
		async sendMsgChangeAdmin({ value, fee, memo }: sendMsgChangeAdminParams): Promise<DeliverTxResponse> {
			if (!signer) {
					throw new Error('TxClient:sendMsgChangeAdmin: Unable to sign Tx. Signer is not present.')
			}
			try {			
				const { address } = (await signer.getAccounts())[0]; 
				const signingClient = await SigningStargateClient.connectWithSigner(addr,signer,{registry, prefix});
				let msg = this.msgChangeAdmin({ value: MsgChangeAdmin.fromPartial(value) })
				return await signingClient.signAndBroadcast(address, [msg], fee ? fee : defaultFee, memo)
			} catch (e: any) {
				throw new Error('TxClient:sendMsgChangeAdmin: Could not broadcast Tx: '+ e.message)
			}
		},
		
		async sendMsgMint({ value, fee, memo }: sendMsgMintParams): Promise<DeliverTxResponse> {
			if (!signer) {
					throw new Error('TxClient:sendMsgMint: Unable to sign Tx. Signer is not present.')
			}
			try {			
				const { address } = (await signer.getAccounts())[0]; 
				const signingClient = await SigningStargateClient.connectWithSigner(addr,signer,{registry, prefix});
				let msg = this.msgMint({ value: MsgMint.fromPartial(value) })
				return await signingClient.signAndBroadcast(address, [msg], fee ? fee : defaultFee, memo)
			} catch (e: any) {
				throw new Error('TxClient:sendMsgMint: Could not broadcast Tx: '+ e.message)
			}
		},
		
		async sendMsgSetDenomMetadata({ value, fee, memo }: sendMsgSetDenomMetadataParams): Promise<DeliverTxResponse> {
			if (!signer) {
					throw new Error('TxClient:sendMsgSetDenomMetadata: Unable to sign Tx. Signer is not present.')
			}
			try {			
				const { address } = (await signer.getAccounts())[0]; 
				const signingClient = await SigningStargateClient.connectWithSigner(addr,signer,{registry, prefix});
				let msg = this.msgSetDenomMetadata({ value: MsgSetDenomMetadata.fromPartial(value) })
				return await signingClient.signAndBroadcast(address, [msg], fee ? fee : defaultFee, memo)
			} catch (e: any) {
				throw new Error('TxClient:sendMsgSetDenomMetadata: Could not broadcast Tx: '+ e.message)
			}
		},
		
		async sendMsgBurn({ value, fee, memo }: sendMsgBurnParams): Promise<DeliverTxResponse> {
			if (!signer) {
					throw new Error('TxClient:sendMsgBurn: Unable to sign Tx. Signer is not present.')
			}
			try {			
				const { address } = (await signer.getAccounts())[0]; 
				const signingClient = await SigningStargateClient.connectWithSigner(addr,signer,{registry, prefix});
				let msg = this.msgBurn({ value: MsgBurn.fromPartial(value) })
				return await signingClient.signAndBroadcast(address, [msg], fee ? fee : defaultFee, memo)
			} catch (e: any) {
				throw new Error('TxClient:sendMsgBurn: Could not broadcast Tx: '+ e.message)
			}
		},
		
		async sendMsgCreateDenom({ value, fee, memo }: sendMsgCreateDenomParams): Promise<DeliverTxResponse> {
			if (!signer) {
					throw new Error('TxClient:sendMsgCreateDenom: Unable to sign Tx. Signer is not present.')
			}
			try {			
				const { address } = (await signer.getAccounts())[0]; 
				const signingClient = await SigningStargateClient.connectWithSigner(addr,signer,{registry, prefix});
				let msg = this.msgCreateDenom({ value: MsgCreateDenom.fromPartial(value) })
				return await signingClient.signAndBroadcast(address, [msg], fee ? fee : defaultFee, memo)
			} catch (e: any) {
				throw new Error('TxClient:sendMsgCreateDenom: Could not broadcast Tx: '+ e.message)
			}
		},
		
		
		msgChangeAdmin({ value }: msgChangeAdminParams): EncodeObject {
			try {
				return { typeUrl: "/bze.tokenfactory.v1.MsgChangeAdmin", value: MsgChangeAdmin.fromPartial( value ) }  
			} catch (e: any) {
				throw new Error('TxClient:MsgChangeAdmin: Could not create message: ' + e.message)
			}
		},
		
		msgMint({ value }: msgMintParams): EncodeObject {
			try {
				return { typeUrl: "/bze.tokenfactory.v1.MsgMint", value: MsgMint.fromPartial( value ) }  
			} catch (e: any) {
				throw new Error('TxClient:MsgMint: Could not create message: ' + e.message)
			}
		},
		
		msgSetDenomMetadata({ value }: msgSetDenomMetadataParams): EncodeObject {
			try {
				return { typeUrl: "/bze.tokenfactory.v1.MsgSetDenomMetadata", value: MsgSetDenomMetadata.fromPartial( value ) }  
			} catch (e: any) {
				throw new Error('TxClient:MsgSetDenomMetadata: Could not create message: ' + e.message)
			}
		},
		
		msgBurn({ value }: msgBurnParams): EncodeObject {
			try {
				return { typeUrl: "/bze.tokenfactory.v1.MsgBurn", value: MsgBurn.fromPartial( value ) }  
			} catch (e: any) {
				throw new Error('TxClient:MsgBurn: Could not create message: ' + e.message)
			}
		},
		
		msgCreateDenom({ value }: msgCreateDenomParams): EncodeObject {
			try {
				return { typeUrl: "/bze.tokenfactory.v1.MsgCreateDenom", value: MsgCreateDenom.fromPartial( value ) }  
			} catch (e: any) {
				throw new Error('TxClient:MsgCreateDenom: Could not create message: ' + e.message)
			}
		},
		
	}
};

interface QueryClientOptions {
  addr: string
}

export const queryClient = ({ addr: addr }: QueryClientOptions = { addr: "http://localhost:1317" }) => {
  return new Api({ baseUrl: addr });
};

class SDKModule {
	public query: ReturnType<typeof queryClient>;
	public tx: ReturnType<typeof txClient>;
	
	public registry: Array<[string, GeneratedType]>;

	constructor(client: IgniteClient) {		
	
		this.query = queryClient({ addr: client.env.apiURL });
		this.tx = txClient({ signer: client.signer, addr: client.env.rpcURL, prefix: client.env.prefix ?? "cosmos" });
	}
};

const Module = (test: IgniteClient) => {
	return {
		module: {
			BzeTokenfactoryV1: new SDKModule(test)
		},
		registry: msgTypes
  }
}
export default Module;