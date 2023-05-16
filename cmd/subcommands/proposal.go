package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
)

var (
	newOnlyProposals = false
	proposalList     []string
)

func proposalSub() []*cobra.Command {
	cmdProposalList := &cobra.Command{
		Use:   "list",
		Short: "List network proposals",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			list, err := conn.ProposalsList()
			if err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(list.Proposals)
				return nil
			}

			result := make(map[string]interface{})

			pList := make([]map[string]interface{}, 0)
			for _, proposal := range list.Proposals {
				approvals := make([]string, len(proposal.Approvals))
				for i, a := range proposal.Approvals {
					approvals[i] = address.Address(a).String()
				}
				expired := false
				expiration := time.Unix(proposal.ExpirationTime/1000, 0)
				if expiration.Before(time.Now()) {
					expired = true
					if newOnlyProposals && expired {
						continue
					}
				}

				data := map[string]interface{}{
					"ID":             proposal.ProposalId,
					"Proposer":       address.Address(proposal.ProposerAddress).String(),
					"CreateTime":     time.Unix(proposal.CreateTime/1000, 0),
					"ExpirationTime": expiration,
					"Expired":        expired,
					"Parameters":     proposal.Parameters,
					"Approvals":      approvals,
				}
				pList = append([]map[string]interface{}{data}, pList...)
			}
			result["totalCount"] = len(list.Proposals)
			result["filterCount"] = len(pList)
			result["proposals"] = pList
			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}
	cmdProposalList.Flags().BoolVar(&newOnlyProposals, "new", false, "Show only new proposals")

	cmdProposalApprove := &cobra.Command{
		Use:   "approve",
		Short: "Approve network proposal",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}
			confirm, err := strconv.ParseBool(args[1])
			if err != nil {
				return err
			}

			tx, err := conn.ProposalApprove(signerAddress.String(), id, confirm)
			if err != nil {
				return err
			}
			var ctrlr *transaction.Controller
			if useLedgerWallet {
				account := keystore.Account{Address: signerAddress.GetAddress()}
				ctrlr = transaction.NewController(conn, nil, &account, tx.Transaction, opts)
			} else {
				ks, acct, err := store.UnlockedKeystore(signerAddress.String(), passphrase)
				if err != nil {
					return err
				}
				ctrlr = transaction.NewController(conn, ks, acct, tx.Transaction, opts)
			}
			if err = ctrlr.ExecuteTransaction(); err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(tx, ctrlr.Receipt, ctrlr.Result)
				return nil
			}

			result := make(map[string]interface{})
			result["from"] = signerAddress.String()
			result["txID"] = common.BytesToHexString(tx.GetTxid())
			result["blockNumber"] = ctrlr.Receipt.BlockNumber
			result["message"] = string(ctrlr.Result.Message)
			result["receipt"] = map[string]interface{}{
				"fee":      ctrlr.Receipt.Fee,
				"netFee":   ctrlr.Receipt.Receipt.NetFee,
				"netUsage": ctrlr.Receipt.Receipt.NetUsage,
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdProposalWithdraw := &cobra.Command{
		Use:   "withdraw",
		Short: "Withdraw network proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}

			tx, err := conn.ProposalWithdraw(signerAddress.String(), id)
			if err != nil {
				return err
			}
			var ctrlr *transaction.Controller
			if useLedgerWallet {
				account := keystore.Account{Address: signerAddress.GetAddress()}
				ctrlr = transaction.NewController(conn, nil, &account, tx.Transaction, opts)
			} else {
				ks, acct, err := store.UnlockedKeystore(signerAddress.String(), passphrase)
				if err != nil {
					return err
				}
				ctrlr = transaction.NewController(conn, ks, acct, tx.Transaction, opts)
			}
			if err = ctrlr.ExecuteTransaction(); err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(tx, ctrlr.Receipt, ctrlr.Result)
				return nil
			}

			result := make(map[string]interface{})
			result["from"] = signerAddress.String()
			result["txID"] = common.BytesToHexString(tx.GetTxid())
			result["blockNumber"] = ctrlr.Receipt.BlockNumber
			result["message"] = string(ctrlr.Result.Message)
			result["receipt"] = map[string]interface{}{
				"fee":      ctrlr.Receipt.Fee,
				"netFee":   ctrlr.Receipt.Receipt.NetFee,
				"netUsage": ctrlr.Receipt.Receipt.NetUsage,
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdProposalCreate := &cobra.Command{
		Use:   "create",
		Short: "Approve network proposal",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			proposals := make(map[int64]int64)
			for _, proposal := range proposalList {
				proposalKeyValue := strings.Split(proposal, ":")
				if len(proposalKeyValue) != 2 {
					return fmt.Errorf("invalid proposal %s", proposalKeyValue)
				}
				paramID, err := strconv.ParseInt(proposalKeyValue[0], 10, 64)
				if err != nil {
					return fmt.Errorf("invalid param ID: %s %+v", proposalKeyValue[0], err)
				}

				if proposals[paramID] > 0 {
					return fmt.Errorf("proposal colision %d:%d -> %s", paramID, proposals[paramID], proposal)
				}
				// check proposal value
				proposalValue, err := strconv.ParseInt(proposalKeyValue[1], 10, 64)
				if err != nil {
					return fmt.Errorf("invalid vote count %s. %+v", proposalKeyValue[1], err)
				}
				proposals[paramID] = proposalValue
			}

			tx, err := conn.ProposalCreate(signerAddress.String(), proposals)
			if err != nil {
				return err
			}

			var ctrlr *transaction.Controller
			if useLedgerWallet {
				account := keystore.Account{Address: signerAddress.GetAddress()}
				ctrlr = transaction.NewController(conn, nil, &account, tx.Transaction, opts)
			} else {
				ks, acct, err := store.UnlockedKeystore(signerAddress.String(), passphrase)
				if err != nil {
					return err
				}
				ctrlr = transaction.NewController(conn, ks, acct, tx.Transaction, opts)
			}
			if err = ctrlr.ExecuteTransaction(); err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(tx, ctrlr.Receipt, ctrlr.Result)
				return nil
			}

			result := make(map[string]interface{})
			result["from"] = signerAddress.String()
			result["txID"] = common.BytesToHexString(tx.GetTxid())
			result["blockNumber"] = ctrlr.Receipt.BlockNumber
			result["message"] = string(ctrlr.Result.Message)
			result["receipt"] = map[string]interface{}{
				"fee":      ctrlr.Receipt.Fee,
				"netFee":   ctrlr.Receipt.Receipt.NetFee,
				"netUsage": ctrlr.Receipt.Receipt.NetUsage,
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdProposalCreate.Flags().StringSliceVar(&proposalList, "params", []string{}, "ID:VALUE,ID:VALUE")

	return []*cobra.Command{cmdProposalList, cmdProposalApprove, cmdProposalWithdraw, cmdProposalCreate}
}

func init() {
	cmdProposal := &cobra.Command{
		Use:   "proposal",
		Short: "Network upgrade proposal",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdProposal.AddCommand(proposalSub()...)
	RootCmd.AddCommand(cmdProposal)
}
