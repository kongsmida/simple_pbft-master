package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kongsmida/simple_pbft-master/pbft/consensus"
)

type Server struct {
	url  string
	node *Node
}

func NewServer(nodeID string) *Server {
	node := NewNode(nodeID)
	var prex string = "localhost:"
	url := prex + nodeID
	server := &Server{url, node}

	server.setRoute()

	return server
}

func (server *Server) Start() {
	fmt.Printf("Server will be started at %s...\n", server.url)
	if err := http.ListenAndServe(server.url, nil); err != nil {
		fmt.Println(err)
		return
	}
}

func (server *Server) setRoute() {
	http.HandleFunc("/as", server.getAS)
	http.HandleFunc("/asf", server.getASF)
	http.HandleFunc("/asF", server.getASFormation)

	http.HandleFunc("/req", server.getReq)
	http.HandleFunc("/preprepare", server.getPrePrepare)
	http.HandleFunc("/prepare", server.getPrepare)
	http.HandleFunc("/commit", server.getCommit)

	//client --> as
	http.HandleFunc("/croreq", server.getCrossReq)
	//as --> as
	http.HandleFunc("/propose", server.getPropose)
	//as --> as
	http.HandleFunc("/precommit", server.getPreCommit)
	//as --> backups
	http.HandleFunc("/synchronize", server.getSyn)
	//as --> client
	http.HandleFunc("/reply", server.getReply)

}

var T_start_req int64
var T_start_croreq int64
var T_start_as int64

func (server *Server) getAS(writer http.ResponseWriter, request *http.Request) {
	T_start_as = time.Now().UnixNano()
	fmt.Println("Start to AS formation ------", T_start_as)
	server.node.GetAS()
}

func (server *Server) getASF(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.ProofMsgs
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println("error causes:", err)
		return
	}
	fmt.Println("Primary starts to generate identify", msg)
	server.node.MsgEntrance <- &msg
}

func (server *Server) getASFormation(writer http.ResponseWriter, request *http.Request) {
	fmt.Println("get aggregated sig!")
	var msg consensus.ProofMsgs
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println("error causes:", err)
		return
	}
	server.node.MsgEntrance <- &msg
}

func (server *Server) getReq(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.SingletxMsg
	T_start_req = time.Now().UnixNano()
	fmt.Println("getRequest success!")
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println("error causes:", err)
		return
	}
	server.node.MsgEntrance <- &msg
}

func (server *Server) getCrossReq(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.CrosstxMsg
	T_start_croreq = time.Now().UnixNano()
	fmt.Println("getCroRequest success!")
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println("error causes:", err)
		return
	}
	server.node.MsgEntrance <- &msg
}

func (server *Server) getPrePrepare(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.PrePrepareMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.node.MsgEntrance <- &msg
}

func (server *Server) getPrepare(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.VoteMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.node.MsgEntrance <- &msg
}

func (server *Server) getPropose(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.ProposeMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.node.MsgEntrance <- &msg
}

func (server *Server) getPreCommit(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.ProofMsgs
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.node.MsgEntrance <- &msg
}

func (server *Server) getSyn(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.SynchronizeMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.node.MsgEntrance <- &msg
}

func (server *Server) getCommit(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.VoteMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.node.MsgEntrance <- &msg
}

func (server *Server) getReply(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.ReplyMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.node.GetReply(&msg)
}

func send(url string, msg []byte) {
	buff := bytes.NewBuffer(msg)
	http.Post("http://"+url, "application/json", buff)
}
