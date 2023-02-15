package network

import (
	"fmt"
	"io"
	"strconv"

	"os"

	"github.com/kongsmida/simple_pbft-master/pbft/consensus"
)

func LogMsg(msg interface{}) {
	switch msg.(type) {
	case *consensus.SingletxMsg:
		reqMsg := msg.(*consensus.SingletxMsg)
		fmt.Printf("[SINGLETX] ClientID: %s, Timestamp: %d, Operation: %s\n", reqMsg.ClientID, reqMsg.Timestamp, reqMsg.Operation)
	case *consensus.CrosstxMsg:
		crosstxMsg := msg.(*consensus.CrosstxMsg)
		tx := crosstxMsg.Operation
		fmt.Printf("[CROSSTX] ClientID: %s, Timestamp: %d, Operation: ", crosstxMsg.ClientID, crosstxMsg.Timestamp)
		fmt.Println(tx)
	case *consensus.ProposeMsg:
		ProposeMsg := msg.(*consensus.ProposeMsg)
		subtx := ProposeMsg.Subtx
		fmt.Printf("[PROPOSE] SubTx: ")
		fmt.Print(subtx)
		fmt.Printf(", PrimaryID: %s\n", ProposeMsg.NodeID)

	case *consensus.PrePrepareMsg:
		prePrepareMsg := msg.(*consensus.PrePrepareMsg)
		fmt.Printf("[PREPREPARE] ClientID: %s, Operation: %s, SequenceID: %d\n", prePrepareMsg.RequestMsg.ClientID, prePrepareMsg.RequestMsg.Operation, prePrepareMsg.SequenceID)
	case *consensus.PreReplyMsg:
		preReplyMsg := msg.(*consensus.PreReplyMsg)
		fmt.Printf("[PREREPLY] SubTx: %s, NodeID: %s\n", preReplyMsg.Subtx, preReplyMsg.NodeID)
	case *consensus.VoteMsg:
		voteMsg := msg.(*consensus.VoteMsg)
		if voteMsg.MsgType == consensus.PreCommitMsg {
			fmt.Printf("[PRECOMMIT] NodeID: %s\n", voteMsg.NodeID)
		} else if voteMsg.MsgType == consensus.CommitMsg {
			fmt.Printf("[COMMIT] NodeID: %s\n", voteMsg.NodeID)
		} else if voteMsg.MsgType == consensus.PrepareMsg {
			fmt.Printf("[PREPARE] NodeID: %s\n", voteMsg.NodeID)
		}
	default:
		fmt.Println("no match txtype")
	}
}

func LogStage(stage string, isDone bool) {
	if isDone {
		fmt.Printf("[STAGE-DONE] %s\n", stage)
	} else {
		fmt.Printf("[STAGE-BEGIN] %s\n", stage)
	}
}
func checkFile(Filename string) bool {
	var exist = true
	if _, err := os.Stat(Filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

var i int = 0

// 写入文件
func Logfile(Log string, msgtype string) {
	i = i + 1
	var f1 *os.File
	var err1 error

	Filenames := "./" + msgtype + "_result.log" //也可将name作为参数传进来

	fmt.Println(Filenames)
	if checkFile(Filenames) { //如果文件存在
		f1, err1 = os.OpenFile(Filenames, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666) //打开文件,第二个参数是写入方式和权限
		if err1 != nil {
			fmt.Println(err1)
		}
	} else {
		f1, err1 = os.Create(Filenames) //创建文件
		if err1 != nil {
			fmt.Println("创建文件失败")
			fmt.Println(err1)
		}
	}
	_, err1 = io.WriteString(f1, strconv.Itoa(i)+"\t"+Log+"\n") //写入文件(字符串)
	if err1 != nil {
		fmt.Println(err1)
	}
	//fmt.Printf("写入 %d 个字节\n", n)

}
