package syncmgr

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"mvs_sync/bc"
	"mvs_sync/db"
	"time"
)

var log = log15.New()

type SyncMgr struct {
	BCClient       *bc.BcClient
	DBClient       *db.DbClient
	BeginFrom      int64
	EndTo          int64
	GoroutineCount int
	ChanHeightNull chan int64
	ChanHeightSync chan int64
	IsRealTimeSync bool
}

var (
	instance   *SyncMgr = nil
	syncheight int64    = -1
)

func GetInstance() *SyncMgr {
	if instance == nil {
		instance = &SyncMgr{}
	}
	return instance
}

func (mgr *SyncMgr) Init(bcclient *bc.BcClient, dbclient *db.DbClient, beginfrom int64, endto int64, goroutinecount int) (err error) {
	mgr.BeginFrom, err = GetBeginFrom(dbclient, beginfrom)
	if err != nil {
		log.Info("Init GetBeginFrom", "Error", err.Error())
		return
	}

	mgr.EndTo, err = GetEndTo(bcclient, endto)
	if err != nil {
		log.Info("Init GetEndTo", "Error", err.Error())
		return
	}

	log.Debug("Mission init", "BeginFrom", mgr.BeginFrom, "EndTo", mgr.EndTo)

	if beginfrom <= 0 && endto <= 0 {
		mgr.IsRealTimeSync = true
	}

	mgr.BCClient = bcclient
	mgr.DBClient = dbclient
	mgr.GoroutineCount = goroutinecount
	mgr.ChanHeightNull = make(chan int64)
	mgr.ChanHeightSync = make(chan int64)

	return nil
}

func GetBeginFrom(dbclient *db.DbClient, default1 int64) (height int64, err error) {
	if default1 > 0 {
		height = default1
	} else {
		height, err = dbclient.GetLocalHeight()
		if err != nil {
			return 0, err
		}

		if height < 0 {
			return 0, err
		}
	}

	return height, nil
}

func GetEndTo(bcclient *bc.BcClient, default1 int64) (height int64, err error) {
	if default1 > 0 {
		height = default1
	} else {
		height, err = bcclient.GetBlockCount()
		if err != nil {
			return 0, err
		}
	}

	return height, nil
}

func (mgr *SyncMgr) Start() {
	fmt.Println("Start...")
	time.Sleep(time.Second * 1)

	// 1. init blocks, then sleep for a while
	for i := 0; i <= 1; i++ {
		go mgr.initBlocks()
	}
	for j := mgr.BeginFrom; j <= mgr.EndTo; j++ {
		mgr.ChanHeightNull <- j
	}
	time.Sleep(time.Second * 1)

	// 2. sync: insert into channel
	for m := 0; m <= mgr.GoroutineCount; m++ {
		go mgr.syncBlocks()
	}
	for syncheight := mgr.BeginFrom; syncheight <= mgr.EndTo; syncheight++ {
		mgr.ChanHeightSync <- syncheight
	}
	time.Sleep(time.Second * 60)

	// 3. find missing blocks & transactions
	go mgr.syncBlocksInMissing()
	go mgr.syncTransactionsInMissing()

	// 4. sync: realtime & nextblockhash
	go mgr.syncBlocksInRealTime()
}

func (mgr *SyncMgr) initBlocks() {
	defer func() {
		if err := recover(); err != nil {
			log.Info("initBlocks", "Error", err)
		}
	}()

	for {
		select {
		case h := <-mgr.ChanHeightNull:
			for i := 0; i < 10; i++ {
				err := mgr.DBClient.InsertBlockNull(h, 0)
				if err != nil {
					continue
				}
				break

			}
		}
	}
}

func (mgr *SyncMgr) syncBlocks() {
	defer func() {
		if err := recover(); err != nil {
			log.Info("syncBlocks", "Error", err)
		}
	}()

	for {
		select {
		case h := <-mgr.ChanHeightSync:
			var blockUpdated int = 1

			for i := 0; i < 10; i++ {
				var b bc.Block
				var transactionUpdated int

				blockUpdated = 1

				hash, err := mgr.BCClient.GetBlockHash(h)
				if err != nil {
					log.Info("GetBlockHash", "Error", err)
					blockUpdated = 0
				}

				b, err = mgr.BCClient.GetBlock(hash)
				if err != nil {
					log.Info("GetBlock", "Error", err)
					blockUpdated = 0
				}

				if blockUpdated == 1 {
					var transactions []*bc.Transaction
					var transactions_updated []int

					for _, tx := range b.Txs.Transactions {
						var t bc.Transaction
						t, err = mgr.BCClient.GetRawTransaction1(tx.Hash)
						if err != nil {
							log.Info("GetRawTransaction", "Error", err)
							transactionUpdated = 0
							t = bc.Transaction{Hash: tx.Hash}
						} else {
							transactionUpdated = 1
							// t, _ = rawT.(bc.Transaction) // transfer rawT(interface) to bc.Transaction
							// //fmt.Printf("%#v\n", t)
						}

						transactions = append(transactions, &t)
						transactions_updated = append(transactions_updated, transactionUpdated)
					}

					for {
						err := mgr.DBClient.UpdateBlockAndTransactions(&b, transactions, transactions_updated)
						if err != nil {
							continue
						}
						break
					}

					break
				}
			}
		}
	}
}

func (mgr *SyncMgr) syncBlocksInMissing() {
	defer func() {
		if err := recover(); err != nil {
			log.Info("syncBlocksInMissing", "Error", err)
		}
	}()

	for {
		Ids, err := mgr.DBClient.SelectBlockNull()
		if err != nil {
			continue
		}

		for _, h := range Ids {
			var blockUpdated int = 1
			var b bc.Block
			var transactionUpdated int

			hash, err := mgr.BCClient.GetBlockHash(h)
			if err != nil {
				log.Info("GetBlockHash", "Error", err)
				blockUpdated = 0
			}

			b, err = mgr.BCClient.GetBlock(hash)
			if err != nil {
				log.Info("GetBlock", "Error", err)
				blockUpdated = 0
			}

			if blockUpdated == 1 {
				var transactions []*bc.Transaction
				var transactions_updated []int

				for _, tx := range b.Txs.Transactions {
					if tx.Hash == "" {
						continue
					}
					var t bc.Transaction
					t, err = mgr.BCClient.GetRawTransaction1(tx.Hash)
					if err != nil {
						log.Info("GetRawTransaction", "Error", err)
						transactionUpdated = 0
						t = bc.Transaction{Hash: tx.Hash}
					} else {
						transactionUpdated = 1
						// t, _ = rawT.(bc.Transaction) // transfer rawT(interface) to bc.Transaction
						// //fmt.Printf("%#v\n", t)
					}

					transactions = append(transactions, &t)
					transactions_updated = append(transactions_updated, transactionUpdated)
				}

				for {
					err := mgr.DBClient.UpdateBlockAndTransactions(&b, transactions, transactions_updated)
					if err != nil {
						fmt.Printf("%#v\n", err.Error())
						continue
					}
					break
				}

				break
			}
		}
	}
}

func (mgr *SyncMgr) syncTransactionsInMissing() {
	defer func() {
		if err := recover(); err != nil {
			log.Info("syncTransactionsInMissing", "Error", err)
		}
	}()

	for {
		hasharr, err := mgr.DBClient.SelectTransactionNull()
		if err != nil {
			continue
		}

		for _, tx := range hasharr {
			//var rawT interface{}
			var err error
			var t bc.Transaction

			for i := 0; i < 10; i++ {
				t, err = mgr.BCClient.GetRawTransaction1(tx)
				if err != nil {
					log.Info("GetRawTransaction", "Error", err)
					break
				}
			}

			if err != nil {
				mgr.DBClient.GiveupTransaction(tx, 1)
			} else {
				//t, _ := rawT.(bc.Transaction)
				mgr.DBClient.UpdateTransaction(&t)
			}
		}
	}
}

func (mgr *SyncMgr) syncBlocksInRealTime() {
	defer func() {
		if err := recover(); err != nil {
			log.Info("syncBlocksInRealTime", "Error", err)
		}
	}()

	if mgr.IsRealTimeSync {
		for {
			beginfrom, err := GetBeginFrom(mgr.DBClient, 0)
			if err != nil {
				continue
			}

			endto, err := GetEndTo(mgr.BCClient, 0)
			if err != nil {
				continue
			}

			if beginfrom == 0 || beginfrom < syncheight || endto <= syncheight {
				time.Sleep(time.Second * 10)
				continue
			}

			for i := beginfrom; i <= endto; i++ {
				mgr.ChanHeightNull <- i
			}
		}
	}
}
