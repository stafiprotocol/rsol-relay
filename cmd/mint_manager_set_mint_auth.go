package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stafiprotocol/rsol-relay/pkg/config"
	"github.com/stafiprotocol/rsol-relay/pkg/vault"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/common"
	"github.com/stafiprotocol/solana-go-sdk/minterprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
)

func mintManagerSetMintAuth() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "set-mint-auth",
		Short: "Set ext mint authorities",

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

			mintManagerProgramID := common.PublicKeyFromString(cfg.MintManagerProgramID)

			feePayerAccount, exist := accountMap[cfg.FeePayerAccount]
			if !exist {
				return fmt.Errorf("fee payer not exit in vault")
			}
			adminAccount, exist := accountMap[cfg.AdminAccount]
			if !exist {
				return fmt.Errorf("admin not exit in vault")
			}
			mintManagerAccount, exist := accountMap[cfg.MintManagerAccount]
			if !exist {
				return fmt.Errorf("mintManager not exit in vault")
			}

			newMintAuthPubkeys := make([]common.PublicKey, 0)
			for _, auth := range cfg.MintAuthorities {
				newMintAuthPubkeys = append(newMintAuthPubkeys, common.PublicKeyFromString(auth))
			}

			fmt.Println("mintManager account:", mintManagerAccount.PublicKey.ToBase58())
			fmt.Println("admin", adminAccount.PublicKey.ToBase58())
			fmt.Println("feePayer:", feePayerAccount.PublicKey.ToBase58())
			fmt.Println("mint authorities:", newMintAuthPubkeys)
		Out:
			for {
				fmt.Println("\ncheck config info, then press (y/n) to continue:")
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
					minterprog.SetExtMintAuthorities(
						mintManagerProgramID,
						mintManagerAccount.PublicKey,
						adminAccount.PublicKey,
						newMintAuthPubkeys,
					),
				},
				Signers:         []types.Account{feePayerAccount, adminAccount},
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

			fmt.Println("SetExtMintAuthorities txHash:", txHash)

			return nil
		},
	}
	cmd.Flags().String(flagConfigPath, defaultConfigPath, "Config file path")
	return cmd
}
