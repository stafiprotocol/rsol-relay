package task

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/common"
	"github.com/stafiprotocol/solana-go-sdk/rsolprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
)

func (task *Task) EraWithdraw() error {
	stakeManager, err := task.client.GetStakeManager(context.Background(), task.cfg.StakeManagerAddress)
	if err != nil {
		return err
	}

	if !needBond(&stakeManager.EraProcessData) {
		return nil
	}

	res, err := task.client.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
		Commitment: client.CommitmentConfirmed,
	})
	if err != nil {
		fmt.Printf("get recent block hash error, err: %v\n", err)
	}

	epochInfo, err := task.client.GetEpochInfo(context.Background(), client.CommitmentFinalized)
	if err != nil {
		return err
	}

	couldWithdrawAccount := make([]common.PublicKey, 0)
	for _, account := range stakeManager.SplitAccounts {
		stakeAccount, err := task.client.GetStakeAccountInfo(context.Background(), account.ToBase58())
		if err != nil {
			return err
		}
		if stakeAccount.StakeAccount.Info.Stake.Delegation.DeactivationEpoch <= int64(epochInfo.Epoch) {
			couldWithdrawAccount = append(couldWithdrawAccount, account)
		}
	}

	for _, stakeAccount := range couldWithdrawAccount {
		rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
			Instructions: []types.Instruction{
				rsolprog.EraWithdraw(
					task.rSolProgramID,
					task.stakeManager,
					task.stakePool,
					stakeAccount,
				),
			},
			Signers:         []types.Account{task.feePayerAccount},
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

		logrus.Infof("EraWithdraw send tx hash: %s", txHash)
	}
	return nil
}
