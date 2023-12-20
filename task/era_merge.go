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

func (task *Task) EraMerge() error {
	stakeManager, err := task.client.GetStakeManager(context.Background(), task.cfg.StakeManagerAddress)
	if err != nil {
		return err
	}

	if !isEmpty(&stakeManager.EraProcessData) {
		return nil
	}

	res, err := task.client.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
		Commitment: client.CommitmentConfirmed,
	})
	if err != nil {
		fmt.Printf("get recent block hash error, err: %v\n", err)
	}

	valToAccount := make(map[string]map[uint64][]common.PublicKey) // voter -> credit -> []stakeAccount
	for _, stakeAccount := range stakeManager.StakeAccounts {
		accountInfo, err := task.client.GetStakeActivation(
			context.Background(),
			stakeAccount.ToBase58(),
			client.GetStakeActivationConfig{})
		if err != nil {
			return err
		}
		if accountInfo.State != client.StakeActivationStateActive {
			continue
		}

		account, err := task.client.GetStakeAccountInfo(context.Background(), stakeAccount.ToBase58())
		if err != nil {
			return err
		}
		voter := account.StakeAccount.Info.Stake.Delegation.Voter.ToBase58()
		credit := account.StakeAccount.Info.Stake.CreditsObserved
		if valToAccount[voter] == nil {
			valToAccount[voter] = make(map[uint64][]common.PublicKey)
		}
		if valToAccount[voter][credit] == nil {
			valToAccount[voter][credit] = make([]common.PublicKey, 0)
		}

		valToAccount[voter][credit] = append(valToAccount[voter][credit], stakeAccount)
	}

	for _, creditToAccounts := range valToAccount {
		for _, accounts := range creditToAccounts {
			if len(accounts) >= 2 {
				srcStakeAccount := accounts[0]
				dstStakeAccount := accounts[1]
				rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
					Instructions: []types.Instruction{
						rsolprog.EraMerge(
							task.stakeManagerProgramID,
							task.stakeManager,
							srcStakeAccount,
							dstStakeAccount,
							task.stakePool,
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

				logrus.Infof("EraMerge send tx hash: %s", txHash)
				if err := task.waitTx(txHash); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
