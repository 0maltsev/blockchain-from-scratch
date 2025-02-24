package network

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/go-kit/log"

	"github.com/0maltsev/blockchain-from-scratch/core"
	"github.com/0maltsev/blockchain-from-scratch/crypto"
	"github.com/sirupsen/logrus"
)

var defaultBlockTime = 5 * time.Second

type ServerOpts struct{
	ID		string
	Logger	log.Logger
	RPCDecodeFunc	RPCDecodeFunc
	RPCProcessor	RPCProcessor
	Transports 	[]Transport
	BlockTime	time.Duration
	PrivateKey	*crypto.PrivateKey
}

type Server struct {
	ServerOpts
	memPool		*TxPool
	isValidator bool
	rpcCh 		chan RPC
	quitCh 		chan struct{}
}

func NewServer(opts ServerOpts) *Server {
	if opts.BlockTime == time.Duration(0) {
		opts.BlockTime = defaultBlockTime
	}

	if opts.RPCDecodeFunc == nil {
		opts.RPCDecodeFunc = DefaultRPCDecodeFunc
	}
	if opts.Logger == nil {
		opts.Logger = log.NewLogfmtLogger(os.Stderr)
		opts.Logger = log.With(opts.Logger, "ID", opts.ID)
	}

	s := &Server{
		ServerOpts: opts,
		memPool: NewTxPool(),
		isValidator: opts.PrivateKey != nil,
		rpcCh: make(chan RPC),
		quitCh: make(chan struct{}, 1),
	}

	if s.RPCProcessor == nil {
		s.RPCProcessor = s
	}

	if s.isValidator {
		go s.validatorLoop()
	}

	return s
}

func (s *Server) Start(){
	s.initTransports()

free:
	for {
		select{
		case rpc := <- s.rpcCh:
			msg, err := s.RPCDecodeFunc(rpc)
			if err != nil {
				s.Logger.Log("error", err)
			}

			if err := s.RPCProcessor.ProcessMessage(msg); err != nil {
				s.Logger.Log("error", err)
			}

		case <-s.quitCh:
			break free
		}
	}
	s.Logger.Log("msg", "Server shutdown")
}

func (s *Server) validatorLoop() {
	ticker := time.NewTicker(s.BlockTime)

	s.Logger.Log("msg", "Starting validator loop", "blocktime", s.BlockTime)

	for {
		<-ticker.C
		s.createNewBlock()
	}
}

func (s *Server) ProcessMessage(msg *DecodedMessage) error {

	switch t := msg.Data.(type) {
	case *core.Transaction:
		return s.processTransaction(t)
	}
	return nil
}

func (s *Server) broadcast(payload []byte) error {
	for _, tr := range s.Transports {
		if err := tr.Broadcast(payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) processTransaction (tx *core.Transaction) error {
	hash := tx.Hash(core.TxHasher{})

	if s.memPool.Has(hash) {
		return nil
	}

	if err := tx.Verify(); err != nil {
		return err
	}

	tx.SetFirstSeen(time.Now().UnixNano())

	logrus.WithFields(logrus.Fields{
		"hash": hash,
		"mempool length": s.memPool.Len(),
	}).Info("adding new tx to the mempool")
	
	s.Logger.Log("msg", "adding new tx to mempool", 
		"hash", hash,
		"mempoolLength", s.memPool.Len())

	go s.broadcastTx(tx)


	return s.memPool.Add(tx)
}

func (s *Server) broadcastTx(tx *core.Transaction) error {
	buf := &bytes.Buffer{}
	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeTx, buf.Bytes())
	return s.broadcast(msg.Bytes())
}

func (s *Server) createNewBlock() error {
	fmt.Println("creating a new block")
	return nil
}

func (s *Server) initTransports(){
	for _, tr := range s.Transports{
		go func(tr Transport) {
			for rpc := range tr.Consume(){
				s.rpcCh <- rpc
			}
		}(tr)
	}
}
