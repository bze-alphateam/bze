// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.

import BzeBurnerV1 from './bze.burner.v1'
import BzeCointrunkV1 from './bze.cointrunk.v1'
import BzeEpochsV1 from './bze.epochs.v1'
import BzeV1Rewards from './bze.v1.rewards'
import BzeScavenge from './bze.scavenge'
import BzeTokenfactoryV1 from './bze.tokenfactory.v1'
import BzeTradebinV1 from './bze.tradebin.v1'


export default { 
  BzeBurnerV1: load(BzeBurnerV1, 'bze.burner.v1'),
  BzeCointrunkV1: load(BzeCointrunkV1, 'bze.cointrunk.v1'),
  BzeEpochsV1: load(BzeEpochsV1, 'bze.epochs.v1'),
  BzeV1Rewards: load(BzeV1Rewards, 'bze.v1.rewards'),
  BzeScavenge: load(BzeScavenge, 'bze.scavenge'),
  BzeTokenfactoryV1: load(BzeTokenfactoryV1, 'bze.tokenfactory.v1'),
  BzeTradebinV1: load(BzeTradebinV1, 'bze.tradebin.v1'),
  
}


function load(mod, fullns) {
    return function init(store) {        
        if (store.hasModule([fullns])) {
            throw new Error('Duplicate module name detected: '+ fullns)
        }else{
            store.registerModule([fullns], mod)
            store.subscribe((mutation) => {
                if (mutation.type == 'common/env/INITIALIZE_WS_COMPLETE') {
                    store.dispatch(fullns+ '/init', null, {
                        root: true
                    })
                }
            })
        }
    }
}