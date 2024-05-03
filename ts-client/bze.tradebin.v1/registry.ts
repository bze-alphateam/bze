import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgCreateMarket } from "./types/tradebin/tx";
import { MsgCancelOrder } from "./types/tradebin/tx";
import { MsgCreateOrder } from "./types/tradebin/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/bze.tradebin.v1.MsgCreateMarket", MsgCreateMarket],
    ["/bze.tradebin.v1.MsgCancelOrder", MsgCancelOrder],
    ["/bze.tradebin.v1.MsgCreateOrder", MsgCreateOrder],
    
];

export { msgTypes }