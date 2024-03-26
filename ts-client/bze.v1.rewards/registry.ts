import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgCreateStakingReward } from "./types/rewards/tx";
import { MsgCreateTradingReward } from "./types/rewards/tx";
import { MsgJoinStaking } from "./types/rewards/tx";
import { MsgUpdateStakingReward } from "./types/rewards/tx";
import { MsgClaimStakingRewards } from "./types/rewards/tx";
import { MsgExitStaking } from "./types/rewards/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/bze.v1.rewards.MsgCreateStakingReward", MsgCreateStakingReward],
    ["/bze.v1.rewards.MsgCreateTradingReward", MsgCreateTradingReward],
    ["/bze.v1.rewards.MsgJoinStaking", MsgJoinStaking],
    ["/bze.v1.rewards.MsgUpdateStakingReward", MsgUpdateStakingReward],
    ["/bze.v1.rewards.MsgClaimStakingRewards", MsgClaimStakingRewards],
    ["/bze.v1.rewards.MsgExitStaking", MsgExitStaking],
    
];

export { msgTypes }