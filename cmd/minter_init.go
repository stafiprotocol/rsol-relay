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
	"github.com/stafiprotocol/solana-go-sdk/minterprog"
	"github.com/stafiprotocol/solana-go-sdk/sysprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
)

var mintAuthoritySeed = []byte("mint")

func minterInitCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "minter-init",
		Short: "Init minter",

		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(flagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg, err := config.Load(configPath)
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
			minterProgramID := common.PublicKeyFromString(cfg.MinterProgramID)
			rSolProgramID := common.PublicKeyFromString(cfg.RSolProgramID)
			bridgeSigner := common.PublicKeyFromString(cfg.BridgeSignerAddress)

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
			minterManagerAccount, exist := accountMap[cfg.MinterManagerAccount]
			if !exist {
				return fmt.Errorf("minterStakeManager not exit in vault")
			}

			stakePool, _, err := common.FindProgramAddress([][]byte{stakeManagerAccount.PublicKey.Bytes(), stakePoolSeed}, rSolProgramID)
			if err != nil {
				return err
			}

			minterManagerRent, err := c.GetMinimumBalanceForRentExemption(context.Background(), minterprog.MinterManagerAccountLengthDefault)
			if err != nil {
				return err
			}

			extMintAthorities := []common.PublicKey{stakePool, bridgeSigner}

			mintAuthority, _, err := common.FindProgramAddress([][]byte{minterManagerAccount.PublicKey.Bytes(), mintAuthoritySeed}, minterProgramID)
			if err != nil {
				return err
			}

			fmt.Printf("\nconfig:\n %+v", cfg)
		Out:
			for {
				fmt.Println("\ncheck config again, then press (y/n) to continue:")
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
					sysprog.CreateAccount(
						feePayerAccount.PublicKey,
						minterManagerAccount.PublicKey,
						minterProgramID,
						minterManagerRent,
						minterprog.MinterManagerAccountLengthDefault,
					),
					minterprog.Initialize(
						minterProgramID,
						minterManagerAccount.PublicKey,
						mintAuthority,
						rSolMint,
						adminAccount.PublicKey,
						extMintAthorities,
					),
				},
				Signers:         []types.Account{feePayerAccount, minterManagerAccount, adminAccount},
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

			fmt.Println("createMinterManager txHash:", txHash)
			retry := 0
			for {
				if retry > 60 {
					return fmt.Errorf("tx %s failed", txHash)
				}
				_, err := c.GetAccountInfo(context.Background(), minterManagerAccount.PublicKey.ToBase58(), client.GetAccountInfoConfig{
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
			fmt.Println("minterManager account:", minterManagerAccount.PublicKey.ToBase58())
			fmt.Println("admin", adminAccount.PublicKey.ToBase58())
			fmt.Println("mintAuthority", mintAuthority.ToBase58())
			fmt.Println("stakepool", stakePool.ToBase58())
			fmt.Println("feePayer:", feePayerAccount.PublicKey.ToBase58())

			return nil
		},
	}
	cmd.Flags().String(flagConfigPath, defaultConfigPath, "Config file path")
	return cmd
}
