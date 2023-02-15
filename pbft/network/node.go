package network

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/kongsmida/simple_pbft-master/pbft/consensus"
)

type StateDB struct {
	Address string
	Balance float64
}

type Node struct {
	NodeID           string
	ShardID          string //加
	View             *View
	CurrentState     *consensus.State
	CommittedMsgs    []*consensus.SingletxMsg //包含此节点已提交的所有片内交易
	CommittedCroMsgs []*consensus.CrosstxMsg  //包含此节点已提交的所有跨分片交易
	MsgBuffer        *MsgBuffer               //一个包含所有类型消息的缓冲区
	MsgEntrance      chan interface{}
	MsgDelivery      chan interface{}
	Alarm            chan bool
	StateDB          []StateDB //加
	MenberShip       []string  //加
}

type MsgBuffer struct {
	SingletxMsgs   []*consensus.SingletxMsg
	PrePrepareMsgs []*consensus.PrePrepareMsg
	PrepareMsgs    []*consensus.VoteMsg
	CommitMsgs     []*consensus.VoteMsg

	CrosstxMsgs         []*consensus.CrosstxMsg
	ProposeMsgs         []*consensus.ProposeMsg
	ProofofProposalMsgs []*consensus.ProofMsgs
	PreCommitMsgs       []*consensus.VoteMsg
	SynchronizeMsg      []*consensus.SynchronizeMsg

	ProofofPrimaryMsgs []*consensus.ProofMsgs //从节点提交给主节点的身份证明消息
	AggProofMsgs       []*consensus.ProofMsgs //收到的主节点的身份聚合签名
}

type View struct {
	ID      int64
	Primary []string
}

var MenberShip_1 = []string{"000100", "000101", "000110", "000111"}
var MenberShip_2 = []string{"001000", "001001", "001010", "001011"}
var MenberShip_3 = []string{"001100", "001101", "001110", "001111"}
var MenberShip_4 = []string{"010000", "010001", "010010", "010011"}
var MenberShip_5 = []string{"010100", "010101", "010110", "010111"}

var AssistantShard = []string{"000100", "001000", "001100", "010000", "010100"}

var nodeDB []string
var stateDB []StateDB
var menberShip []string

const ResolvingTimeDuration = time.Millisecond * 1000 // 1 second.

func NewNode(nodeID string) *Node {
	const viewID = 0000 // temporary.
	// 节点同步所在分片的stateDB
	shardID := nodeID[0:4]
	switch shardID {
	case "0001":
		{
			nodeDB = MenberShip_1
			stateDB = []StateDB{{"A", 20}, {"B", 30}, {"C", 40}}
			menberShip = MenberShip_1
		}
	case "0010":
		{
			nodeDB = MenberShip_2
			stateDB = []StateDB{{"D", 12}, {"E", 13}, {"F", 14}}
			menberShip = MenberShip_2
		}
	case "0011":
		{
			nodeDB = MenberShip_3
			stateDB = []StateDB{{"G", 22}, {"H", 23}, {"I", 24}}
			menberShip = MenberShip_3
		}
	case "0100":
		{
			nodeDB = MenberShip_4
			stateDB = []StateDB{{"J", 32}, {"K", 33}, {"L", 34}}
			menberShip = MenberShip_4
		}
	case "0101":
		{
			nodeDB = MenberShip_5
			stateDB = []StateDB{{"M", 42}, {"N", 43}, {"O", 44}}
			menberShip = MenberShip_5
		}
	default:
		{
			fmt.Printf("找不到对应的分片\n")
		}
	}
	// 主节点同步全部stateDB
	primaryID := nodeID[4:6]
	if primaryID == "00" {
		stateDB = []StateDB{{"A", 2}, {"B", 3}, {"C", 4}, {"D", 2},
			{"E", 3}, {"F", 4}, {"G", 2}, {"H", 3}, {"I", 4},
			{"J", 2}, {"K", 3}, {"L", 4}, {"M", 2}, {"N", 3}, {"O", 4}}
		menberShip = AssistantShard
	}

	node := &Node{
		// Hard-coded for test.
		NodeID:  nodeID,
		ShardID: shardID,
		View: &View{
			ID:      viewID,
			Primary: AssistantShard,
		},
		StateDB:    stateDB,
		MenberShip: menberShip,

		// Consensus-related struct
		CurrentState:     nil,
		CommittedMsgs:    make([]*consensus.SingletxMsg, 0),
		CommittedCroMsgs: make([]*consensus.CrosstxMsg, 0),
		MsgBuffer: &MsgBuffer{
			SingletxMsgs:        make([]*consensus.SingletxMsg, 0),
			CrosstxMsgs:         make([]*consensus.CrosstxMsg, 0),
			PrePrepareMsgs:      make([]*consensus.PrePrepareMsg, 0),
			PrepareMsgs:         make([]*consensus.VoteMsg, 0),
			CommitMsgs:          make([]*consensus.VoteMsg, 0),
			ProposeMsgs:         make([]*consensus.ProposeMsg, 0),
			ProofofProposalMsgs: make([]*consensus.ProofMsgs, 0),
			ProofofPrimaryMsgs:  make([]*consensus.ProofMsgs, 0),
			AggProofMsgs:        make([]*consensus.ProofMsgs, 0),
		},

		// Channels 代表并发管道
		MsgEntrance: make(chan interface{}, 10),
		MsgDelivery: make(chan interface{}, 10),
		Alarm:       make(chan bool),
	}

	// Start message dispatcher
	go node.dispatchMsg()

	// Start alarm trigger
	go node.alarmToDispatcher()

	// Start message resolver
	go node.resolveMsg()

	return node
}

func (node *Node) SelectBalance(operation string) bool {
	flysnowRegexp := regexp.MustCompile(`^From:([\w]{1}),To:([\w]{1}),Value:([\-|\+]?\d+[\.\d+]?)$`)
	var params []string
	params = flysnowRegexp.FindStringSubmatch(operation)
	from := params[1]
	value, _ := strconv.ParseFloat(params[3], 64)
	for _, v := range stateDB {
		if v.Address == from {
			if v.Balance >= value {
				return true
			}
		}
	}
	return false
}

func (node *Node) ModifyBalance(operation string) {
	flysnowRegexp := regexp.MustCompile(`^From:([\w]{1}),To:([\w]{1}),Value:([\-|\+]?\d+[\.\d+]?)$`)
	var params []string
	params = flysnowRegexp.FindStringSubmatch(operation)
	fmt.Println("params:", params)
	from := params[1]
	to := params[2]
	value, _ := strconv.ParseFloat(params[3], 64)
	if from == to {
		for i, v := range stateDB {
			if v.Address == from {
				v.Balance += value
				stateDB[i].Balance = v.Balance
			}
		}
	} else {
		for i, v := range stateDB {
			if v.Address == from {
				v.Balance -= value
				stateDB[i].Balance = v.Balance
			}
			if v.Address == to {
				v.Balance += value
				stateDB[i].Balance = v.Balance
			}
		}
	}

	node.StateDB = stateDB
}

func (node *Node) Broadcast(msg interface{}, path string) map[string]error {
	errorMap := make(map[string]error)
	var Mer []string
	switch node.NodeID[0:4] {
	case "0001":
		Mer = MenberShip_1
	case "0010":
		Mer = MenberShip_2
	case "0011":
		Mer = MenberShip_3
	case "0100":
		Mer = MenberShip_4
	case "0101":
		Mer = MenberShip_5
	}
	for _, url := range Mer {
		if url == node.NodeID {
			continue
		}
		jsonMsg, err := json.Marshal(msg) //将msg编码为json格式存入jsonMsg
		if err != nil {
			errorMap[url[3:6]] = err
			continue
		}
		println("sendstring:", "localhost:"+url+path)
		send("localhost:"+url+path, jsonMsg)
	}

	if len(errorMap) == 0 {
		return nil
	} else {
		return errorMap
	}
}

func (node *Node) BroadcastForAS(msg interface{}, path string) map[string]error {
	errorMap := make(map[string]error)

	for _, url := range AssistantShard {

		jsonMsg, err := json.Marshal(msg) //将msg编码为json格式存入jsonMsg
		if err != nil {
			errorMap[url[3:6]] = err
			continue
		}
		println("sendstring:", "localhost:"+url+path)
		send("localhost:"+url+path, jsonMsg)
	}

	if len(errorMap) == 0 {
		return nil
	} else {
		return errorMap
	}
}

func (node *Node) Reply(msg *consensus.ReplyMsg) error { //只给主节点返回reply消息的json数据
	// Print all committed messages.
	for _, value := range node.CommittedMsgs {
		fmt.Printf("Committed value: %s, %d, %s, %d\n", value.ClientID, value.Timestamp, value.Operation, value.SequenceID)
	}
	// jsonMsg, err := json.Marshal(msg)
	// if err != nil {
	// 	return err
	// }

	// // 这里要改成返回给client
	// for _, i := range AssistantShard {
	// 	fmt.Println(i)
	// 	send("localhost:"+i+"/reply", jsonMsg)
	// }

	return nil
}

// GetReq can be called when the node's CurrentState is nil.
// Consensus start procedure for the Primary.
func (node *Node) GetReq(reqMsg *consensus.SingletxMsg) error {
	for i, v := range node.StateDB {
		if i%2 == 0 {
			fmt.Print("\n")
		}
		fmt.Print(v.Address, ":", v.Balance, " ")
	}
	// LogMsg(reqMsg) //打印消息

	// 在新一轮共识前，创建一个新的状态，即节点此时位于哪个视图、最新的一个requestID
	err := node.createStateForNewConsensus()
	if err != nil {
		return err
	}

	if !node.SelectBalance(reqMsg.Operation) {
		node.CurrentState = nil
		return errors.New(fmt.Sprint("Do not have enough balance"))
	}
	// 开始共识，创建一个pre-prepare消息，包含viewID，序列号，摘要，请求消息
	prePrepareMsg, err := node.CurrentState.StartConsensus(reqMsg)
	if err != nil {
		return err
	}

	LogStage(fmt.Sprintf("Consensus Process (ViewID:%d)", node.CurrentState.ViewID), false)

	// 广播pre-prepare消息并打印
	if prePrepareMsg != nil {
		node.Broadcast(prePrepareMsg, "/preprepare")
		LogStage("Pre-prepared", true)
	}

	return nil
}

func (node *Node) GetCroReq(croreqMsg *consensus.CrosstxMsg) error {
	for i, v := range node.StateDB {
		if i < 6 {
			fmt.Println(v.Address, ":", v.Balance)
		}
	}
	// LogMsg(croreqMsg) //打印消息

	// 在新一轮共识前，创建一个新的状态，即节点此时位于哪个视图、最新的一个requestID
	if node.CurrentState == nil {
		err := node.createStateForNewConsensus()
		if err != nil {
			return err
		}
	}

	//检测fake tx：cross-shard tx的value是否小于等于输入方的余额
	flysnowRegexp := regexp.MustCompile(`^From:([\w]{1}),To:([\w]{1}),Value:([\-|\+]?\d+[\.\d+]?)$`)
	var params []string
	params = flysnowRegexp.FindStringSubmatch(croreqMsg.Operation)
	from := params[1] //A
	to := params[2]   //D
	v := params[3]    //1
	var flag bool
	flag = false
	//如果跨分片交易无误，则主节点先执行，即修改自己的本地数据库
	if node.SelectBalance(croreqMsg.Operation) {
		flag = true
		node.ModifyBalance(croreqMsg.Operation)
	}
	if flag == false {
		Logfile(croreqMsg.Operation, node.NodeID+"_faketx")
		fmt.Println("This is a fake tx")
		return nil
	}
	for i, v := range node.StateDB {
		if i < 6 {
			fmt.Println(v.Address, ":", v.Balance)
		}
	}

	// 检测通过后，开始划分子交易
	var subtx string
	var start, end int
	if node.NodeID[0:4] == "0001" {
		start = 0
		end = 2
	} else if node.NodeID[0:4] == "0010" {
		start = 3
		end = 5
	} else if node.NodeID[0:4] == "0011" {
		start = 6
		end = 8
	} else if node.NodeID[0:4] == "0100" {
		start = 9
		end = 11
	} else {
		start = 12
		end = 15
	}
	for i, s := range node.StateDB {
		if s.Address == from {
			if i >= start && i <= end {
				subtx = "From:" + from + ",To:" + from + ",Value:-" + v
				break
			}
		}
		if s.Address == to {
			if i >= start && i <= end {
				subtx = "From:" + to + ",To:" + to + ",Value:" + v
				break
			}
		}
	}
	if subtx != "" {
		fmt.Println("subtx:", subtx)
	} else {
		return nil
	}
	// 创建propose消息，包含原始的跨分片请求消息、新的子交易、sequenceID、viewID
	proposeMsg, err := node.CurrentState.StartConsensusForAS(croreqMsg, subtx)
	if err != nil {
		return err
	}

	LogStage(fmt.Sprintf("Consensus Process (ViewID:%d)", node.CurrentState.ViewID), false)

	// 只有所涉及的主节点才需要广播proposeMsg给其他主节点
	// 广播propose消息并打印
	if proposeMsg != nil {
		proposeMsg.NodeID = node.NodeID
		node.BroadcastForAS(proposeMsg, "/propose")
		LogStage("Idle", true)

	}
	return nil
}

// GetPrePrepare can be called when the node's CurrentState is nil.
// Consensus start procedure for normal participants.
func (node *Node) GetPrePrepare(prePrepareMsg *consensus.PrePrepareMsg) error {
	// LogMsg(prePrepareMsg)

	// Create a new state for the new consensus.
	err := node.createStateForNewConsensus()
	if err != nil {
		return err
	}

	prepareMsg, err := node.CurrentState.PrePrepare(prePrepareMsg)
	if err != nil {
		return err
	}

	if prepareMsg != nil {
		// Attach node ID to the message
		prepareMsg.NodeID = node.NodeID

		LogStage("Pre-prepared", true)
		node.Broadcast(prepareMsg, "/prepare")
		LogStage("Prepare", false)
	}

	return nil
}

func (node *Node) GetPropose(proposeMsg *consensus.ProposeMsg) error {
	// LogMsg(proposeMsg)
	if node.CurrentState == nil {
		node.createStateForNewConsensus()
	}
	proofofProposal, err := node.CurrentState.Proposal(proposeMsg)
	if err != nil {
		return err
	}
	if proofofProposal != nil {
		fmt.Println(node.NodeID, "认同所有的proposals,已生成Proof of Proposal")
		node.CurrentState.CurrentStage = consensus.Proposed
		proofofProposal.NodeID = node.NodeID
		LogStage("Proposed", true)
		node.BroadcastForAS(proofofProposal, "/precommit")
		LogStage("Precommit", false)
	}
	return nil
}

var T_complete int64

func (node *Node) GetPreCommit(preCommitMsg *consensus.ProofMsgs) error {
	// LogMsg(preCommitMsg)

	synchronizemsg, err := node.CurrentState.PreCommit(preCommitMsg)
	if err != nil {
		return err
	}

	if synchronizemsg != nil {
		LogStage("Precommitted", true)
		// Attach node ID to the message
		synchronizemsg.NodeID = node.NodeID
		for _, v := range preCommitMsg.ProofofProposalMsgs {
			if v.NodeID[0:4] == node.NodeID[0:4] {
				synchronizemsg.Subtx = v.Subtx
				node.Broadcast(synchronizemsg, "/synchronize")
				LogStage("Backup nodes are synchronizing!!!", false)
				T_complete = time.Now().UnixNano()
				Logfile(v.Subtx+"\n Execute time:"+strconv.FormatInt(T_complete-T_start_as, 10), node.NodeID)
			}
		}
		node.CurrentState = nil
	}

	return nil
}

func (node *Node) GetSyn(synchronizemsg *consensus.SynchronizeMsg) error {
	for _, v := range node.StateDB {
		fmt.Println(v)
	}
	fmt.Println(synchronizemsg.AggProof.PrimaryID)
	if node.NodeID[4:6] != "00" {
		node.ModifyBalance(synchronizemsg.Subtx)
	}
	for _, v := range node.StateDB {
		fmt.Println(v)
	}
	LogStage("Backup nodes have synchronized!!!", false)
	return nil
}

func (node *Node) GetPrepare(prepareMsg *consensus.VoteMsg) error {
	// LogMsg(prepareMsg)

	commitMsg, err := node.CurrentState.Prepare(prepareMsg)
	if err != nil {
		return err
	}

	if commitMsg != nil {
		// Attach node ID to the message
		commitMsg.NodeID = node.NodeID

		LogStage("Prepared", true)
		node.Broadcast(commitMsg, "/commit")
		LogStage("Commit", false)
	}

	return nil
}

func (node *Node) GetCommit(commitMsg *consensus.VoteMsg) error {
	// LogMsg(commitMsg)

	replyMsg, committedMsg, err := node.CurrentState.Commit(commitMsg)
	if err != nil {
		node.CurrentState = nil
		return err
	}

	if replyMsg != nil {
		if committedMsg == nil {
			return errors.New("committed message is nil, even though the reply message is not nil")
		}

		fmt.Println(committedMsg.Operation)
		node.ModifyBalance(committedMsg.Operation)
		fmt.Println(node.StateDB)
		// Attach node ID to the message
		replyMsg.NodeID = node.NodeID
		T_complete = time.Now().UnixNano()
		Logfile(committedMsg.Operation+"\nExecute time:"+strconv.FormatInt(T_complete-T_start_req, 10), node.NodeID+"inShard")
		// Save the last version of committed messages to node.
		node.CommittedMsgs = append(node.CommittedMsgs, committedMsg)

		LogStage("Committed", true)
		node.Reply(replyMsg)
		LogStage("Reply", true)
		node.CurrentState = nil
	}

	return nil
}

// 从节点生成主节点身份证明并转发给主节点
func (node *Node) GetAS() {
	var proofofprimarymsg consensus.ProofMsgs
	proofofprimarymsg.ViewID = node.View.ID
	proofofprimarymsg.Accepted = true
	proofofprimarymsg.PrimaryID = node.NodeID[0:4] + "00"
	proofofprimarymsg.ProofType = consensus.ProofofPrimaryMsg
	proofofprimarymsg.NodeID = node.NodeID
	proofofprimarymsg.StartTime = T_start_as

	//对消息prooofprimarymsg签名
	jsonMsg, _ := json.Marshal(proofofprimarymsg)
	println("sendstring:", "localhost:"+proofofprimarymsg.PrimaryID+"/asf")
	send("localhost:"+proofofprimarymsg.PrimaryID+"/asf", jsonMsg)
}

// 主节点调用
func (node *Node) GetProofofPrimary(proofs *consensus.ProofMsgs) {
	if node.CurrentState == nil {
		node.createStateForNewConsensus()
	}
	// LogMsg(proofs)
	pai, _ := node.CurrentState.ProofofPrimary(proofs)
	// 将从节点发来的proofofprimarymsg聚合成pai，转发给其余所有主节点
	if pai != nil {
		pai.NodeID = node.NodeID
		jsonMsg, _ := json.Marshal(pai)
		for _, v := range AssistantShard {
			if v == node.NodeID {
				continue
			}
			fmt.Println("sendstring:" + v + "/asF")
			send("localhost:"+v+"/asF", jsonMsg)
		}
	} else {
		fmt.Println(pai, "is null")
	}
}

// 主节点调用
func (node *Node) GetAggProof(proofs *consensus.ProofMsgs) {
	// LogMsg(proofs)
	if node.CurrentState.AggProof(proofs) {
		fmt.Println("AS Formation Success!------", time.Now().UnixNano()-proofs.StartTime)
	}
}

func (node *Node) GetReply(msg *consensus.ReplyMsg) {
	fmt.Printf("Result: %s by %s\n", msg.Result, msg.NodeID)
}

func (node *Node) createStateForNewConsensus() error {
	// Check if there is an ongoing consensus process.
	if node.CurrentState != nil && node.CurrentState.CurrentStage != consensus.Idle {
		return errors.New("another consensus is ongoing")
	}

	// Get the last sequence ID
	var lastSequenceID int64
	if len(node.CommittedMsgs) == 0 {
		lastSequenceID = -1
	} else {
		lastSequenceID = node.CommittedMsgs[len(node.CommittedMsgs)-1].SequenceID
	}

	// Create a new state for this new consensus process in the Primary
	node.CurrentState = consensus.CreateState(node.View.ID, lastSequenceID)

	LogStage("Create the replica status", true)

	return nil
}

func (node *Node) dispatchMsg() {
	for {
		select {
		case msg := <-node.MsgEntrance:
			err := node.routeMsg(msg)
			if err != nil {
				fmt.Println(err)
				// TODO: send err to ErrorChannel
			}
		case <-node.Alarm:
			err := node.routeMsgWhenAlarmed()
			if err != nil {
				fmt.Println(err)
				// TODO: send err to ErrorChannel
			}
		}
	}
}

func (node *Node) routeMsg(msg interface{}) []error {
	switch msg.(type) {
	case *consensus.SingletxMsg:
		if node.CurrentState == nil {
			// Copy buffered messages first.
			msgs := make([]*consensus.SingletxMsg, len(node.MsgBuffer.SingletxMsgs))
			copy(msgs, node.MsgBuffer.SingletxMsgs)

			// Append a newly arrived message.
			msgs = append(msgs, msg.(*consensus.SingletxMsg))

			// Empty the buffer.
			node.MsgBuffer.SingletxMsgs = make([]*consensus.SingletxMsg, 0)

			// Send messages to MsgDelivery通道(这边传出去后，resolveMsg goroutine会立马接收这个通道中的消息).
			node.MsgDelivery <- msgs

			//node.CrosstxDelivery <- msgs
		} else { //当前状态不为空，就得等执行完当前任务才能处理此msg，则此msg会先进入MsgBuffer中
			node.MsgBuffer.SingletxMsgs = append(node.MsgBuffer.SingletxMsgs, msg.(*consensus.SingletxMsg))
		}
	case *consensus.CrosstxMsg:
		if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Idle {
			// Copy buffered messages first.
			msgs := make([]*consensus.CrosstxMsg, len(node.MsgBuffer.CrosstxMsgs))
			copy(msgs, node.MsgBuffer.CrosstxMsgs)
			// Append a newly arrived message.
			msgs = append(msgs, msg.(*consensus.CrosstxMsg))
			// Empty the buffer.
			node.MsgBuffer.CrosstxMsgs = make([]*consensus.CrosstxMsg, 0)
			// Send messages to MsgDelivery通道(这边传出去后，resolveMsg goroutine会立马接收这个通道中的消息).
			node.MsgDelivery <- msgs
		} else { //当前状态不为空，就得等执行完当前任务才能处理此msg，则此msg会先进入MsgBuffer中
			node.MsgBuffer.CrosstxMsgs = append(node.MsgBuffer.CrosstxMsgs, msg.(*consensus.CrosstxMsg))
		}
	case *consensus.PrePrepareMsg:
		if node.CurrentState == nil {
			// Copy buffered messages first.
			msgs := make([]*consensus.PrePrepareMsg, len(node.MsgBuffer.PrePrepareMsgs))
			copy(msgs, node.MsgBuffer.PrePrepareMsgs)

			// Append a newly arrived message.
			msgs = append(msgs, msg.(*consensus.PrePrepareMsg))

			// Empty the buffer.
			node.MsgBuffer.PrePrepareMsgs = make([]*consensus.PrePrepareMsg, 0)

			// Send messages.
			node.MsgDelivery <- msgs
		} else {
			node.MsgBuffer.PrePrepareMsgs = append(node.MsgBuffer.PrePrepareMsgs, msg.(*consensus.PrePrepareMsg))
		}
	case *consensus.ProposeMsg:
		if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Idle {
			msgs := make([]*consensus.ProposeMsg, len(node.MsgBuffer.ProposeMsgs))
			copy(msgs, node.MsgBuffer.ProposeMsgs)
			msgs = append(msgs, msg.(*consensus.ProposeMsg))
			node.MsgBuffer.ProposeMsgs = make([]*consensus.ProposeMsg, 0)
			node.MsgDelivery <- msgs
		} else {
			node.MsgBuffer.ProposeMsgs = append(node.MsgBuffer.ProposeMsgs, msg.(*consensus.ProposeMsg))
		}
	case *consensus.VoteMsg:
		if msg.(*consensus.VoteMsg).MsgType == consensus.PreCommitMsg {
			if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.Proposed {
				node.MsgBuffer.PreCommitMsgs = append(node.MsgBuffer.PreCommitMsgs, msg.(*consensus.VoteMsg))
			} else {
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.PrepareMsgs))
				copy(msgs, node.MsgBuffer.PrepareMsgs)
				msgs = append(msgs, msg.(*consensus.VoteMsg))
				node.MsgBuffer.PreCommitMsgs = make([]*consensus.VoteMsg, 0)
				node.MsgDelivery <- msgs
			}
		} else if msg.(*consensus.VoteMsg).MsgType == consensus.CommitMsg {
			if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.Prepared {
				node.MsgBuffer.CommitMsgs = append(node.MsgBuffer.CommitMsgs, msg.(*consensus.VoteMsg))
			} else {
				// Copy buffered messages first.
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.CommitMsgs))
				copy(msgs, node.MsgBuffer.CommitMsgs)
				msgs = append(msgs, msg.(*consensus.VoteMsg))
				node.MsgBuffer.CommitMsgs = make([]*consensus.VoteMsg, 0)
				node.MsgDelivery <- msgs
			}
		} else if msg.(*consensus.VoteMsg).MsgType == consensus.PrepareMsg {
			if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.PrePrepared {
				node.MsgBuffer.PrepareMsgs = append(node.MsgBuffer.PrepareMsgs, msg.(*consensus.VoteMsg))
			} else {
				// Copy buffered messages first.
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.PrepareMsgs))
				copy(msgs, node.MsgBuffer.PrepareMsgs)

				// Append a newly arrived message.
				msgs = append(msgs, msg.(*consensus.VoteMsg))

				// Empty the buffer.
				node.MsgBuffer.PrepareMsgs = make([]*consensus.VoteMsg, 0)

				// Send messages.
				node.MsgDelivery <- msgs
			}
		}
	case *consensus.ProofMsgs:
		if msg.(*consensus.ProofMsgs).ProofType == consensus.ProofofPrimaryMsg {
			if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Idle {
				msgs := make([]*consensus.ProofMsgs, len(node.MsgBuffer.ProofofPrimaryMsgs))
				copy(msgs, node.MsgBuffer.ProofofPrimaryMsgs)
				// Append a newly arrived message.
				msgs = append(msgs, msg.(*consensus.ProofMsgs))
				node.MsgBuffer.ProofofPrimaryMsgs = make([]*consensus.ProofMsgs, 0)
				node.MsgDelivery <- msgs
			} else {
				node.MsgBuffer.ProofofPrimaryMsgs = append(node.MsgBuffer.ProofofPrimaryMsgs, msg.(*consensus.ProofMsgs))
			}
		} else if msg.(*consensus.ProofMsgs).ProofType == consensus.ProofofProposalMsg {
			if node.CurrentState != nil && node.CurrentState.CurrentStage == consensus.Proposed {
				msgs := make([]*consensus.ProofMsgs, len(node.MsgBuffer.ProofofProposalMsgs))
				copy(msgs, node.MsgBuffer.ProofofProposalMsgs)
				// Append a newly arrived message.
				msgs = append(msgs, msg.(*consensus.ProofMsgs))
				node.MsgBuffer.ProofofProposalMsgs = make([]*consensus.ProofMsgs, 0)
				node.MsgDelivery <- msgs
			} else {
				node.MsgBuffer.ProofofProposalMsgs = append(node.MsgBuffer.ProofofProposalMsgs, msg.(*consensus.ProofMsgs))
			}
		} else if msg.(*consensus.ProofMsgs).ProofType == consensus.AggProofMsg {
			if node.CurrentState != nil && node.CurrentState.CurrentStage == consensus.Aggregated {
				msgs := make([]*consensus.ProofMsgs, len(node.MsgBuffer.AggProofMsgs))

				// Copy buffered messages first.
				copy(msgs, node.MsgBuffer.AggProofMsgs)

				// Append a newly arrived message.
				msgs = append(msgs, msg.(*consensus.ProofMsgs))

				// Empty the buffer.
				node.MsgBuffer.AggProofMsgs = make([]*consensus.ProofMsgs, 0)

				// Send messages.
				node.MsgDelivery <- msgs
			} else {
				node.MsgBuffer.AggProofMsgs = append(node.MsgBuffer.AggProofMsgs, msg.(*consensus.ProofMsgs))
			}
		}
	case *consensus.SynchronizeMsg:
		if node.CurrentState == nil {
			msgs := make([]*consensus.SynchronizeMsg, len(node.MsgBuffer.SynchronizeMsg))

			// Copy buffered messages first.
			copy(msgs, node.MsgBuffer.SynchronizeMsg)

			// Append a newly arrived message.
			msgs = append(msgs, msg.(*consensus.SynchronizeMsg))

			// Empty the buffer.
			node.MsgBuffer.SynchronizeMsg = make([]*consensus.SynchronizeMsg, 0)

			// Send messages.
			node.MsgDelivery <- msgs
		}
	}
	return nil
}

func (node *Node) routeMsgWhenAlarmed() []error {
	if node.CurrentState == nil {
		// Check ReqMsgs, send them.
		if len(node.MsgBuffer.SingletxMsgs) != 0 {
			msgs := make([]*consensus.SingletxMsg, len(node.MsgBuffer.SingletxMsgs))
			copy(msgs, node.MsgBuffer.SingletxMsgs)

			node.MsgDelivery <- msgs
		}

		// Check PrePrepareMsgs, send them.
		if len(node.MsgBuffer.PrePrepareMsgs) != 0 {
			msgs := make([]*consensus.PrePrepareMsg, len(node.MsgBuffer.PrePrepareMsgs))
			copy(msgs, node.MsgBuffer.PrePrepareMsgs)

			node.MsgDelivery <- msgs
		}
	} else {
		switch node.CurrentState.CurrentStage {
		case consensus.PrePrepared:
			// Check PrepareMsgs, send them.
			if len(node.MsgBuffer.PrepareMsgs) != 0 {
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.PrepareMsgs))
				copy(msgs, node.MsgBuffer.PrepareMsgs)

				node.MsgDelivery <- msgs
			}
		case consensus.PreCommitted:
			// Check CommitMsgs, send them.
			if len(node.MsgBuffer.PreCommitMsgs) != 0 {
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.PreCommitMsgs))
				copy(msgs, node.MsgBuffer.PreCommitMsgs)

				node.MsgDelivery <- msgs
			}
		}
	}

	return nil
}

func (node *Node) resolveMsg() {
	for {
		// Get buffered messages from the dispatcher.
		msgs := <-node.MsgDelivery
		switch msgs.(type) {
		case []*consensus.SynchronizeMsg:
			errs := node.resolveSynMsg(msgs.([]*consensus.SynchronizeMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
			}
		case []*consensus.SingletxMsg:
			errs := node.resolveRequestMsg(msgs.([]*consensus.SingletxMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
			}
		case []*consensus.CrosstxMsg:
			errs := node.resolveCroRequestMsg(msgs.([]*consensus.CrosstxMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
			}
		case []*consensus.PrePrepareMsg:
			errs := node.resolvePrePrepareMsg(msgs.([]*consensus.PrePrepareMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
			}
		case []*consensus.ProposeMsg:
			errs := node.resolveProposeMsg(msgs.([]*consensus.ProposeMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
			}
		case []*consensus.VoteMsg:
			voteMsgs := msgs.([]*consensus.VoteMsg)
			if len(voteMsgs) == 0 {
				break
			}
			if voteMsgs[0].MsgType == consensus.PrepareMsg {
				errs := node.resolvePrepareMsg(voteMsgs)
				if len(errs) != 0 {
					for _, err := range errs {
						fmt.Println(err)
					}
				}
			} else if voteMsgs[0].MsgType == consensus.CommitMsg {
				errs := node.resolveCommitMsg(voteMsgs)
				if len(errs) != 0 {
					for _, err := range errs {
						fmt.Println(err)
					}
				}
			}
		case []*consensus.ProofMsgs:
			proofMsgs := msgs.([]*consensus.ProofMsgs)
			if len(proofMsgs) == 0 {
				break
			}
			if proofMsgs[0].ProofType == consensus.ProofofPrimaryMsg {
				errs := node.resolveProofofPrimaryMsg(proofMsgs)
				if len(errs) != 0 {
					for _, err := range errs {
						fmt.Println(err)
					}
				}
			} else if proofMsgs[0].ProofType == consensus.AggProofMsg {
				errs := node.resolveAggProofMsg(proofMsgs)
				if len(errs) != 0 {
					for _, err := range errs {
						fmt.Println(err)
					}
				}
			} else if proofMsgs[0].ProofType == consensus.ProofofProposalMsg {
				errs := node.resolvePreCommitMsg(proofMsgs)
				if len(errs) != 0 {
					for _, err := range errs {
						fmt.Println(err)
					}
				}
			}
		}
	}
}

func (node *Node) alarmToDispatcher() {
	for {
		time.Sleep(ResolvingTimeDuration)
		node.Alarm <- true
	}
}

func (node *Node) resolveSynMsg(msgs []*consensus.SynchronizeMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, synMsg := range msgs {
		err := node.GetSyn(synMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolveRequestMsg(msgs []*consensus.SingletxMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, reqMsg := range msgs {
		err := node.GetReq(reqMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolveCroRequestMsg(msgs []*consensus.CrosstxMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, reqMsg := range msgs {
		if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Idle {
			err := node.GetCroReq(reqMsg)
			if err != nil {
				errs = append(errs, err)
			}
		}

	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolvePrePrepareMsg(msgs []*consensus.PrePrepareMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, prePrepareMsg := range msgs {
		fmt.Println("this is a preprepare msg")
		err := node.GetPrePrepare(prePrepareMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolveProposeMsg(msgs []*consensus.ProposeMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, proposeMsg := range msgs {
		if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Idle {
			err := node.GetPropose(proposeMsg)
			if err != nil {
				errs = append(errs, err)
			}
		}

	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolvePreCommitMsg(msgs []*consensus.ProofMsgs) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, precommitMsg := range msgs {
		if node.CurrentState.CurrentStage == consensus.Proposed {
			err := node.GetPreCommit(precommitMsg)
			if err != nil {
				errs = append(errs, err)
			}
		}

	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolvePrepareMsg(msgs []*consensus.VoteMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, prepareMsg := range msgs {
		if node.CurrentState.CurrentStage == consensus.PrePrepared {
			err := node.GetPrepare(prepareMsg)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolveCommitMsg(msgs []*consensus.VoteMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, commitMsg := range msgs {
		if node.CurrentState != nil {
			if node.CurrentState.CurrentStage == consensus.Prepared {
				err := node.GetCommit(commitMsg)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}

	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolveProofofPrimaryMsg(msgs []*consensus.ProofMsgs) []error {
	errs := make([]error, 0)
	// Resolve messages
	for _, proofofPrimaryMsg := range msgs {
		if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Idle {
			node.GetProofofPrimary(proofofPrimaryMsg)
		}

	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolveAggProofMsg(msgs []*consensus.ProofMsgs) []error {
	errs := make([]error, 0)

	for _, agg_proof := range msgs {
		if node.CurrentState != nil {
			if node.CurrentState.CurrentStage == consensus.Aggregated {
				node.GetAggProof(agg_proof)
			}
		}

	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}
