package bc

import (
	"encoding/json"
	//"fmt"
	"github.com/inconshreveable/log15"
	"mvs_sync/config"
	"mvs_sync/rpc"
	"strconv"
)

var log = log15.New()

type BcClient struct {
	rpc *rpc.RpcClient
}

func New(host string, port int, user string, password string, useSSL bool) (*BcClient, error) {
	rpc, err := rpc.New(host, port, user, password, useSSL)
	if err != nil {
		return nil, err
	}

	return &BcClient{rpc}, nil
}

func (cli *BcClient) GetBlockCount() (height int64, err error) {
	var params []interface{}
	params = append(params, "")

	data, err := cli.rpc.Call("getbestblockheader", params)
	if err != nil {
		log.Info("getbestblockheader", "Error", err)
		return
	}

	var r BlockHeader
	err = json.Unmarshal(data, &r)
	if err != nil {
		log.Info("getbestblockheader json Unmarshal", "Error", err)
		return
	}

	height, err = strconv.ParseInt(r.Result.Number, 10, 64)
	if err != nil {
		log.Info("getbestblockheader strconv", "Error", err)
		return
	}

	return
}

func (cli *BcClient) GetBlockHash(height int64) (hash string, err error) {
	var params []interface{}
	rpcuser := config.Get("mvsd", "rpcuser")
	rpcpassword := config.Get("mvsd", "rpcpassword")
	params = append(params, rpcuser)
	params = append(params, rpcpassword)
	params = append(params, height)

	data, err := cli.rpc.Call("fetchheaderext", params)
	if err != nil {
		log.Info("fetchheaderext", "Error", err)
		return
	}

	var r BlockHeader
	err = json.Unmarshal(data, &r)
	if err != nil {
		log.Info("fetchheaderext json Unmarshal", "Error", err)
		return
	}

	hash = r.Result.Hash

	return
}

func (cli *BcClient) GetBlock(hash string) (block Block, err error) {
	var params []interface{}
	params = append(params, hash)
	params = append(params, "true")

	data, err := cli.rpc.Call("getblock", params)
	if err != nil {
		log.Info("getblock", "Error", err)
		return
	}

	err = json.Unmarshal(data, &block)
	if err != nil {
		log.Info("getblock json Unmarshal", "Error", err)
		return
	}

	return
}

func (cli *BcClient) GetRawTransaction(txid string) (rawT interface{}, err error) {
	var params []interface{}
	params = append(params, txid)

	data, err := cli.rpc.Call("gettransaction", params)
	if err != nil {
		log.Info("gettransaction", "Error", err)
		return
	}

	err = json.Unmarshal(data, &rawT)
	if err != nil {
		log.Info("gettransaction json Unmarshal", "Error", err)
		return
	}

	return
}

func (cli *BcClient) GetRawTransaction1(txid string) (t Transaction, err error) {
	var params []interface{}
	params = append(params, txid)

	data, err := cli.rpc.Call("gettransaction", params)
	if err != nil {
		log.Info("gettransaction", "Error", err)
		return
	}

	err = json.Unmarshal(data, &t)
	if err != nil {
		log.Info("gettransaction json Unmarshal", "Error", err)
		return
	}

	return
}
