package models

import (
	"github.com/sasaxie/go-client-api/common/global"
)

type NodeList struct {
	Nodes []Node
}

type Node struct {
	Address Address
}

type Address struct {
	Host string
	Port int32
}

func GetNodeList() NodeList {
	var nodes NodeList
	nodes.Nodes = make([]Node, 0)

	grpcNodes := global.TronClient.ListNodes()

	if grpcNodes == nil {
		return nodes
	}

	for _, n := range grpcNodes.Nodes {
		var node Node
		var address Address

		if n.Address != nil {
			address.Host = string(n.Address.Host)
			address.Port = n.Address.Port
		}

		node.Address = address

		nodes.Nodes = append(nodes.Nodes, node)
	}

	return nodes
}
