package transaction

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/abelgalef/block/src/protofiles"
	"github.com/abelgalef/block/src/wallet"
)

type TransactionService struct {
	sync.Mutex
	Pending   []*pb.Transaction
	Wallet    *wallet.Wallet
	lastBlock [32]byte
}

func (ts *TransactionService) Send(amount float64, receiver string) {
	a_str := strconv.FormatFloat(amount, 'f', -1, 64)
	now := strconv.FormatInt(time.Now().Unix(), 10)
	hash := sha256.Sum256([]byte(a_str + ts.Wallet.PublicKeyString + receiver + now))
	sign, err := rsa.SignPKCS1v15(rand.Reader, ts.Wallet.PrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		panic(err)
	}

	t := &pb.Transaction{
		Amount:    amount,
		Sender:    ts.Wallet.PublicKeyString,
		Recipient: receiver,
		Timestamp: time.Now().Unix(),
		Signature: sign,
	}

	ts.Lock()
	defer ts.Unlock()

	if len(ts.Pending) >= 4 {
		// TODO: Send the pending transactions to the network
		ts.Pending = append(ts.Pending[:len(ts.Pending)], t)
	} else {
		ts.Pending = append(ts.Pending, t)
	}
}

func (ts *TransactionService) VerifyTransaction(t *pb.Transaction) bool {
	a_str := strconv.FormatFloat(t.Amount, 'f', -1, 64)
	hash := sha256.Sum256([]byte(a_str + t.Sender + t.Recipient + strconv.FormatInt(t.Timestamp, 10)))

	pubkey, err := wallet.ParseRsaPublicKeyFromPemStr(t.Sender)
	if err != nil {
		panic(err)
	}
	err = rsa.VerifyPKCS1v15(pubkey, crypto.SHA256, hash[:], t.Signature)

	return err == nil
}

func (ts *TransactionService) MakeBlock() *pb.Block {
	var txs []*pb.Transaction
	copy(txs, ts.Pending)
	var validTransactions []*pb.Transaction
	var signColl []byte

	// TRANSACTIONS NEED TO BE SORTED BY DATE SO THAT EVERY NODE GETS THE SAME RESULT
	sort.SliceStable(txs, func(i, j int) bool {
		return txs[i].Timestamp < txs[j].Timestamp
	})

	// ITERATE OVER THE TXS MANUALLY BECAUSE THE RANGE KEYWORD DOESN'T GUARANTEE ORDER
	for i := 0; i < len(txs); i++ {
		if ts.VerifyTransaction(txs[i]) {
			validTransactions = append(validTransactions, txs[i])
			signColl = append(signColl, txs[i].Signature...)
		}
	}

	var bits2hash []byte
	// CONCATENATES THE SIGNATURE BITS AND THE PREVIOUS BLOCK'S HASH BITS TOGETHER
	bits2hash = append(bits2hash, signColl...)
	var lbh [32]byte
	if ts.lastBlock != lbh {
		lbh = ts.lastBlock
	}
	bits2hash = append(bits2hash, lbh[:]...)
	blockHash := sha256.Sum256(bits2hash)

	nonce, hash := ts.MineBlock(blockHash)
	block := &pb.Block{
		Hash:         hash[:],
		Nonce:        nonce,
		Transactions: validTransactions,
		PrevHash:     lbh[:],
	}

	ts.lastBlock = hash
	return block
}

func (ts *TransactionService) MineBlock(blockHash [32]byte) (uint64, [32]byte) {
	var i uint64
	var hash [32]byte
	var bits []byte
	start := time.Now()
	bits = append(bits, blockHash[:]...)
	log.Println("Mining Block...")
	for {
		bits = append(bits, strconv.FormatUint(i, 10)...)
		hash = sha256.Sum256(bits)
		if strings.HasPrefix(base64.URLEncoding.EncodeToString(hash[:]), "00") {
			break
		}
		fmt.Printf("Got hash %s for NONCE %d\n", base64.URLEncoding.EncodeToString(hash[:]), i)
		i++
	}

	fmt.Printf("\nHash: %s\nNONCE: %d\nElapsed time %s", base64.URLEncoding.EncodeToString(hash[:]), i, time.Since(start))
	return i, hash
}
