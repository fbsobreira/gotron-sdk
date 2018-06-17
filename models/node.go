package models

import (
	"github.com/sasaxie/go-client-api/common/global"
)

type Node struct {
	Address Address
}

type Address struct {
	Host string
	Port int32
}

func GetNodeList() []Node {
	nodes := make([]Node, 0)

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

		nodes = append(nodes, node)
	}

	return nodes
}
