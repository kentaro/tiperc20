package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/nlopes/slack"
)

var slackBotId string
var slackBotToken string
var tokenAddress string
var infuraAccessToken string
var ropstenKeyJson string
var ropstenPassword string

var cmdRegex = regexp.MustCompile("^<@[^>]+> ([^<]+) (?:<@)?([^ <>]+)(?:>)?")
var accounts = make(map[string]string)

func init() {
	slackBotToken = os.Getenv("SLACK_BOT_TOKEN")
	tokenAddress = os.Getenv("TIPERC20_TOKEN_ADDRESS")
	infuraAccessToken = os.Getenv("INFURA_ACCESS_TOKEN")
	ropstenKeyJson = os.Getenv("ROPSTEN_KEY_JSON")
	ropstenPassword = os.Getenv("ROPSTEN_PASSWORD")
}

func main() {
	api := slack.New(slackBotToken)
	rtm := api.NewRTM()
	go rtm.ManageConnection()

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				slackBotId = ev.Info.User.ID
			case *slack.MessageEvent:
				handleMessage(ev)
			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())
			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop
			default:
				fmt.Printf("Unknown error")
			}
		}
	}
}

func handleMessage(ev *slack.MessageEvent) {
	if !strings.HasPrefix(ev.Text, "<@"+slackBotId+">") {
		return
	}

	matched := cmdRegex.FindStringSubmatch(ev.Text)
	fmt.Println(matched)
	switch matched[1] {
	case "tip":
		handleTipCommand(matched[2])
	case "register":
		handleRegister(ev.User, matched[2])
	default:
		fmt.Printf("Unknown command")
	}
}

func handleTipCommand(userId string) {
	conn, err := ethclient.Dial("https://ropsten.infura.io/" + infuraAccessToken)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	token, err := NewToken(common.HexToAddress(tokenAddress), conn)
	if err != nil {
		log.Fatalf("Failed to instantiate a Token contract: %v", err)
	}

	auth, err := bind.NewTransactor(strings.NewReader(ropstenKeyJson), ropstenPassword)
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}

	toAddress := retrieveAddressFor(userId)
	if toAddress != "" {
		tx, err := token.Transfer(auth, common.HexToAddress(toAddress), big.NewInt(1000000000000000000))
		if err != nil {
			log.Fatalf("Failed to request token transfer: %v", err)
		}
		fmt.Printf("Transfer pending: 0x%x\n", tx.Hash())
	}
}

func handleRegister(userId string, address string) {
	accounts[userId] = address
}

func retrieveAddressFor(userId string) (address string) {
	address = accounts[userId]
	return
}
