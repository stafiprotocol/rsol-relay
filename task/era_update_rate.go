package task

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/rsolprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
)

func (task *Task) EraUpdataRate() error {
	stakeManager, err := task.client.GetStakeManager(context.Background(), task.cfg.StakeManagerAddress)
	if err != nil {
		return err
	}

	if !needUpdataRate(&stakeManager.EraProcessData) {
		return nil
	}

	res, err := task.client.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
		Commitment: client.CommitmentConfirmed,
	})
	if err != nil {
		fmt.Printf("get recent block hash error, err: %v\n", err)
	}

	rSolMint := stakeManager.RSolMint

	rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
		Instructions: []types.Instruction{
			rsolprog.EraUpdateRate(
				task.rSolProgramID,
				task.stakeManager,
				task.stakePool,
				task.mintManager,
				rSolMint,
				task.feeRecipient,
				task.mintAuthority,
				task.minterProgramID,
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

	logrus.Infof("EraUpdateRate send tx hash: %s", txHash)
	return nil
}