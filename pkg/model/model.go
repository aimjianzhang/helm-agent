package model

import (
	"k8s.io/client-go/kubernetes"
	cmd_util "k8s.io/kubectl/pkg/cmd/util"
)

type Channel struct {
	CommandChannel chan *Packet
	ResponseChannel chan *Packet
}

func NewChannel(commandSize, responseSize int) *Channel {
	return &Channel{
		CommandChannel: make(chan *Packet, commandSize),
		ResponseChannel: make(chan *Packet, responseSize),
	}
}

type Packet struct {
	Key         string     `json:"key,omitempty"`
	Type        string     `json:"type,omitempty"`
	Payload     string     `json:"payload,omitempty"`
}

type Context struct {
	Client *kubernetes.Clientset
	CmdFactory cmd_util.Factory
	Channel *Channel
}
