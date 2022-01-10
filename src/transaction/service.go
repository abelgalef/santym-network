package transaction

import pb "github.com/abelgalef/block/src/protofiles/"

type TransactionServcie struct {
	Pending []*pb.Transaction
}
