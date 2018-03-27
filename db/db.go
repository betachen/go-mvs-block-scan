package db

import (
	"database/sql"
	//"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/inconshreveable/log15"
	"mvs_sync/bc"
	"mvs_sync/config"
	"strconv"
	"strings"
	"time"
)

var log = log15.New()

type DbClient struct {
	conn *sql.DB
}

var (
	instance *DbClient = nil
)

const bornDate = "2006-01-02 15:04:05"

func GetInstance() *DbClient {
	if instance == nil {
		instance = &DbClient{}
	}
	return instance
}

func (cli *DbClient) Open() (err error) {
	uri := config.Get("db", "uri")
	cli.conn, err = sql.Open("mysql", uri)
	if err == nil {
		log.Info("Open database successed")
	}
	return err
}

func (cli *DbClient) Close() {
	if cli.conn != nil {
		cli.conn.Close()
	}
}

func (cli *DbClient) GetLocalHeight() (height int64, err error) {
	var id int64 = -1
	err = cli.conn.QueryRow("SELECT id FROM bc_etp_blocks").Scan(&id)
	if id == -1 {
		return 0, nil
	}

	// err = cli.conn.QueryRow("SELECT MAX(id) as maxId FROM bc_etp_blocks").Scan(&height)

	// return height + 1, err

	err = cli.conn.QueryRow("SELECT id FROM bc_etp_blocks WHERE hash IS NULL ORDER BY id ASC LIMIT 1").Scan(&id)
	if err != nil {
		err = cli.conn.QueryRow("SELECT MAX(id) FROM bc_etp_blocks").Scan(&id)
		if err != nil {
			return 0, nil
		}
	}
	if id <= 0 {
		id = 0
	}
	height = id
	return height, err
}

func (cli *DbClient) InsertBlockNull(height int64, updated int) error {

	statment, err := cli.conn.Prepare("INSERT bc_etp_blocks SET id=?, updated=?")
	if err != nil {
		log.Info("InsertBlockNull statment", "id", height, "Error", err)
		return err
	}

	_, err = statment.Exec(height, updated)
	if err != nil {
		if strings.Index(err.Error(), "Error 1062") > -1 {
			//log.Info("Insert null block into database", "Error", 1062, "id", height)
			return err
		} else {
			log.Info("InsertBlockNull", "id", height, "Error", err)
			return err
		}
	}
	defer statment.Close()

	log.Debug("Insert null block into database", "id", height)

	return nil
}

func (cli *DbClient) UpdateBlockAndTransactions(b *bc.Block, transactions []*bc.Transaction, transactions_updated []int) error {
	tx, err := cli.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statment1, err := tx.Prepare(`UPDATE bc_etp_blocks SET hash=?, confirmations=?, strippedsize=?, size=?, weight=?, height=?, version=?, merkleroot=?, time=?, mediantime=?, nonce=?, bits=?, difficulty=?, chainwork=?, previousblockhash=?, nextblockhash=?, tx_count=?, updated=? WHERE id=?`)
	if err != nil {
		return err
	}
	defer statment1.Close()

	statment2, err := tx.Prepare(`INSERT bc_etp_transactions SET hash=?, confirmations=?, block_hash=?, lock_time=?, time=?, blocktime=?, version=?, updated=?`)
	if err != nil {
		return err
	}
	defer statment2.Close()

	statment3, err := tx.Prepare(`INSERT bc_etp_inputs SET tx_hash=?, previous_output_hash=?, previous_output_index=?, sequence=?`)
	if err != nil {
		return err
	}
	defer statment3.Close()

	statment4, err := tx.Prepare(`INSERT bc_etp_outputs SET tx_hash=?, index1=?, address=?, type=?, value=?, attachment_type=?, quantity=?`)
	if err != nil {
		return err
	}
	defer statment4.Close()

	statment5, err := tx.Prepare(`INSERT bc_etp_transactions SET hash=?, updated=?`)
	if err != nil {
		return err
	}
	defer statment5.Close()

	time1, err := strconv.ParseInt(b.Header.Result.TimeStamp, 10, 64)
	timestamp1 := time.Unix(time1, 0).Format(bornDate)
	nonce1, err := strconv.ParseInt(b.Header.Result.Nonce, 10, 64)

	_, err = statment1.Exec(b.Header.Result.Hash, 0, 0, 0, 0, b.Header.Result.Number, b.Header.Result.Version, b.Header.Result.MerkleTreeHash, timestamp1, nil, nonce1, b.Header.Result.Bits, 0, "", b.Header.Result.PreviousBlockHash, "", len(transactions), 1, b.Header.Result.Number)
	if err != nil {
		return err
	}

	for i, t := range transactions {
		if transactions_updated[i] == 0 {
			_, err = statment5.Exec(t.Hash, transactions_updated[i])
			if err != nil {
				return err
			}
		} else {
			locktime2, err := strconv.ParseInt(t.LockTime, 10, 64)
			version2, err := strconv.ParseInt(t.Version, 10, 32)

			_, err = statment2.Exec(t.Hash, 0, b.Header.Result.Hash, locktime2, nil, nil, version2, transactions_updated[i])
			if err != nil {
				return err
			}

			for _, v := range t.Inputs {
				_, err = statment3.Exec(t.Hash, v.PreviousOutput.Hash, v.PreviousOutput.Index, v.Sequence)
				if err != nil {
					return err
				}
			}

			for _, v := range t.Outputs {
				log.Info("UpdateBlockAndTransactions", "index", v.Index, "address", v.Address)
				value4, _ := strconv.ParseInt(v.Value, 10, 64)
				quantity4, _ := strconv.ParseInt(v.Attachment.Quantity, 10, 64)

				if len(v.Attachment.Symbol) > 0 {
					_, err = statment4.Exec(t.Hash, v.Index, v.Address, v.Attachment.Symbol, value4, "ETP", quantity4)
					if err != nil {
						log.Info("UpdateBlockAndTransactions", "Error", err)
					}
				} else {
					_, err = statment4.Exec(t.Hash, v.Index, v.Address, v.Attachment.Type, value4, "ETP", 0)
					if err != nil {
						log.Info("UpdateBlockAndTransactions", "Error", err)
					}
				}
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	log.Debug("Insert real block into database", "id", b.Header.Result.Number, "hash", b.Header.Result.Hash, "tx_count", len(transactions))
	return nil
}

func (cli *DbClient) SelectBlockNull() ([]int64, error) {
	var Ids []int64
	rows, err := cli.conn.Query("SELECT id FROM bc_etp_blocks WHERE updated=0 LIMIT 64")
	if err != nil {
		return Ids, err
	}

	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return Ids, err
		}
		Ids = append(Ids, id)
	}

	if err = rows.Err(); err != nil {
		return Ids, err
	}

	return Ids, nil
}

func (cli *DbClient) SelectTransactionNull() ([]string, error) {
	var hasharr []string
	rows, err := cli.conn.Query("SELECT hash FROM bc_etp_transactions WHERE updated=0 LIMIT 1024")
	if err != nil {
		return hasharr, err
	}

	for rows.Next() {
		var hash string
		if err = rows.Scan(&hash); err != nil {
			return hasharr, err
		}
		hasharr = append(hasharr, hash)
	}

	if err = rows.Err(); err != nil {
		return hasharr, err
	}

	return hasharr, nil
}

func (cli *DbClient) GiveupTransaction(hash string, updated int) error {
	statment1, err := cli.conn.Prepare(`UPDATE bc_etp_transactions SET updated=? WHERE hash=?`)
	if err != nil {
		return err
	}
	defer statment1.Close()

	_, err = statment1.Exec(1, hash)
	if err != nil {
		return err
	}

	log.Info("Giveup transaction", "hash", hash, "updated", updated)

	return nil
}

func (cli *DbClient) UpdateTransaction(t *bc.Transaction) error {
	tx, err := cli.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statment2, err := tx.Prepare(`UPDATE bc_etp_transactions SET block_hash=?, lock_time=?, version=?, updated=? WHERE hash=?`)
	if err != nil {
		return err
	}
	defer statment2.Close()

	statment3, err := tx.Prepare(`INSERT bc_etp_inputs SET tx_hash=?, previous_output_hash=?, previous_output_index=?, sequence=?`)
	if err != nil {
		return err
	}
	defer statment3.Close()

	statment4, err := tx.Prepare(`INSERT bc_etp_outputs SET tx_hash=?, index1=?, address=?, type=?, value=?, attachment_type=?, quantity=?`)
	if err != nil {
		return err
	}
	defer statment4.Close()

	locktime2, err := strconv.ParseInt(t.LockTime, 10, 64)
	version2, err := strconv.ParseInt(t.Version, 10, 32)

	_, err = statment2.Exec("", locktime2, version2, 1, t.Hash)
	if err != nil {
		return err
	}

	for _, v := range t.Inputs {
		_, err = statment3.Exec(t.Hash, v.PreviousOutput.Hash, v.PreviousOutput.Index, v.Sequence)
		if err != nil {
			return err
		}
	}

	for _, v := range t.Outputs {
		log.Info("UpdateTransaction", "index", v.Index, "address", v.Address)
		value4, _ := strconv.ParseInt(v.Value, 10, 64)
		quantity4, _ := strconv.ParseInt(v.Attachment.Quantity, 10, 64)

		if len(v.Attachment.Symbol) > 0 {
			_, err = statment4.Exec(t.Hash, v.Index, v.Address, v.Attachment.Symbol, value4, "ETP", quantity4)
			if err != nil {
				log.Info("UpdateTransaction", "Error", err)
			}
		} else {
			_, err = statment4.Exec(t.Hash, v.Index, v.Address, v.Attachment.Type, value4, "ETP", 0)
			if err != nil {
				log.Info("UpdateTransaction", "Error", err)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	log.Debug("Update transaction", "hash", t.Hash, "updated", 1)

	return nil
}
