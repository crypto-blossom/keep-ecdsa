package cmd

import (
	"encoding/hex"
	"fmt"

	"github.com/keep-network/keep-tecdsa/internal/config"
	"github.com/keep-network/keep-tecdsa/pkg/btc"
	"github.com/keep-network/keep-tecdsa/pkg/btc/chain"
	"github.com/keep-network/keep-tecdsa/pkg/chain/btc/blockcypher"
	"github.com/keep-network/keep-tecdsa/pkg/chain/btc/electrum"
	"github.com/urfave/cli"
)

// PublishCommand contains the definition of the publish command-line subcommand.
var PublishCommand cli.Command

const publishDescription = `The publish command submits a transaction using the 
specific external service.`

func init() {
	PublishCommand = cli.Command{
		Name:        "publish",
		Usage:       "Publish a transaction",
		Description: publishDescription,
		Action:      Publish,
	}
}

// Publish sends a raw transaction provided as a CLI argument.
func Publish(c *cli.Context) error {
	arg := c.Args().First()

	configFile, err := config.ReadConfig(c.GlobalString("config"))
	if err != nil {
		return err
	}

	var btcChain chain.Interface

	switch chainFlag := c.GlobalString("broadcast-api"); chainFlag {
	case "blockcypher":
		btcChain, err = blockcypher.Connect(&configFile.BlockCypher)
		if err != nil {
			return err
		}
	case "electrum":
		btcChain, err = electrum.Connect(&configFile.Electrum)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown transaction publication service")
	}

	rawTx, err := hex.DecodeString(arg)
	if err != nil {
		return fmt.Errorf("transaction decoding failed [%s]", err)
	}

	result, err := btc.Publish(btcChain, rawTx)
	if err != nil {
		return fmt.Errorf("publish failed [%s]", err)
	}

	fmt.Printf("Published transaction ID: %v\n", result)

	return nil
}
