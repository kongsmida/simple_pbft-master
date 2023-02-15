package consensus

type PBFT interface {
	StartConsensus(request *SingletxMsg) (*PrePrepareMsg, error)
	StartConsensusForAS(request *CrosstxMsg) (*ProposeMsg, error)
	PrePrepare(prePrepareMsg *PrePrepareMsg) (*VoteMsg, error)
	PreCommit(preCommitMsg *VoteMsg) (*VoteMsg, error)
	CommitLocal(commitLocalMsg *CommitLocalMsg)
	CommitFinal(commitFinal *VoteMsg) (*VoteMsg, error)
	Commit(commitMsg *VoteMsg) (*ReplyMsg, *SingletxMsg, error)
}
