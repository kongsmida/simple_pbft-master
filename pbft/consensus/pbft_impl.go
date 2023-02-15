package consensus

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

type State struct {
	ViewID         int64
	MsgLogs        *MsgLogs
	LastSequenceID int64
	CurrentStage   Stage
}

type MsgLogs struct {
	SingletxMsg         *SingletxMsg
	CrosstxMsg          *CrosstxMsg
	ProposeMsgs         map[string]*ProposeMsg
	PrePrepareMsgs      map[string]*PrePrepareMsg
	PrepareMsgs         map[string]*VoteMsg
	CommitMsgs          map[string]*VoteMsg
	CommitLocalMsgs     map[string]*CommitLocalMsg
	CommitFinalMsgs     map[string]*VoteMsg
	PreCommitMsgs       map[string]string
	ProofofProposalMsgs map[string]*ProofMsgs
	ProofofPrimaryMsgs  map[string]*ProofMsgs
	AggProofMsgs        map[string]*ProofMsgs
}

type Stage int

const (
	Idle        Stage = iota // Node is created successfully, but the consensus process is not started yet.
	PrePrepared              // The ReqMsgs is processed successfully. The node is ready to head to the Prepare stage.
	Prepared
	PreCommitted
	CommittedLocal
	CommittedFinal
	Committed // Same with `committed-local` stage explained in the original paper.
	Proposed
	Aggregated
)

// f: # of Byzantine faulty node  u: Byzantine nodes in AS
// f = (n­1) / 3
// n = 4, in this case.
const (
	f = 1
	u = 1
)

// lastSequenceID will be -1 if there is no last sequence ID.
func CreateState(viewID int64, lastSequenceID int64) *State {
	return &State{
		ViewID: viewID,
		MsgLogs: &MsgLogs{
			SingletxMsg:         nil,
			CrosstxMsg:          nil,
			PrePrepareMsgs:      nil,
			PrepareMsgs:         make(map[string]*VoteMsg, 0),
			ProposeMsgs:         make(map[string]*ProposeMsg, 0),
			CommitMsgs:          make(map[string]*VoteMsg, 0),
			CommitLocalMsgs:     make(map[string]*CommitLocalMsg, 0),
			CommitFinalMsgs:     make(map[string]*VoteMsg, 0),
			PreCommitMsgs:       make(map[string]string, 0),
			ProofofProposalMsgs: make(map[string]*ProofMsgs, 0),
			ProofofPrimaryMsgs:  make(map[string]*ProofMsgs, 0),
			AggProofMsgs:        make(map[string]*ProofMsgs, 0),
		},
		LastSequenceID: lastSequenceID,
		CurrentStage:   Idle,
	}
}

func (state *State) StartConsensus(request *SingletxMsg) (*PrePrepareMsg, error) {
	// `sequenceID` will be the index of this message.
	sequenceID := time.Now().UnixNano()

	// Find the unique and largest number for the sequence ID
	if state.LastSequenceID != -1 {
		for state.LastSequenceID >= sequenceID {
			sequenceID += 1
		}
	}

	// Assign a new sequence ID to the request message object.
	request.SequenceID = sequenceID

	// Save ReqMsgs to its logs.
	state.MsgLogs.SingletxMsg = request

	// Get the digest of the request message
	digest, err := digest(request)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Change the stage to pre-prepared.
	state.CurrentStage = PrePrepared

	return &PrePrepareMsg{
		ViewID:     state.ViewID,
		SequenceID: sequenceID,
		Digest:     digest,
		RequestMsg: request,
	}, nil
}

func (state *State) StartConsensusForAS(request *CrosstxMsg, subtx string) (*ProposeMsg, error) {
	sequenceID := time.Now().UnixNano()

	if state.LastSequenceID != -1 {
		for state.LastSequenceID >= sequenceID {
			sequenceID += 1
		}
	}

	request.SequenceID = sequenceID

	state.MsgLogs.CrosstxMsg = request

	state.CurrentStage = Idle

	return &ProposeMsg{
		ViewID:     state.ViewID,
		SequenceID: sequenceID,
		Subtx:      subtx,
	}, nil
}

func (state *State) PrePrepare(prePrepareMsg *PrePrepareMsg) (*VoteMsg, error) {
	// Get ReqMsgs and save it to its logs like the primary.
	state.MsgLogs.SingletxMsg = prePrepareMsg.RequestMsg

	// Verify if v, n(a.k.a. sequenceID), d are correct.
	if !state.verifyMsg(prePrepareMsg.ViewID, prePrepareMsg.SequenceID, prePrepareMsg.Digest) {
		return nil, errors.New("pre-prepare message is corrupted")
	}

	// Change the stage to pre-prepared.
	state.CurrentStage = PrePrepared

	return &VoteMsg{
		ViewID:     state.ViewID,
		SequenceID: prePrepareMsg.SequenceID,
		Digest:     prePrepareMsg.Digest,
		MsgType:    PrepareMsg,
	}, nil
}

// 此函数用于 直到主节点收到两个proposal，才能返回proofofproposalmsg（包含这两个proposal及提出者id）
func (state *State) Proposal(proposal *ProposeMsg) (*ProofMsgs, error) {
	state.MsgLogs.ProposeMsgs[proposal.NodeID] = proposal
	fmt.Printf("已经收到%d个proposal\n", len(state.MsgLogs.ProposeMsgs))
	if len(state.MsgLogs.ProposeMsgs) == 3 {
		if proposal.NodeID[4:6] == "00" {
			return &ProofMsgs{
				ViewID:              proposal.ViewID,
				Accepted:            true,
				ProofType:           ProofofProposalMsg,
				ProofofProposalMsgs: state.MsgLogs.ProposeMsgs,
				// PrimaryID:           proposal.NodeID,
			}, nil
		}
	}
	return nil, nil
}

func (state *State) PreCommit(preCommitMsg *ProofMsgs) (*SynchronizeMsg, error) {
	if !state.verifyProofMsg(preCommitMsg.ViewID, preCommitMsg.Accepted) {
		return nil, errors.New("preCommit message is corrupted")
	}

	// Append msg to its logs
	state.MsgLogs.PreCommitMsgs[preCommitMsg.NodeID] = preCommitMsg.NodeID

	// Print current voting status
	fmt.Printf("[Precommit-Vote]: %d from %s\n", len(state.MsgLogs.PreCommitMsgs), preCommitMsg.NodeID)

	if state.PreCommitted() {
		// Change the stage to prepared.
		state.CurrentStage = PreCommitted

		return &SynchronizeMsg{
			ViewID:   state.ViewID,
			AggProof: *preCommitMsg,
		}, nil
	}

	return nil, nil
}

func (state *State) Prepare(prepareMsg *VoteMsg) (*VoteMsg, error) {
	if !state.verifyMsg(prepareMsg.ViewID, prepareMsg.SequenceID, prepareMsg.Digest) {
		return nil, errors.New("prepare message is corrupted")
	}

	// Append msg to its logs
	state.MsgLogs.PrepareMsgs[prepareMsg.NodeID] = prepareMsg

	// Print current voting status
	fmt.Printf("[Prepare-Vote]: %d from %s\n", len(state.MsgLogs.PrepareMsgs), prepareMsg.NodeID)

	if state.Prepared() {

		// Change the stage to prepared.
		state.CurrentStage = Prepared

		return &VoteMsg{
			ViewID:     state.ViewID,
			SequenceID: prepareMsg.SequenceID,
			Digest:     prepareMsg.Digest,
			MsgType:    CommitMsg,
		}, nil
	}

	return nil, nil
}

func (state *State) Commit(commitMsg *VoteMsg) (*ReplyMsg, *SingletxMsg, error) {
	if !state.verifyMsg(commitMsg.ViewID, commitMsg.SequenceID, commitMsg.Digest) {
		return nil, nil, errors.New("commit message is corrupted")
	}

	// Append msg to its logs
	state.MsgLogs.CommitMsgs[commitMsg.NodeID] = commitMsg

	// Print current voting status
	fmt.Printf("[Commit-Vote]: %d from %s\n", len(state.MsgLogs.CommitMsgs), commitMsg.NodeID)

	if state.Committed() {
		// This node executes the requested operation locally and gets the result.
		result := "Executed"

		// Change the stage to prepared.
		state.CurrentStage = Committed

		return &ReplyMsg{
			ViewID:    state.ViewID,
			Timestamp: state.MsgLogs.SingletxMsg.Timestamp,
			ClientID:  state.MsgLogs.SingletxMsg.ClientID,
			Result:    result,
		}, state.MsgLogs.SingletxMsg, nil
	}

	return nil, nil, nil
}

func (state *State) ProofofPrimary(proofofPrimaryMsg *ProofMsgs) (*ProofMsgs, error) {
	if !state.verifyProofMsg(proofofPrimaryMsg.ViewID, proofofPrimaryMsg.Accepted) {
		return nil, errors.New("proof of primary message is corrupted")
	}
	// Append msg to its logs
	state.MsgLogs.ProofofPrimaryMsgs[proofofPrimaryMsg.NodeID] = proofofPrimaryMsg

	// Print current voting status
	fmt.Printf("[ProofofPrimaryMsg-Vote]: %d for %s from %s\n", len(state.MsgLogs.ProofofPrimaryMsgs), proofofPrimaryMsg.PrimaryID, proofofPrimaryMsg.NodeID)

	pID := strconv.FormatInt(proofofPrimaryMsg.ViewID%4, 10)

	if state.ProofofPrimaried() {
		//缺少一个BLS聚合签名的步骤，还得将本分片的StateDB放入ProofMsgs中一起返回

		state.CurrentStage = Aggregated

		return &ProofMsgs{
			ViewID:    state.ViewID,
			ProofType: AggProofMsg,
			PrimaryID: pID,
			Accepted:  true,
			StartTime: proofofPrimaryMsg.StartTime,
		}, nil
	}

	return nil, nil
}

func (state *State) AggProof(agg_proof *ProofMsgs) bool {
	//1. 验证每个aggProofMsg的正确性
	//2. 验证AggProofMsg的个数是否等于分片个数
	state.MsgLogs.AggProofMsgs[agg_proof.NodeID] = agg_proof
	fmt.Printf("[AggProof-Vote]: %d from %s\n", len(state.MsgLogs.AggProofMsgs), agg_proof.NodeID)
	if len(state.MsgLogs.AggProofMsgs) == 4 {
		return true
	}
	return false
}

func (state *State) verifyMsg(viewID int64, sequenceID int64, digestGot string) bool {
	// Wrong view. That is, wrong configurations of peers to start the consensus.
	if state.ViewID != viewID {
		return false
	}

	if state.LastSequenceID != -1 {
		if state.LastSequenceID >= sequenceID {
			return false
		}
	}

	digest, err := digest(state.MsgLogs.SingletxMsg)
	if err != nil {
		fmt.Println(err)
		return false
	}

	// Check digest.
	if digestGot != digest {
		return false
	}

	return true
}

func (state *State) verifyProofMsg(viewID int64, accepted bool) bool {
	if state.ViewID != viewID {
		return false
	}

	if accepted != true {
		return false
	}

	return true
}

func (state *State) proposed() bool {
	if state.MsgLogs.CrosstxMsg == nil {
		return false
	}

	if len(state.MsgLogs.ProofofProposalMsgs) < 2*f {
		return false
	}
	return true
}

func (state *State) PreCommitted() bool {
	if state.CurrentStage != Proposed {
		return false
	}

	if len(state.MsgLogs.PreCommitMsgs) < 5 {
		return false
	}

	return true
}

func (state *State) CommittedLocal() bool {
	if !state.PreCommitted() {
		return false
	}

	if len(state.MsgLogs.CommitLocalMsgs) < 2*f {
		return false
	}
	return true
}

func (state *State) CommittedFinal() bool {
	if !state.proposed() {
		return false
	}

	if len(state.MsgLogs.CommitFinalMsgs) < 2*f {
		return false
	}
	return true
}

func (state *State) Prepared() bool {
	if state.CurrentStage != PrePrepared {
		return false
	}

	if len(state.MsgLogs.PrepareMsgs) < 2*f {
		return false
	}

	return true
}

func (state *State) Committed() bool {
	if state.CurrentStage != Prepared {
		return false
	}

	if len(state.MsgLogs.CommitMsgs) < 2*f {
		return false
	}

	return true
}

func (state *State) ProofofPrimaried() bool {
	if len(state.MsgLogs.ProofofPrimaryMsgs) != 3 {
		return false
	}
	return true
}

func digest(object interface{}) (string, error) {
	msg, err := json.Marshal(object)

	if err != nil {
		return "", err
	}

	return Hash(msg), nil
}
