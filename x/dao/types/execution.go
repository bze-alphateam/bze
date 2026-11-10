package types

// MaxDaoExecutionDepth is the most DAO executions that can be nested inside
// each other, the outermost included: a proposal or MsgAdminExecute running
// as a DAO is one level, and every MsgAdminExecute it runs for a sub-DAO adds
// one more. Running one level deeper fails with ErrExecutionTooDeep.
const MaxDaoExecutionDepth = 5
