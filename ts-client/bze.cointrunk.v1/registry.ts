import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgPayPublisherRespect } from "./types/cointrunk/tx";
import { MsgAddArticle } from "./types/cointrunk/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/bze.cointrunk.v1.MsgPayPublisherRespect", MsgPayPublisherRespect],
    ["/bze.cointrunk.v1.MsgAddArticle", MsgAddArticle],
    
];

export { msgTypes }