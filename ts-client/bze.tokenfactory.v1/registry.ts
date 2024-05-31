import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgChangeAdmin } from "./types/tokenfactory/tx";
import { MsgMint } from "./types/tokenfactory/tx";
import { MsgSetDenomMetadata } from "./types/tokenfactory/tx";
import { MsgBurn } from "./types/tokenfactory/tx";
import { MsgCreateDenom } from "./types/tokenfactory/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/bze.tokenfactory.v1.MsgChangeAdmin", MsgChangeAdmin],
    ["/bze.tokenfactory.v1.MsgMint", MsgMint],
    ["/bze.tokenfactory.v1.MsgSetDenomMetadata", MsgSetDenomMetadata],
    ["/bze.tokenfactory.v1.MsgBurn", MsgBurn],
    ["/bze.tokenfactory.v1.MsgCreateDenom", MsgCreateDenom],
    
];

export { msgTypes }