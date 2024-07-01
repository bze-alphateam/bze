import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgSubmitScavenge } from "./types/scavenge/tx";
import { MsgCommitSolution } from "./types/scavenge/tx";
import { MsgRevealSolution } from "./types/scavenge/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/bze.scavenge.MsgSubmitScavenge", MsgSubmitScavenge],
    ["/bze.scavenge.MsgCommitSolution", MsgCommitSolution],
    ["/bze.scavenge.MsgRevealSolution", MsgRevealSolution],
    
];

export { msgTypes }