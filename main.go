package main

import (
	"database/sql"
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

	_ "github.com/lib/pq"
)

var slackBotId string
var slackBotToken string
var tokenAddress string
var infuraAccessToken string
var ropstenKeyJson string
var ropstenPassword string

var cmdRegex = regexp.MustCompile("^<@[^>]+> ([^<]+) (?:<@)?([^ <>]+)(?:>)?")

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
				handleMessage(api, ev)
			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())
			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop
			case *slack.ReactionAddedEvent:
				handleReaction(api, ev)
			default:
				// Ignore unknown errors because it's emitted too much time
			}
		}
	}
}

func handleMessage(api *slack.Client, ev *slack.MessageEvent) {
	if !strings.HasPrefix(ev.Text, "<@"+slackBotId+">") {
		return
	}

	matched := cmdRegex.FindStringSubmatch(ev.Text)
	fmt.Println(matched)
	switch matched[1] {
	case "tip":
		handleTipCommand(api, matched[2])
	case "register":
		handleRegister(api, ev, matched[2])
	default:
		fmt.Printf("Unknown command")
	}
}

func handleReaction(api *slack.Client, ev *slack.ReactionAddedEvent) {
	if ev.Reaction != "hi-ether" {
		return
	}

	sendTokenTo(api, ev.ItemUser)
}

func handleTipCommand(api *slack.Client, userId string) {
	sendTokenTo(api, userId)
}

func handleRegister(api *slack.Client, ev *slack.MessageEvent, address string) {
	userId := ev.User

	db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO accounts(slack_user_id, ethereum_address) VALUES ($1, $2)
		ON CONFLICT ON CONSTRAINT accounts_slack_user_id_key
		DO UPDATE SET ethereum_address=$2;
	`, userId, address)

	if err != nil {
		sendSlackMessage(api, ev.Channel, ":x: "+err.Error())
	} else {
		sendSlackMessage(api, ev.Channel, ":o: Registered `"+address+"`")
	}
}

func sendTokenTo(api *slack.Client, userId string) {
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

func sendSlackMessage(api *slack.Client, channel, message string) {
	_, _, err := api.PostMessage(channel, message, slack.PostMessageParameters{})
	if err != nil {
		fmt.Println(err)
	}
}

func retrieveAddressFor(userId string) (address string) {
	db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	defer db.Close()

	err := db.QueryRow(`
		SELECT ethereum_address FROM accounts WHERE slack_user_id = $1 LIMIT 1;
	`, userId).Scan(&address)
	if err != nil {
		log.Fatal(err)
	}

	return
}
