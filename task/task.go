package task

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/rsol-relay/pkg/config"
	"github.com/stafiprotocol/rsol-relay/pkg/utils"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/common"
	"github.com/stafiprotocol/solana-go-sdk/rsolprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
)

var stakePoolSeed = []byte("pool_seed")
var mintAuthoritySeed = []byte("mint")

type Task struct {
	stop        chan struct{}
	cfg         config.ConfigStart
	accountsMap map[string]types.Account

	rSolProgramID   common.PublicKey
	minterProgramID common.PublicKey
	stakeManager    common.PublicKey
	mintManager     common.PublicKey
	rSolMint        common.PublicKey
	feeRecipient    common.PublicKey
	stakePool       common.PublicKey
	mintAuthority   common.PublicKey

	feePayerAccount types.Account

	client   *client.Client
	handlers []Handler
}

type Handler struct {
	method func() error
	name   string
}

func NewTask(cfg config.ConfigStart, accouts map[string]types.Account) *Task {
	s := &Task{
		stop:        make(chan struct{}),
		cfg:         cfg,
		accountsMap: accouts,
	}
	return s
}

func (task *Task) Start() error {

	rSolProgramID := common.PublicKeyFromString(task.cfg.RSolProgramID)
	minterProgramID := common.PublicKeyFromString(task.cfg.MinterProgramID)
	stakeManager := common.PublicKeyFromString(task.cfg.StakeManagerAddress)
	mintManager := common.PublicKeyFromString(task.cfg.MintManagerAddress)
	rSolMint := common.PublicKeyFromString(task.cfg.RSolMintAddress)
	feeRecipient := common.PublicKeyFromString(task.cfg.FeeRecipientAddress)

	stakePool, _, err := common.FindProgramAddress([][]byte{stakeManager.Bytes(), stakePoolSeed}, rSolProgramID)
	if err != nil {
		return err
	}
	mintAuthority, _, err := common.FindProgramAddress([][]byte{mintManager.Bytes(), mintAuthoritySeed}, minterProgramID)
	if err != nil {
		return err
	}

	feePayerAccount, exist := task.accountsMap[task.cfg.FeePayerAccount]
	if !exist {
		return fmt.Errorf("fee payer not exit in vault")
	}

	task.rSolProgramID = rSolProgramID
	task.minterProgramID = minterProgramID
	task.stakeManager = stakeManager
	task.mintManager = mintManager
	task.rSolMint = rSolMint
	task.feeRecipient = feeRecipient
	task.stakePool = stakePool
	task.mintAuthority = mintAuthority
	task.feePayerAccount = feePayerAccount

	task.client = client.NewClient(task.cfg.EndpointList)

	task.appendHandlers(task.EraNew, task.EraBond, task.EraUnbond, task.EraUpdataActive, task.EraUpdataRate, task.EraMerge, task.EraWithdraw)
	SafeGoWithRestart(task.handler)
	return nil
}

func (s *Task) appendHandlers(handlers ...func() error) {
	for _, handler := range handlers {

		funcNameRaw := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()

		splits := strings.Split(funcNameRaw, "/")
		funcName := splits[len(splits)-1]

		s.handlers = append(s.handlers, Handler{
			method: handler,
			name:   funcName,
		})
	}
}

func (task *Task) Stop() {
	close(task.stop)
}

func (s *Task) handler() {
	logrus.Info("start handlers")
	retry := 0

Out:
	for {
		if retry > 200 {
			utils.ShutdownRequestChannel <- struct{}{}
			return
		}

		select {
		case <-s.stop:
			logrus.Info("task has stopped")
			return
		default:

			for _, handler := range s.handlers {
				funcName := handler.name
				logrus.Debugf("handler %s start...", funcName)

				err := handler.method()
				if err != nil {
					logrus.Warnf("handler %s failed: %s, will retry.", funcName, err)
					time.Sleep(time.Second * 6)
					retry++
					continue Out
				}
				logrus.Debugf("handler %s end", funcName)
			}

			retry = 0
		}

		time.Sleep(12 * time.Second)
	}
}

func isEmpty(data *rsolprog.EraProcessData) bool {
	return data.NeedBond == 0 && data.NeedUnbond == 0 && data.NewActive == 0 && data.OldActive == 0 && len(data.PendingStakeAccounts) == 0
}
func needBond(data *rsolprog.EraProcessData) bool {
	return data.NeedBond > 0
}

func needUnbond(data *rsolprog.EraProcessData) bool {
	return data.NeedUnbond > 0
}

func needUpdataActive(data *rsolprog.EraProcessData) bool {
	return data.NeedUnbond == 0 && data.NeedBond == 0 && len(data.PendingStakeAccounts) > 0
}

func needUpdataRate(data *rsolprog.EraProcessData) bool {
	return data.NeedUnbond == 0 && data.NeedBond == 0 && len(data.PendingStakeAccounts) == 0 && data.NewActive != 0 && data.OldActive != 0
}
