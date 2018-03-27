package bc

type BlockHeader struct {
	Result struct {
		Bits              string `json:"bits"`
		Hash              string `json:"hash"`
		MerkleTreeHash    string `json:"merkle_tree_hash"`
		MixHash           string `json:"mixhash"`
		Nonce             string `json:"nonce"`
		Number            string `json:"number"`
		PreviousBlockHash string `json:"previous_block_hash"`
		TimeStamp         string `json:"time_stamp"`
		TransactionCount  string `json:"transaction_count"`
		Version           string `json:"version"`
	} `json:"result"`
}

type Input struct {
	PreviousOutput struct {
		Hash  string `json:"hash"`
		Index string `json:"index"`
	} `json:"previous_output"`
	Script   string `json:"script"`
	Sequence string `json:"sequence"`
}

type Output struct {
	Index      string `json:"index"`
	Address    string `json:"address"`
	Script     string `json:"script"`
	Value      string `json:"value"`
	Attachment struct {
		Type     string `json:"type"`
		Symbol   string `json:"symbol"`
		Quantity string `json:"quantity"`
	} `json:"attachment"`
}

type Transaction struct {
	Hash     string   `json:"hash"`
	Height   string   `json:"height"`
	Inputs   []Input  `json:"inputs"`
	LockTime string   `lock_time`
	Outputs  []Output `json:"outputs"`
	Version  string   `json:"version"`
}

type Block struct {
	Header BlockHeader `json:"header"`
	Txs    struct {
		Transactions []struct {
			Hash string `json:"hash"`
		} `json:"transactions"`
	} `json:"txs"`
}
