package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/stafiprotocol/rsol-relay/pkg/config"
	"github.com/stafiprotocol/rsol-relay/pkg/vault"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/common"
	"github.com/stafiprotocol/solana-go-sdk/rsolprog"
	"github.com/stafiprotocol/solana-go-sdk/sysprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
)

var stakePoolSeed = []byte("pool_seed")

func rsolInitCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "rsol-init",
		Short: "Init rsol",

		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(flagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg, err := config.LoadInitConfig(configPath)
			if err != nil {
				return err
			}
			v, err := vault.NewVaultFromWalletFile(cfg.KeystorePath)
			if err != nil {
				return err
			}
			boxer, err := vault.SecretBoxerForType(v.SecretBoxWrap)
			if err != nil {
				return fmt.Errorf("secret boxer: %w", err)
			}

			if err := v.Open(boxer); err != nil {
				return fmt.Errorf("opening: %w", err)
			}

			privateKeyMap := make(map[string]vault.PrivateKey)
			accountMap := make(map[string]types.Account)
			for _, privKey := range v.KeyBag {
				privateKeyMap[privKey.PublicKey().String()] = privKey
				accountMap[privKey.PublicKey().String()] = types.AccountFromPrivateKeyBytes(privKey)
			}

			c := client.NewClient(cfg.EndpointList)

			res, err := c.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
				Commitment: client.CommitmentConfirmed,
			})
			if err != nil {
				fmt.Printf("get recent block hash error, err: %v\n", err)
			}

			rSolMint := common.PublicKeyFromString(cfg.RSolMintAddress)
			rSolProgramID := common.PublicKeyFromString(cfg.RSolProgramID)
			feeRecipient := common.PublicKeyFromString(cfg.FeeRecipientAddress)
			validator := common.PublicKeyFromString(cfg.ValidatorAddress)

			feePayerAccount, exist := accountMap[cfg.FeePayerAccount]
			if !exist {
				return fmt.Errorf("fee payer not exit in vault")
			}
			adminAccount, exist := accountMap[cfg.AdminAccount]
			if !exist {
				return fmt.Errorf("admin not exit in vault")
			}
			stakeManagerAccount, exist := accountMap[cfg.StakeManagerAccount]
			if !exist {
				return fmt.Errorf("stakeManager not exit in vault")
			}

			stakePool, _, err := common.FindProgramAddress([][]byte{stakeManagerAccount.PublicKey.Bytes(), stakePoolSeed}, rSolProgramID)
			if err != nil {
				return err
			}

			stakePoolRent, err := c.GetMinimumBalanceForRentExemption(context.Background(), 0)
			if err != nil {
				return err
			}

			stakeManagerRent, err := c.GetMinimumBalanceForRentExemption(context.Background(), rsolprog.StakeManagerAccountLengthDefault)
			if err != nil {
				return err
			}

			fmt.Println("stakeManager account:", stakeManagerAccount.PublicKey.ToBase58())
			fmt.Println("stakePool account:", stakePool.ToBase58())
			fmt.Println("admin", adminAccount.PublicKey.ToBase58())
			fmt.Println("feePayer:", feePayerAccount.PublicKey.ToBase58())
			fmt.Println("stake pool rent:", stakePoolRent)
			fmt.Println("stake manager rent:", stakeManagerRent)
		Out:
			for {
				fmt.Println("\ncheck account info, then press (y/n) to continue:")
				var input string
				fmt.Scanln(&input)
				switch input {
				case "y":
					break Out
				case "n":
					return nil
				default:
					fmt.Println("press `y` or `n`")
					continue
				}
			}

			rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
				Instructions: []types.Instruction{
					sysprog.Transfer(
						feePayerAccount.PublicKey,
						stakePool,
						stakePoolRent,
					),
					sysprog.CreateAccount(
						feePayerAccount.PublicKey,
						stakeManagerAccount.PublicKey,
						rSolProgramID,
						stakeManagerRent,
						rsolprog.StakeManagerAccountLengthDefault,
					),
					rsolprog.Initialize(
						rSolProgramID,
						stakeManagerAccount.PublicKey,
						stakePool,
						feeRecipient,
						rSolMint,
						adminAccount.PublicKey,
						rsolprog.InitializeData{
							RSolMint:         rSolMint,
							Validator:        validator,
							Bond:             cfg.Bond,
							Unbond:           cfg.Unbond,
							Active:           cfg.Active,
							LatestEra:        cfg.LatestEra,
							Rate:             cfg.Rate,
							TotalRSolSupply:  cfg.TotalRSolSupply,
							TotalProtocolFee: cfg.TotalProtocolFee,
						},
					),
				},
				Signers:         []types.Account{feePayerAccount, stakeManagerAccount, adminAccount},
				FeePayer:        feePayerAccount.PublicKey,
				RecentBlockHash: res.Blockhash,
			})
			if err != nil {
				fmt.Printf("generate tx error, err: %v\n", err)
			}
			txHash, err := c.SendRawTransaction(context.Background(), rawTx)
			if err != nil {
				fmt.Printf("send tx error, err: %v\n", err)
			}

			fmt.Println("createStakeManager txHash:", txHash)

			retry := 0
			for {
				if retry > 60 {
					return fmt.Errorf("tx %s failed", txHash)
				}
				_, err := c.GetAccountInfo(context.Background(), cfg.StakeManagerAccount, client.GetAccountInfoConfig{
					Encoding:  client.GetAccountInfoConfigEncodingBase64,
					DataSlice: client.GetAccountInfoConfigDataSlice{},
				})
				if err != nil {
					retry++
					time.Sleep(time.Second)
					continue
				}

				break
			}

			return nil
		},
	}
	cmd.Flags().String(flagConfigPath, defaultConfigPath, "Config file path")
	return cmd
}
