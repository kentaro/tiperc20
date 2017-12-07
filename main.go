package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/nlopes/slack"

	_ "github.com/lib/pq"
)

var slackBotId string
var slackBotToken string
var slackTipReaction string
var slackTipAmount string
var tokenAddress string
var ethApiEndpoint string
var ethKeyJson string
var ethPassword string

var httpdPort int

var cmdRegex = regexp.MustCompile("^<@[^>]+> ([^<]+) (?:<@)?([^ <>]+)(?:>)?")

func init() {
	slackBotToken = os.Getenv("SLACK_BOT_TOKEN")
	slackTipReaction = os.Getenv("SLACK_TIP_REACTION")
	slackTipAmount = os.Getenv("SLACK_TIP_AMOUNT")
	tokenAddress = os.Getenv("ERC20_TOKEN_ADDRESS")
	ethApiEndpoint = os.Getenv("ETH_API_ENDPOINT")
	ethKeyJson = os.Getenv("ETH_KEY_JSON")
	ethPassword = os.Getenv("ETH_PASSWORD")

	flag.IntVar(&httpdPort, "port", 20020, "port number")
}

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "TipERC20: https://github.com/kentaro/tiperc20")
	})
	log.Println(httpdPort)
	go log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", httpdPort), nil))

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
		handleTipCommand(api, ev, matched[2])
	case "register":
		handleRegister(api, ev, matched[2])
	default:
		fmt.Printf("Unknown command")
	}
}

func handleReaction(api *slack.Client, ev *slack.ReactionAddedEvent) {
	if ev.Reaction != slackTipReaction {
		return
	}

	address := retrieveAddressFor(ev.ItemUser)
	if address == "" {
		sendSlackMessage(api, ev.ItemUser, `
:question: Please register your Ethereum address:

> @tiperc20 register YOUR_ADDRESS
		`)
	} else {
		tx, err := sendTokenTo(address)
		if err == nil {
			sendSlackMessage(api, ev.ItemUser, ":+1: You got a token at "+tx.Hash().String())
		}
	}
}

func handleTipCommand(api *slack.Client, ev *slack.MessageEvent, userID string) {
	address := retrieveAddressFor(userID)

	if address == "" {
		sendSlackMessage(api, userID, `
:question: Please register your Ethereum address:

> @tiperc20 register YOUR_ADDRESS
		`)
	} else {
		tx, err := sendTokenTo(address)
		if err != nil {
			sendSlackMessage(api, ev.Channel, ":x: "+err.Error())
		} else {
			sendSlackMessage(api, userID, ":+1: You got a token at "+tx.Hash().String())
		}
	}
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

func sendTokenTo(address string) (tx *types.Transaction, err error) {
	conn, err := ethclient.Dial(ethApiEndpoint)
	if err != nil {
		log.Printf("Failed to instantiate a Token contract: %v", err)
		return
	}

	token, err := NewToken(common.HexToAddress(tokenAddress), conn)
	if err != nil {
		log.Printf("Failed to instantiate a Token contract: %v", err)
		return
	}

	auth, err := bind.NewTransactor(strings.NewReader(ethKeyJson), ethPassword)
	if err != nil {
		log.Printf("Failed to create authorized transactor: %v", err)
		return
	}

	amount, err := strconv.ParseInt(slackTipAmount, 10, 64)
	if err != nil {
		log.Printf("Invalid tip amount: %v", err)
		return
	}

	tx, err = token.Transfer(auth, common.HexToAddress(address), big.NewInt(amount))
	if err != nil {
		log.Printf("Failed to request token transfer: %v", err)
		return
	}

	log.Printf("Transfer pending: 0x%x\n", tx.Hash())
	return
}

func sendSlackMessage(api *slack.Client, channel, message string) {
	_, _, err := api.PostMessage(channel, message, slack.PostMessageParameters{})
	if err != nil {
		log.Println(err)
	}
}

func retrieveAddressFor(userID string) (address string) {
	db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	defer db.Close()

	db.QueryRow(`
		SELECT ethereum_address FROM accounts WHERE slack_user_id = $1 LIMIT 1;
	`, userID).Scan(&address)

	return
}
