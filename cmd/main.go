package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/abelgalef/block/src/wallet"
)

type Transaction struct {
	Amount    float64
	Sender    string
	Receiver  string
	Date      string
	Signature []byte // Don't forget to zero this out when verifying
}

type TransactionSlice []*Transaction

type Block struct {
	Hash          [32]byte
	NUONCE        uint64
	Transactions  TransactionSlice
	PrevBlockHash [32]byte
}

var LastBlock *Block = nil

func main() {

	Chain := make(map[[32]byte]*Block)

	// CREATE TWO USERS
	w1 := wallet.NewWallet()
	w2 := wallet.NewWallet()

	// GENESIS BLOCK
	T1 := Send(100.0, w1, w2)
	T2 := Send(100.0, w1, w1)
	T3 := Send(90.0, w2, w2)
	T4 := Send(80.0, w2, w1)

	var txs TransactionSlice
	txs = append(txs, T1, T2, T3, T4)
	block := MakeBlock(txs)
	Chain[block.Hash] = block

	txs = nil
	T1 = Send(1.0, w1, w2)
	T2 = Send(30.0, w2, w1)
	T3 = Send(190.0, w2, w2)
	T4 = Send(800.0, w1, w1)

	txs = append(txs, T1, T2, T3, T4)
	block = MakeBlock(txs)
	Chain[block.Hash] = block
	for _, b := range Chain {
		fmt.Printf("\n%v\t%v\n", b.Hash, b.NUONCE)
		for _, t := range b.Transactions {
			fmt.Printf("\n%v\t%v\n", t.Sender, t.Amount)
			fmt.Printf("\n%+v %v\n", t.Receiver, t.Date)
			fmt.Printf("\n%+v\n", t.Signature)
		}
		fmt.Printf("%+v\n", b.PrevBlockHash)
	}

}

func MineBlock(blockHash [32]byte) (uint64, [32]byte) {
	var i uint64
	var hash [32]byte
	var bits []byte
	start := time.Now()
	fmt.Println("\t\t...Mining Started...")
	for {
		bits = nil
		bits = append(bits, blockHash[:]...)
		bits = append(bits, strconv.FormatUint(i, 10)...)
		hash = sha256.Sum256(bits)
		if strings.HasPrefix(base64.URLEncoding.EncodeToString(hash[:]), "00") {
			break
		}
		fmt.Printf("Got hash %s for NUONCE %d\n", base64.URLEncoding.EncodeToString(hash[:]), i)
		i++
	}

	fmt.Printf("\n\nHash: %s\nNUONCE: %d\nElapsed time %s", base64.URLEncoding.EncodeToString(hash[:]), i, time.Since(start))
	return i, hash
}

func Send(amount float64, sender *wallet.Wallet, receiver *wallet.Wallet) *Transaction {
	a_str := strconv.FormatFloat(amount, 'f', -1, 64)
	now := strconv.FormatInt(time.Now().Unix(), 10)
	hash := sha256.Sum256([]byte(a_str + sender.PublicKeyString + receiver.PublicKeyString + now))
	sign, err := rsa.SignPKCS1v15(rand.Reader, sender.PrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		panic(err)
	}
	return &Transaction{Amount: amount, Sender: sender.PublicKeyString, Receiver: receiver.PublicKeyString, Signature: sign, Date: now}
}

func VerifyTransaction(t *Transaction) bool {
	a_str := strconv.FormatFloat(t.Amount, 'f', -1, 64)
	hash := sha256.Sum256([]byte(a_str + t.Sender + t.Receiver + t.Date))

	pubkey, err := wallet.ParseRsaPublicKeyFromPemStr(t.Sender)
	if err != nil {
		panic(err)
	}
	err = rsa.VerifyPKCS1v15(pubkey, crypto.SHA256, hash[:], t.Signature)

	return err == nil
}

func MakeBlock(txs TransactionSlice) *Block {
	var validTransactions TransactionSlice
	var signColl []byte

	// TRANSACTIONS NEED TO BE SORTED BY DATE SO THAT EVERY NODE GETS THE SAME RESULT
	sort.Sort(validTransactions)

	// ITERATE OVER THE TXS MANUALLY BECAUSE THE RANGE KEYWORD DOESN'T GUARANTEE ORDER
	for i := 0; i < len(txs); i++ {
		if VerifyTransaction(txs[i]) {
			validTransactions = append(validTransactions, txs[i])
			signColl = append(signColl, txs[i].Signature...)
		}
	}

	//TODO MINE AND GET NONCE THEN HASH WITH TXS SIGNITURES AND PREVIOUS BLOCKS HASH

	var bits2hash []byte
	// CONCATENATES THE SIGNATURE BITS AND THE PREVIOUS BLOCK'S HASH BITS TOGETHER
	bits2hash = append(bits2hash, signColl...)
	var lbh [32]byte
	if LastBlock != nil {
		lbh = LastBlock.Hash
	}
	bits2hash = append(bits2hash, lbh[:]...)
	blockHash := sha256.Sum256(bits2hash)

	nounce, hash := MineBlock(blockHash)
	block := &Block{
		Hash:          hash,
		NUONCE:        nounce,
		Transactions:  validTransactions,
		PrevBlockHash: lbh,
	}

	LastBlock = block
	return block
}

func (t TransactionSlice) Len() int      { return len(t) }
func (t TransactionSlice) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t TransactionSlice) Less(i, j int) bool {
	date1, err := time.Parse(time.RFC3339, t[i].Date)
	if err != nil {
		panic("sort transactions: incorrect time given: " + err.Error())
	}

	date2, err := time.Parse(time.RFC3339, t[j].Date)
	if err != nil {
		panic("sort transactions: incorrect time given: " + err.Error())
	}
	return date1.Before(date2)
}
