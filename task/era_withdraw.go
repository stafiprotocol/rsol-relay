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

	couldWithdrawAccount := make([]common.PublicKey, 0)
	for _, account := range stakeManager.SplitAccounts {
		accountInfo, err := task.client.CalStakeActivation(
			context.Background(),
			account.ToBase58())
		if err != nil {
			return err
		}
		if accountInfo.State == client.StakeActivationStateInactive {
			couldWithdrawAccount = append(couldWithdrawAccount, account)
		}
	}

	if len(couldWithdrawAccount) == 0 {
		return nil
	}

	for _, stakeAccount := range couldWithdrawAccount {
		stakeAccountInfo, err := task.client.GetStakeAccountInfo(context.Background(), stakeAccount.ToBase58())
		if err != nil {
			return err
		}

		res, err := task.client.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
			Commitment: client.CommitmentConfirmed,
		})
		if err != nil {
			fmt.Printf("get recent block hash error, err: %v\n", err)
		}

		rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
			Instructions: []types.Instruction{
				rsolprog.EraWithdraw(
					task.stakeManagerProgramID,
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

		logrus.Infof("EraWithdraw send tx hash: %s, stakeAccount: %s, withdrawAmount: %d",
			txHash, stakeAccount.ToBase58(), stakeAccountInfo.Lamports)

		if err := task.waitTx(txHash); err != nil {
			_, err := task.client.GetStakeAccountInfo(context.Background(), stakeAccount.ToBase58())
			if err != nil && err == client.ErrAccountNotFound {
				logrus.Info("EraWithdraw success")
				return nil
			}

			return err
		}

		logrus.Info("EraWithdraw success")
	}
	return nil
}
