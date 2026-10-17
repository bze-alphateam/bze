package keeper

import (
	"github.com/bze-alphateam/bze/x/dao/types"
)

// isLiveElectorate reports whether the DAO has an electorate able to vote (D11).
// A static list is live when it is non-empty.
// TODO(stake DAOs): a stake electorate is live when at least one of its sources
// still exists; extend this when stake DAOs are added.
func isLiveElectorate(dao types.Dao) bool {
	switch dao.Type {
	case types.DaoType_DAO_TYPE_STATIC_LIST:
		return len(dao.Members) > 0
	default:
		return false
	}
}
