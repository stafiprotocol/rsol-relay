package task

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/rsolprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
)

func (task *Task) EraUnbond() error {
	stakeManager, err := task.client.GetStakeManager(context.Background(), task.cfg.StakeManagerAddress)
	if err != nil {
		return err
	}

	if !needUnbond(&stakeManager.EraProcessData) {
		return nil
	}

	res, err := task.client.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
		Commitment: client.CommitmentConfirmed,
	})
	if err != nil {
		fmt.Printf("get recent block hash error, err: %v\n", err)
	}

	stakeAccount := stakeManager.StakeAccounts[0]
	stakeAccountInfo, err := task.client.GetStakeAccountInfo(context.Background(), stakeAccount.ToBase58())
	if err != nil {
		return err
	}
	validator := stakeAccountInfo.StakeAccount.Info.Stake.Delegation.Voter

	splitStakeAccount := types.NewAccount() //random account

	rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
		Instructions: []types.Instruction{
			rsolprog.EraUnbond(
				task.stakeManagerProgramID,
				task.stakeManager,
				task.stakePool,
				stakeAccount,
				splitStakeAccount.PublicKey,
				validator,
				task.feePayerAccount.PublicKey,
			),
		},
		Signers:         []types.Account{task.feePayerAccount, splitStakeAccount},
		FeePayer:        task.feePayerAccount.PublicKey,
		RecentBlockHash: res.Blockhash,
	})

	if err != nil {
		fmt.Printf("generate tx error, err: %v\n", err)
	}
	txHash, err := task.client.SendRawTransaction(context.Background(), rawTx)
	if err != nil {
		fmt.Printf("send tx error, err: %v\n", err)
	}

	logrus.Infof("EraUnbond send tx hash: %s", txHash)
	if err := task.waitTx(txHash); err != nil {
		return err
	}
	return nil
}
