package consensus

type txType int

const (
	Singletx txType = 1
	Crosstx  txType = 0
)

type SingletxMsg struct {
	Timestamp  int64  `json:"timestamp"`
	ClientID   string `json:"clientID"`
	Operation  string `json:"operation"`
	SequenceID int64  `json:"sequenceID"`
}

type SynchronizeMsg struct {
	Subtx    string    `json:"subtx"`
	AggProof ProofMsgs `json:"aggProof"`
	NodeID   string    `json:"nodeID"`
	ViewID   int64     `json:"viewID"`
}

type CrosstxMsg struct {
	Timestamp  int64  `json:"timestamp"`
	ClientID   string `json:"clientID"`
	Operation  string `json:"operation"`
	SequenceID int64  `json:"sequenceID"`
}

type ProposeMsg struct {
	ViewID     int64  `json:"viewID"`
	SequenceID int64  `json:"sequenceID"`
	Subtx      string `json:"subtx"`
	NodeID     string `json:"nodeID"`
}

type CommitLocalMsg struct {
	ViewID     int64  `json:"viewID"`
	SequenceID int64  `json:"sequenceID"`
	Digest     string `json:"digest"`
	NodeID     string `json:"nodeID"`
}

type ReplyMsg struct {
	ViewID    int64  `json:"viewID"`
	Timestamp int64  `json:"timestamp"`
	ClientID  string `json:"clientID"`
	NodeID    string `json:"nodeID"`
	Result    string `json:"result"`
}

type PreReplyMsg struct {
	ViewID    int64  `json:"viewID"`
	Timestamp int64  `json:"timestamp"`
	PrimaryID string `json:"primaryID"`
	NodeID    string `json:"nodeID"`
	Subtx     string `json:"subtx"`
}

type PrePrepareMsg struct {
	ViewID     int64        `json:"viewID"`
	SequenceID int64        `json:"sequenceID"`
	Digest     string       `json:"digest"`
	RequestMsg *SingletxMsg `json:"requestMsg"`
}

type VoteMsg struct {
	ViewID     int64  `json:"viewID"`
	SequenceID int64  `json:"sequenceID"`
	Digest     string `json:"digest"`
	NodeID     string `json:"nodeID"`
	MsgType    `json:"msgType"`
}

type MsgType int

const (
	PrepareMsg MsgType = iota
	CommitMsg
	PreCommitMsg
	// CommitFinalMsg
)

type ProofMsgs struct {
	ViewID              int64  `json:"viewID"`
	Accepted            bool   `json:"accepted"`
	PrimaryID           string `json:"primaryID"`
	StartTime           int64  `json:"startTime"`
	ProofType           `json:"proofType"`
	NodeID              string                 `json:"nodeID"`
	ProofofProposalMsgs map[string]*ProposeMsg `json:"proofofProposal"`
}

type ProofType int

const (
	ProofofProposalMsg ProofType = iota
	ProofofPrimaryMsg
	AggProofMsg
)
