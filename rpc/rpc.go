package rpc

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	REQUEST_TIMEOUT = 30
)

type RpcClient struct {
	uri         string
	rpcuser     string
	rpcpassword string
	httpClient  *http.Client
}

type RpcReq struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type RpcRes struct {
	Id     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Err    interface{}     `json:"error"`
}

func New(host string, port int, user string, password string, useSSL bool) (cli *RpcClient, err error) {
	if len(host) == 0 {
		err = errors.New("missing argument host")
		return
	}
	var prefix string
	var httpClient *http.Client
	if useSSL {
		prefix = "https://"
		t := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = &http.Client{Transport: t}
	} else {
		prefix = "http://"
		httpClient = &http.Client{}
	}

	cli = &RpcClient{uri: fmt.Sprintf("%s%s:%d/rpc", prefix, host, port), rpcuser: user, rpcpassword: password, httpClient: httpClient}
	return
}

func (cli *RpcClient) Call(method string, params interface{}) (data []byte, err error) {
	rpcReq := RpcReq{method, params}

	buff := &bytes.Buffer{}
	err = json.NewEncoder(buff).Encode(rpcReq)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", cli.uri, buff)
	if err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("Accept", "application/json")
	if len(cli.rpcuser) > 0 || len(cli.rpcpassword) > 0 {
		req.SetBasicAuth(cli.rpcuser, cli.rpcpassword)
	}

	timer := time.NewTimer(REQUEST_TIMEOUT * time.Second)
	res, err := cli.doTimeoutRequest(timer, req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	data, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	if res.StatusCode != 200 {
		err = errors.New("Http error: " + res.Status)
		return
	}

	return
}

func (cli *RpcClient) doTimeoutRequest(timer *time.Timer, req *http.Request) (r *http.Response, err error) {
	type result struct {
		res *http.Response
		err error
	}

	done := make(chan result, 1)
	go func() {
		response, err := cli.httpClient.Do(req)
		done <- result{response, err}
	}()

	select {
	case r := <-done:
		return r.res, r.err
	case <-timer.C:
		return nil, errors.New("http client timeout...")
	}
}
